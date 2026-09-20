# Implementation plan: resilient broker purchases in Go

This directory is supporting material for the post. It is a runnable, intentionally small reference implementation; the mock broker is the only third-party API.

## Run the example

Start PostgreSQL, the mock broker, the purchase API, and the reconciliation worker:

```bash
docker compose up --build
```

Submit a purchase from another terminal:

```bash
curl --request POST http://localhost:8080/purchases \
  --header 'Content-Type: application/json' \
  --data '{"account_id":"acct_123","symbol":"ACME","quantity":10}'
```

The mock broker randomly produces `201 Created`, `400 Bad Request`, `500 Internal Server Error`, a timeout-style response, a closed connection, or an accepted order whose response is lost. For an ambiguous outcome, the API returns `202 Accepted` and a `transaction_id`. Inspect it with:

```bash
curl http://localhost:8080/transactions/TRANSACTION_ID
```

The PostgreSQL image runs `migrations/001_create_transaction_history.up.sql` only when its named volume is first created. To start over with a fresh database, run `docker compose down --volumes` from this directory.

## Goal and boundary

Build two Go applications:

1. **Purchase API** accepts a buy-stock request, persists a transaction, and makes exactly one synchronous broker call.
2. **Reconciliation worker** receives due unknown transactions, queries or retries the broker, and schedules another attempt when the result remains uncertain.

The domain and application layers must not import `database/sql`, a PostgreSQL driver, or a Kafka client. Those technologies live only in adapters. This keeps the behavior unchanged when PostgreSQL polling is replaced with Kafka delivery.

## Proposed package layout

```text
cmd/
├── purchase-api/main.go        # HTTP server, PostgreSQL adapter, broker HTTP-client wiring
└── reconciliation-worker/main.go # Worker loop and adapter wiring

internal/
├── domain/
│   └── transaction.go          # Transaction states and value objects
├── application/
│   ├── submit_purchase.go      # First-attempt use case
│   └── reconcile_purchase.go   # Worker use case
├── ports/
│   ├── transaction_store.go    # Durable transaction-history port
│   ├── reconciliation_queue.go # Claim, acknowledge, and defer work port
│   ├── broker.go               # Third-party broker port
│   └── clock.go                # Time source for deterministic tests
└── adapters/
    ├── postgres/               # PostgreSQL implementations of store and queue ports
    ├── kafka/                  # Future Kafka queue implementation
    ├── httpapi/                # Request parsing and HTTP response mapping
    ├── mockbroker/             # Random-behavior broker HTTP service
    └── httpbroker/             # Broker HTTP-client adapter used by API and worker
```

## Domain model

Define transaction states in the domain package rather than passing string literals through the application:

```go
type TransactionStatus string

const (
	Pending      TransactionStatus = "PENDING"
	Completed    TransactionStatus = "COMPLETED"
	Rejected     TransactionStatus = "REJECTED"
	Unknown      TransactionStatus = "UNKNOWN"
	ManualReview TransactionStatus = "MANUAL_REVIEW"
)

type Transaction struct {
	ID             uuid.UUID
	Status         TransactionStatus
	IdempotencyKey string
	Purchase       PurchaseRequest
	BrokerOrderID  string
	Attempts       int
	NextRetry      *time.Time // nil until a retry has been scheduled
}
```

Persisting only `id`, `status`, and `next_retry` is sufficient to illustrate queue scheduling, but not to perform a real retry. The production transaction record needs a safe representation of the request, the idempotency key, the broker operation ID when known, the attempt count, and audit timestamps.

## Ports: PostgreSQL is replaceable

Separate durable transaction history from delivery of reconciliation work. Both are interfaces owned by the application layer.

```go
type TransactionStore interface {
	CreatePending(ctx context.Context, purchase PurchaseRequest) (Transaction, error)
	MarkCompleted(ctx context.Context, id uuid.UUID, result BrokerResult) error
	MarkRejected(ctx context.Context, id uuid.UUID, result BrokerResult) error
	MarkUnknown(ctx context.Context, id uuid.UUID, nextRetry time.Time) error
	MarkManualReview(ctx context.Context, id uuid.UUID, reason string) error
	Get(ctx context.Context, id uuid.UUID) (Transaction, error)
}

type ReconciliationQueue interface {
	ClaimDue(ctx context.Context, limit int, now time.Time) ([]Transaction, error)
	Acknowledge(ctx context.Context, id uuid.UUID) error
	Defer(ctx context.Context, id uuid.UUID, nextRetry time.Time) error
}

type Broker interface {
	Submit(ctx context.Context, purchase PurchaseRequest, idempotencyKey string) (BrokerResult, error)
	Lookup(ctx context.Context, idempotencyKey string) (BrokerResult, error)
}
```

The PostgreSQL adapter can implement both persistence and queue behavior against `transaction_history`. `ClaimDue` should atomically select due `UNKNOWN` rows with `FOR UPDATE SKIP LOCKED`, move `next_retry` forward as a short lease, and commit before the broker call.

A Kafka adapter can implement `ReconciliationQueue` by consuming a transaction ID from a topic, acknowledging only terminal outcomes, and publishing a delayed retry message (or using a retry-topic strategy). Kafka does **not** eliminate the need for durable transaction state: `TransactionStore` still records the purchase outcome. Use an outbox transaction when PostgreSQL writes must reliably produce a Kafka event.

## Purchase API flow

1. Validate the incoming symbol, quantity, and account balance. Return `400` before creating a broker transaction if validation fails.
2. Call `TransactionStore.CreatePending` to create the transaction and idempotency key.
3. Call `Broker.Submit` once with a context deadline.
4. If the broker responds with `2xx`, mark `COMPLETED` and return its success response.
5. If the broker responds with `4xx`, mark `REJECTED` and return its client-error response.
6. If the call times out, the connection closes, or the broker returns `5xx`, mark `UNKNOWN`, enqueue or schedule reconciliation, and return `202 Accepted` with the transaction ID.

Never tell the client that a stock purchase failed merely because the HTTP response was lost. The broker might already have accepted it.

## Reconciliation worker flow

1. Repeatedly call `ReconciliationQueue.ClaimDue` with a small batch size.
2. For each transaction, call `Broker.Lookup` using its original idempotency key when the provider supports lookup.
3. If lookup returns a final result, persist `COMPLETED` or `REJECTED`, then acknowledge the queue item.
4. If lookup is still unknown, call `Broker.Submit` with the **same** idempotency key.
5. Persist and acknowledge returned `2xx`/`4xx` outcomes.
6. For another timeout, connection error, or `5xx`, calculate a capped exponential delay with jitter and call `Defer`.
7. After the configured attempt or age limit, mark `MANUAL_REVIEW` and alert instead of retrying forever.

Do not hold a PostgreSQL row lock while calling the broker. Claim a short lease, commit, then perform the network call.

## PostgreSQL adapter plan

Start with a `transaction_history` table with the post's required columns, then add the fields the application needs:

```sql
ALTER TABLE transaction_history
  ADD COLUMN idempotency_key TEXT NOT NULL UNIQUE,
  ADD COLUMN purchase_payload JSONB NOT NULL,
  ADD COLUMN broker_order_id TEXT,
  ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
```

Use parameterized queries and a PostgreSQL connection pool. Define one SQL query for each port method; do not expose SQL or database rows to application services.

## Mock broker with randomized behavior

The mock is a small HTTP service and the API/worker use an HTTP client that implements the `Broker` port. The service injects a `*rand.Rand`, rather than using the package-global generator, so tests can use a fixed seed and reproduce a failure.

```go
type MockBroker struct {
	rng *rand.Rand
}

func (m *MockBroker) Submit(
	ctx context.Context, purchase PurchaseRequest, key string,
) (BrokerResult, error) {
	switch m.rng.Intn(5) {
	case 0:
		return BrokerResult{StatusCode: http.StatusCreated}, nil
	case 1:
		return BrokerResult{StatusCode: http.StatusBadRequest}, nil
	case 2:
		return BrokerResult{StatusCode: http.StatusInternalServerError}, nil
	case 3:
		return BrokerResult{}, context.DeadlineExceeded
	default:
		return BrokerResult{}, io.ErrUnexpectedEOF
	}
}
```

Keep an in-memory map keyed by idempotency key in the mock. When it selects a successful purchase, store the result before returning it. A later call with the same key must return that stored outcome; this makes the mock exercise the real safety property instead of only producing random errors.

Make response weights and latency configurable. In unit tests, pass a seeded generator and explicit weights; in local demos, seed from the current time to vary behavior.

## Delivery milestones

1. Create domain types, ports, and unit tests for status transitions and backoff calculation.
2. Implement the random mock broker and test that repeated idempotency keys never create a second purchase.
3. Implement the purchase API use case with an in-memory store and queue for fast tests.
4. Add the PostgreSQL adapters, migration, and integration tests using a real PostgreSQL instance.
5. Add the worker with row-claim/lease behavior and verify two worker processes do not process the same transaction simultaneously.
6. Add HTTP endpoints, structured logs, metrics for unknown queue depth and age, and a transaction-status endpoint.
7. Add a Kafka `ReconciliationQueue` adapter only when polling PostgreSQL no longer meets throughput or latency requirements; retain PostgreSQL as the transaction-history system of record.

## Acceptance checks

- A broker `2xx` is persisted as `COMPLETED` and returned synchronously.
- A broker `4xx` is persisted as `REJECTED` and returned synchronously.
- A timeout, closed connection, and `5xx` become `UNKNOWN` and return `202` with a transaction ID.
- Two workers cannot claim the same PostgreSQL transaction concurrently.
- A repeated request or retry uses the same idempotency key and does not create a second broker purchase.
- Backoff prevents a transaction from being processed before `next_retry`.
- A transaction exceeding the retry policy reaches `MANUAL_REVIEW` and is observable.

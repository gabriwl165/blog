package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/resilient-broker-purchases/internal/domain"
)

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) CreatePending(ctx context.Context, purchase domain.PurchaseRequest) (domain.Transaction, error) {
	payload, err := json.Marshal(purchase)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("marshal purchase: %w", err)
	}
	var transaction domain.Transaction
	var nextRetry pgtype.Timestamptz
	err = s.pool.QueryRow(ctx, `
		INSERT INTO transaction_history (status, idempotency_key, purchase_payload)
		VALUES ('PENDING', gen_random_uuid()::text, $1)
		RETURNING id::text, status, idempotency_key, purchase_payload, attempts, next_retry`, payload).
		Scan(&transaction.ID, &transaction.Status, &transaction.IdempotencyKey, &payload, &transaction.Attempts, &nextRetry)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("insert pending transaction: %w", err)
	}
	if err := json.Unmarshal(payload, &transaction.Purchase); err != nil {
		return domain.Transaction{}, fmt.Errorf("decode purchase: %w", err)
	}
	transaction.NextRetry = nullableTime(nextRetry)
	return transaction, nil
}

func (s *Store) Get(ctx context.Context, id string) (domain.Transaction, error) {
	return s.get(ctx, s.pool, `SELECT id::text, status, idempotency_key, purchase_payload, COALESCE(broker_order_id, ''), attempts, next_retry FROM transaction_history WHERE id = $1`, id)
}

func (s *Store) MarkCompleted(ctx context.Context, id string, result domain.BrokerResult) error {
	return s.markResult(ctx, id, domain.Completed, result)
}
func (s *Store) MarkRejected(ctx context.Context, id string, result domain.BrokerResult) error {
	return s.markResult(ctx, id, domain.Rejected, result)
}
func (s *Store) markResult(ctx context.Context, id string, status domain.Status, result domain.BrokerResult) error {
	_, err := s.pool.Exec(ctx, `UPDATE transaction_history SET status=$2, broker_order_id=$3, next_retry=NULL, updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id, status, result.BrokerOrderID)
	return err
}
func (s *Store) MarkUnknown(ctx context.Context, id string, next time.Time) error {
	return s.deferTo(ctx, id, next)
}
func (s *Store) MarkManualReview(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `UPDATE transaction_history SET status='MANUAL_REVIEW', next_retry=NULL, updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id)
	return err
}
func (s *Store) Defer(ctx context.Context, id string, next time.Time) error {
	return s.deferTo(ctx, id, next)
}
func (s *Store) deferTo(ctx context.Context, id string, next time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE transaction_history SET status='UNKNOWN', next_retry=$2, attempts=attempts+1, updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id, next)
	return err
}

// ClaimDue leases rows atomically, then returns after commit. The broker request never runs while
// a database row lock is held.
func (s *Store) ClaimDue(ctx context.Context, limit int, now time.Time) ([]domain.Transaction, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		WITH due AS (
			SELECT id FROM transaction_history
			WHERE status='UNKNOWN' AND next_retry <= $1
			ORDER BY next_retry FOR UPDATE SKIP LOCKED LIMIT $2
		)
		UPDATE transaction_history t SET next_retry=$3, updated_at=CURRENT_TIMESTAMP
		FROM due WHERE t.id=due.id
		RETURNING t.id::text, t.status, t.idempotency_key, t.purchase_payload,
			COALESCE(t.broker_order_id, ''), t.attempts, t.next_retry`, now, limit, now.Add(30*time.Second))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var claimed []domain.Transaction
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		claimed = append(claimed, transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return claimed, nil
}

type rowScanner interface{ Scan(...any) error }

func (s *Store) get(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, query, id string) (domain.Transaction, error) {
	return scanTransaction(q.QueryRow(ctx, query, id))
}
func scanTransaction(row rowScanner) (domain.Transaction, error) {
	var transaction domain.Transaction
	var payload []byte
	var nextRetry pgtype.Timestamptz
	if err := row.Scan(&transaction.ID, &transaction.Status, &transaction.IdempotencyKey, &payload, &transaction.BrokerOrderID, &transaction.Attempts, &nextRetry); err != nil {
		return domain.Transaction{}, err
	}
	if err := json.Unmarshal(payload, &transaction.Purchase); err != nil {
		return domain.Transaction{}, err
	}
	transaction.NextRetry = nullableTime(nextRetry)
	return transaction, nil
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

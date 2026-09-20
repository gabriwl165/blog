package ports

import (
	"context"
	"time"

	"example.com/resilient-broker-purchases/internal/domain"
)

// TransactionStore owns durable transaction history. Its implementation may be PostgreSQL,
// while callers depend only on this port.
type TransactionStore interface {
	CreatePending(context.Context, domain.PurchaseRequest) (domain.Transaction, error)
	Get(context.Context, string) (domain.Transaction, error)
	MarkCompleted(context.Context, string, domain.BrokerResult) error
	MarkRejected(context.Context, string, domain.BrokerResult) error
	MarkUnknown(context.Context, string, time.Time) error
	MarkManualReview(context.Context, string) error
}

// ReconciliationQueue is deliberately independent from TransactionStore. PostgreSQL polling
// implements it today; a Kafka consumer/producer can implement it later.
type ReconciliationQueue interface {
	ClaimDue(context.Context, int, time.Time) ([]domain.Transaction, error)
	Defer(context.Context, string, time.Time) error
}

type Broker interface {
	Submit(context.Context, domain.PurchaseRequest, string) (domain.BrokerResult, error)
	Lookup(context.Context, string) (result domain.BrokerResult, found bool, err error)
}

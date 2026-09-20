package application

import (
	"context"
	"errors"
	"net/http"
	"time"

	"example.com/resilient-broker-purchases/internal/domain"
	"example.com/resilient-broker-purchases/internal/ports"
)

type SubmitPurchase struct {
	Store    ports.TransactionStore
	Broker   ports.Broker
	Now      func() time.Time
	Deadline time.Duration
}

type SubmitResult struct {
	TransactionID string              `json:"transaction_id"`
	StatusCode    int                 `json:"-"`
	Status        domain.Status       `json:"status"`
	Broker        domain.BrokerResult `json:"broker,omitempty"`
}

func (s SubmitPurchase) Execute(ctx context.Context, purchase domain.PurchaseRequest) (SubmitResult, error) {
	transaction, err := s.Store.CreatePending(ctx, purchase)
	if err != nil {
		return SubmitResult{}, err
	}
	callCtx, cancel := context.WithTimeout(ctx, s.Deadline)
	defer cancel()
	result, err := s.Broker.Submit(callCtx, purchase, transaction.IdempotencyKey)
	if err != nil || result.StatusCode >= 500 {
		if updateErr := s.Store.MarkUnknown(ctx, transaction.ID, s.Now()); updateErr != nil {
			return SubmitResult{}, updateErr
		}
		return SubmitResult{TransactionID: transaction.ID, StatusCode: http.StatusAccepted, Status: domain.Unknown}, nil
	}
	if result.StatusCode >= 400 && result.StatusCode < 500 {
		if updateErr := s.Store.MarkRejected(ctx, transaction.ID, result); updateErr != nil {
			return SubmitResult{}, updateErr
		}
		return SubmitResult{TransactionID: transaction.ID, StatusCode: result.StatusCode, Status: domain.Rejected, Broker: result}, nil
	}
	if result.StatusCode >= 200 && result.StatusCode < 300 {
		if updateErr := s.Store.MarkCompleted(ctx, transaction.ID, result); updateErr != nil {
			return SubmitResult{}, updateErr
		}
		return SubmitResult{TransactionID: transaction.ID, StatusCode: result.StatusCode, Status: domain.Completed, Broker: result}, nil
	}
	return SubmitResult{}, errors.New("broker returned unsupported status")
}

type Reconciler struct {
	Store       ports.TransactionStore
	Queue       ports.ReconciliationQueue
	Broker      ports.Broker
	Now         func() time.Time
	MaxAttempts int
}

func (r Reconciler) ProcessDue(ctx context.Context, limit int) error {
	transactions, err := r.Queue.ClaimDue(ctx, limit, r.Now())
	if err != nil {
		return err
	}
	for _, transaction := range transactions {
		if err := r.reconcile(ctx, transaction); err != nil {
			return err
		}
	}
	return nil
}

func (r Reconciler) reconcile(ctx context.Context, transaction domain.Transaction) error {
	if transaction.Attempts >= r.MaxAttempts {
		return r.Store.MarkManualReview(ctx, transaction.ID)
	}
	if result, found, err := r.Broker.Lookup(ctx, transaction.IdempotencyKey); err == nil && found {
		return r.persistFinal(ctx, transaction.ID, result)
	}
	result, err := r.Broker.Submit(ctx, transaction.Purchase, transaction.IdempotencyKey)
	if err == nil && result.StatusCode < 500 {
		return r.persistFinal(ctx, transaction.ID, result)
	}
	return r.Queue.Defer(ctx, transaction.ID, r.Now().Add(backoff(transaction.Attempts)))
}

func (r Reconciler) persistFinal(ctx context.Context, id string, result domain.BrokerResult) error {
	if result.StatusCode >= 200 && result.StatusCode < 300 {
		return r.Store.MarkCompleted(ctx, id, result)
	}
	return r.Store.MarkRejected(ctx, id, result)
}

func backoff(attempt int) time.Duration {
	if attempt > 6 {
		attempt = 6
	}
	return time.Minute * time.Duration(1<<attempt)
}

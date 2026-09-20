package domain

import "time"

type Status string

const (
	Pending      Status = "PENDING"
	Completed    Status = "COMPLETED"
	Rejected     Status = "REJECTED"
	Unknown      Status = "UNKNOWN"
	ManualReview Status = "MANUAL_REVIEW"
)

type PurchaseRequest struct {
	AccountID string `json:"account_id"`
	Symbol    string `json:"symbol"`
	Quantity  int    `json:"quantity"`
}

type Transaction struct {
	ID             string
	Status         Status
	IdempotencyKey string
	Purchase       PurchaseRequest
	BrokerOrderID  string
	Attempts       int
	NextRetry      *time.Time
}

type BrokerResult struct {
	StatusCode    int    `json:"status_code"`
	BrokerOrderID string `json:"broker_order_id,omitempty"`
	Message       string `json:"message,omitempty"`
}

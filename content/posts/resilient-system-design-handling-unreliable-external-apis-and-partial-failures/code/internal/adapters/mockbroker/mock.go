package mockbroker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"sync"

	"example.com/resilient-broker-purchases/internal/domain"
)

// Broker is an in-memory third-party API simulator. A successful order is retained by
// idempotency key, including when the simulated response is lost after acceptance.
type Broker struct {
	mu       sync.Mutex
	rng      *rand.Rand
	accepted map[string]domain.BrokerResult
}

func New(rng *rand.Rand) *Broker {
	return &Broker{rng: rng, accepted: make(map[string]domain.BrokerResult)}
}

func (b *Broker) Submit(_ context.Context, purchase domain.PurchaseRequest, key string) (domain.BrokerResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if result, ok := b.accepted[key]; ok {
		return result, nil
	}

	switch b.rng.IntN(6) {
	case 0:
		return b.accept(key), nil
	case 1:
		return domain.BrokerResult{StatusCode: http.StatusBadRequest, Message: "broker rejected purchase"}, nil
	case 2:
		return domain.BrokerResult{StatusCode: http.StatusInternalServerError, Message: "broker unavailable"}, nil
	case 3:
		return domain.BrokerResult{}, context.DeadlineExceeded
	case 4:
		return domain.BrokerResult{}, io.ErrUnexpectedEOF
	default:
		// The broker accepted the order, but its response never reached the platform.
		b.accept(key)
		return domain.BrokerResult{}, errors.New("broker closed connection after accepting order")
	}
}

func (b *Broker) Lookup(_ context.Context, key string) (domain.BrokerResult, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	result, found := b.accepted[key]
	return result, found, nil
}

func (b *Broker) accept(key string) domain.BrokerResult {
	result := domain.BrokerResult{
		StatusCode:    http.StatusCreated,
		BrokerOrderID: fmt.Sprintf("broker-%08x", b.rng.Uint64()),
		Message:       "purchase accepted",
	}
	b.accepted[key] = result
	return result
}

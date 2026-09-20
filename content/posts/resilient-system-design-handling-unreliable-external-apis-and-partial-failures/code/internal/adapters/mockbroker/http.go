package mockbroker

import (
	"encoding/json"
	"net/http"
	"strings"

	"example.com/resilient-broker-purchases/internal/domain"
)

func (b *Broker) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			http.Error(w, "Idempotency-Key is required", http.StatusBadRequest)
			return
		}
		var purchase domain.PurchaseRequest
		if err := json.NewDecoder(r.Body).Decode(&purchase); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		result, err := b.Submit(r.Context(), purchase, key)
		if err != nil {
			// Closing without a response mimics a third-party socket failure.
			if strings.Contains(err.Error(), "closed connection") || err.Error() == "unexpected EOF" {
				panic(http.ErrAbortHandler)
			}
			http.Error(w, err.Error(), http.StatusGatewayTimeout)
			return
		}
		writeJSON(w, result.StatusCode, result)
	})
	mux.HandleFunc("GET /orders/{key}", func(w http.ResponseWriter, r *http.Request) {
		result, found, err := b.Lookup(r.Context(), r.PathValue("key"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

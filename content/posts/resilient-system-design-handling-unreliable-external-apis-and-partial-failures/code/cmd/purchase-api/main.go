package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/resilient-broker-purchases/internal/adapters/httpbroker"
	"example.com/resilient-broker-purchases/internal/adapters/postgres"
	"example.com/resilient-broker-purchases/internal/application"
	"example.com/resilient-broker-purchases/internal/domain"
)

func main() {
	pool := connect(os.Getenv("DATABASE_URL"))
	store := postgres.New(pool)
	broker := httpbroker.New(os.Getenv("BROKER_URL"))
	service := application.SubmitPurchase{Store: store, Broker: broker, Now: time.Now, Deadline: 2 * time.Second}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /purchases", func(w http.ResponseWriter, r *http.Request) {
		var request domain.PurchaseRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.AccountID == "" || request.Symbol == "" || request.Quantity <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "account_id, symbol, and positive quantity are required"})
			return
		}
		result, err := service.Execute(r.Context(), request)
		if err != nil {
			log.Printf("submit purchase: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		writeJSON(w, result.StatusCode, result)
	})
	mux.HandleFunc("GET /transactions/{id}", func(w http.ResponseWriter, r *http.Request) {
		transaction, err := store.Get(r.Context(), r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "transaction not found"})
			return
		}
		writeJSON(w, http.StatusOK, transaction)
	})
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func connect(url string) *pgxpool.Pool {
	for {
		pool, err := pgxpool.New(context.Background(), url)
		if err == nil {
			return pool
		}
		log.Printf("database unavailable: %v", err)
		time.Sleep(time.Second)
	}
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

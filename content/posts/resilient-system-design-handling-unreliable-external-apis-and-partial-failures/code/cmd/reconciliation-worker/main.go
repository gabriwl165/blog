package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/resilient-broker-purchases/internal/adapters/httpbroker"
	"example.com/resilient-broker-purchases/internal/adapters/postgres"
	"example.com/resilient-broker-purchases/internal/application"
)

func main() {
	pool := connect(os.Getenv("DATABASE_URL"))
	store := postgres.New(pool)
	broker := httpbroker.New(os.Getenv("BROKER_URL"))
	reconciler := application.Reconciler{Store: store, Queue: store, Broker: broker, Now: time.Now, MaxAttempts: 8}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := reconciler.ProcessDue(context.Background(), 50); err != nil {
			log.Printf("reconcile due purchases: %v", err)
		}
	}
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

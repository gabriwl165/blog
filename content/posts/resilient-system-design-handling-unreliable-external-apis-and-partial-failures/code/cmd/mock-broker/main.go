package main

import (
	"log"
	"math/rand/v2"
	"net/http"
	"time"

	"example.com/resilient-broker-purchases/internal/adapters/mockbroker"
)

func main() {
	broker := mockbroker.New(rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 1)))
	log.Fatal(http.ListenAndServe(":8090", broker.Handler()))
}

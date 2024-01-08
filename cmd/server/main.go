package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/shevchukeugeni/metrics/internal/server"
	"github.com/shevchukeugeni/metrics/internal/store"
)

var flagRunAddr string

func init() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")

	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagRunAddr = envRunAddr
	}
}

func main() {
	flag.Parse()

	memStorage := store.NewMemStorage()

	router := server.SetupRouter(memStorage)

	log.Println("Running server on", flagRunAddr)
	err := http.ListenAndServe(flagRunAddr, router)
	if err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe Error: %v", err)
	}
}

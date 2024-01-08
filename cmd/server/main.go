package main

import (
	"flag"
	"net/http"
	"os"

	"go.uber.org/zap"

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

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	memStorage := store.NewMemStorage()

	router := server.SetupRouter(logger, memStorage)

	logger.Info("Running server on", zap.String("address", flagRunAddr))
	err = http.ListenAndServe(flagRunAddr, router)
	if err != http.ErrServerClosed {
		logger.Fatal("HTTP server ListenAndServe Error", zap.Error(err))
	}
}

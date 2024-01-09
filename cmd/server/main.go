package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/caarlos0/env/v6"
	"go.uber.org/zap"

	"github.com/shevchukeugeni/metrics/internal/server"
	"github.com/shevchukeugeni/metrics/internal/store"
	"github.com/shevchukeugeni/metrics/internal/types"
)

var dcfg types.DumpConfig

var flagRunAddr string

func init() {
	flag.UintVar(&dcfg.StoreInterval, "i", 300, "dump to file interval")
	flag.BoolVar(&dcfg.Restore, "r", true, "restore data from file")
	flag.StringVar(&dcfg.FileStoragePath, "f", "/tmp/metrics-db.json", "dump file path")

	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")

	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagRunAddr = envRunAddr
	}
}

func main() {
	flag.Parse()

	err := env.Parse(&dcfg)
	if err != nil {
		log.Fatal(err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	memStorage := store.NewMemStorage()

	dumpWorker := store.NewDumpWorker(logger, &dcfg, memStorage)

	router := server.SetupRouter(logger, memStorage, dumpWorker)

	ctx, cancelCtx := context.WithCancel(context.Background())

	if dumpWorker != nil {
		go dumpWorker.Start(ctx)
	}

	logger.Info("Running server on", zap.String("address", flagRunAddr))
	err = http.ListenAndServe(flagRunAddr, router)
	if err != http.ErrServerClosed {
		logger.Fatal("HTTP server ListenAndServe Error", zap.Error(err))
	}

	cancelCtx()
}

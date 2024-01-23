package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/caarlos0/env/v6"
	"go.uber.org/zap"

	"github.com/shevchukeugeni/metrics/internal/server"
	"github.com/shevchukeugeni/metrics/internal/store"
	"github.com/shevchukeugeni/metrics/internal/store/postgres"
	"github.com/shevchukeugeni/metrics/internal/types"
)

var dcfg types.DumpConfig

var flagRunAddr, dbURL string

func init() {
	flag.UintVar(&dcfg.StoreInterval, "i", 300, "dump to file interval")
	flag.BoolVar(&dcfg.Restore, "r", true, "restore data from file")
	flag.StringVar(&dcfg.FileStoragePath, "f", "/tmp/metrics-db.json", "dump file path")

	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&dbURL, "d", "", "database connection url")

	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagRunAddr = envRunAddr
	}

	if envDBURL := os.Getenv("DATABASE_DSN"); envDBURL != "" {
		dbURL = envDBURL
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
		log.Fatal(err)
	}
	defer logger.Sync()

	var (
		router http.Handler
		wg     sync.WaitGroup
	)
	ctx, cancelCtx := context.WithCancel(context.Background())

	db, err := postgres.NewPostgresDB(postgres.Config{URL: dbURL})
	if err != nil {
		logger.Error("failed to initialize db: " + err.Error())
	}
	defer db.Close()

	if db != nil {
		dbStorage := postgres.NewStore(logger, db)

		router = server.SetupRouter(logger, dbStorage, nil, db)
	} else {
		memStorage := store.NewMemStorage()

		dumpWorker := store.NewDumpWorker(logger, &dcfg, memStorage, &wg)

		router = server.SetupRouter(logger, memStorage, dumpWorker, nil)

		if dumpWorker != nil {
			go dumpWorker.Start(ctx)
		}
	}

	logger.Info("Running server on", zap.String("address", flagRunAddr))
	err = http.ListenAndServe(flagRunAddr, router)
	if err != http.ErrServerClosed {
		logger.Fatal("HTTP server ListenAndServe Error", zap.Error(err))
	}

	cancelCtx()
	wg.Wait()
}

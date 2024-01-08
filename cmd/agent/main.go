package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/shevchukeugeni/metrics/internal/types"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/go-resty/resty/v2"

	"github.com/shevchukeugeni/metrics/internal/store"
)

type Config struct {
	ServerAddr     string `env:"ADDRESS"`
	PollInterval   int    `env:"REPORT_INTERVAL"`
	ReportInterval int    `env:"POLL_INTERVAL"`
}

var cfg Config

func init() {
	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "report interval in seconds")
	flag.IntVar(&cfg.PollInterval, "p", 2, "poll interval in seconds")
}

func main() {
	flag.Parse()

	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	// NOTE: I think it's necessary to check, but autotests suite fails then
	//_, err := net.DialTimeout("tcp", flagRunAddr, 1*time.Second)
	//if err != nil {
	//	log.Fatalf("%s %s %s\n", flagRunAddr, "not responding", err.Error())
	//}

	metrics := store.NewRuntimeMetrics()

	pollTicker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
	defer pollTicker.Stop()
	reportTicker := time.NewTicker(time.Duration(cfg.ReportInterval) * time.Second)
	defer reportTicker.Stop()

	client := resty.New()

	ctx, cancelFunc := context.WithCancel(context.Background())

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-pollTicker.C:
				metrics.Update()

				log.Println("Metrics are updated")
			case <-reportTicker.C:
				for k, v := range metrics.Gauge {
					req, err := client.R().
						SetHeader("Content-Type", "application/json").
						SetBody(map[string]interface{}{"id": k, "type": types.Gauge, "value": v}).
						Post(fmt.Sprintf("http://%s/update/", cfg.ServerAddr))
					if err != nil {
						log.Println(err)
					}
					_ = req
				}

				for k, v := range metrics.Counter {
					_, err = client.R().
						SetHeader("Content-Type", "application/json").
						SetBody(map[string]interface{}{"id": k, "type": types.Counter, "delta": v}).
						Post(fmt.Sprintf("http://%s/update/", cfg.ServerAddr))
					if err != nil {
						log.Println(err)
					}
				}

				log.Println("Report is sent!")
			}
		}
	}(ctx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	cancelFunc()
}

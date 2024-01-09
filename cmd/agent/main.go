package main

import (
	"bytes"
	"compress/gzip"
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
					data := fmt.Sprintf("{\"id\": \"%s\", \"type\": \"%s\", \"value\": %v}", k, types.Gauge, v)
					cdata, err := Compress([]byte(data))
					if err != nil {
						log.Println(err)
						return
					}

					_, err = client.R().
						SetHeader("Content-Type", "application/json").
						SetHeader("Content-Encoding", "gzip").
						SetBody(cdata).
						Post(fmt.Sprintf("http://%s/update/", cfg.ServerAddr))
					if err != nil {
						log.Println(err)
					}
				}

				for k, v := range metrics.Counter {
					data := fmt.Sprintf("{\"id\": \"%s\", \"type\": \"%s\", \"delta\": %v}", k, types.Counter, v)
					cdata, err := Compress([]byte(data))
					if err != nil {
						log.Println(err)
						return
					}
					_, err = client.R().
						SetHeader("Content-Type", "application/json").
						SetHeader("Content-Encoding", "gzip").
						SetBody(cdata).
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

// Compress сжимает слайс байт.
func Compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)

	_, err := w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed write data to compress temporary buffer: %v", err)
	}
	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed compress data: %v", err)
	}

	return b.Bytes(), nil
}

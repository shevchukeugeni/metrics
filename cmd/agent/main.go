package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/avast/retry-go"
	"github.com/caarlos0/env/v6"
	"github.com/go-resty/resty/v2"

	"github.com/shevchukeugeni/metrics/internal/store"
	"github.com/shevchukeugeni/metrics/internal/types"
)

type Config struct {
	ServerAddr     string `env:"ADDRESS"`
	PollInterval   int    `env:"REPORT_INTERVAL"`
	ReportInterval int    `env:"POLL_INTERVAL"`
	SignKey        string `env:"KEY"`
}

var cfg Config

func init() {
	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "report interval in seconds")
	flag.IntVar(&cfg.PollInterval, "p", 2, "poll interval in seconds")
	flag.StringVar(&cfg.SignKey, "k", "", "hash signing key")
}

func main() {
	flag.Parse()

	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	var sign = cfg.SignKey != ""

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
				type mtrc struct {
					ID    string  `json:"id"`
					Mtype string  `json:"type"`
					Value float64 `json:"value"`
					Delta int64   `json:"delta"`
				}

				var mtrcs []mtrc

				for k, v := range metrics.Gauge {
					mtrcs = append(mtrcs, mtrc{ID: k, Mtype: types.Gauge, Value: v})
				}

				for k, v := range metrics.Counter {
					mtrcs = append(mtrcs, mtrc{ID: k, Mtype: types.Counter, Delta: v})
				}

				data, err := json.Marshal(mtrcs)
				if err != nil {
					log.Println(err)
					return
				}

				cdata, err := Compress(data)
				if err != nil {
					log.Println(err)
					return
				}

				req := client.R().
					SetHeader("Content-Type", "application/json").
					SetHeader("Content-Encoding", "gzip")

				if sign {
					h := hmac.New(sha256.New, []byte(cfg.SignKey))
					h.Write(data)
					b := base64.StdEncoding.EncodeToString(h.Sum(nil))
					req.SetHeader("HashSHA256", b)
				}

				var innerErr error

				err = WithRetry(func() error {
					_, innerErr = req.SetBody(cdata).Post(fmt.Sprintf("http://%s/updates/", cfg.ServerAddr))
					if innerErr != nil {
						return innerErr
					}
					return nil
				}, "failed to send metric")
				if err != nil {
					log.Println(err)
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

func WithRetry(fn func() error, warn string) error {
	interval := time.Second
	return retry.Do(fn,
		retry.Attempts(3),
		retry.Delay(interval),
		retry.OnRetry(func(n uint, err error) {
			log.Println(warn, zap.Uint("attempt", n), zap.Error(err))
			interval += 2 * time.Second
		}))
}

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shevchukeugeni/metrics/internal/utils"
)

const pollInterval = 2 * time.Second
const reportInterval = 10 * time.Second

func main() {
	metrics := utils.NewRuntimeMetrics()

	pollTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)

	go func() {
		for {
			select {
			case <-pollTicker.C:
				metrics.Update()

				log.Println("Metrics are updated")
			case <-reportTicker.C:
				for k, v := range metrics.Gauge {
					req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:8080/update/gauge/%s/%s", k, v), nil)
					if err != nil {
						log.Println(err)
					}

					res, err := http.DefaultClient.Do(req)
					defer res.Body.Close()

					if err != nil {
						log.Println(err)
					}
				}

				for k, v := range metrics.Counter {
					req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:8080/update/counter/%s/%v", k, v), nil)
					if err != nil {
						log.Println(err)
					}

					res, err := http.DefaultClient.Do(req)
					defer res.Body.Close()

					if err != nil {
						log.Println(err)
					}
				}

				log.Println("Report is sent!")
			}
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}

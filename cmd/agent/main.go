package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shevchukeugeni/metrics/internal/utils"
)

const pollInterval = 2 * time.Second
const reportInterval = 10 * time.Second

func main() {
	metrics := utils.NewRuntimeMetrics()

	pollTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)

	client := resty.New()

	go func() {
		for {
			select {
			case <-pollTicker.C:
				metrics.Update()

				log.Println("Metrics are updated")
			case <-reportTicker.C:
				for k, v := range metrics.Gauge {
					_, err := client.R().SetPathParams(map[string]string{
						"name":  k,
						"value": v,
					}).Post("http://localhost:8080/update/gauge/{name}/{value}")
					if err != nil {
						log.Println(err)
					}
				}

				for k, v := range metrics.Counter {
					_, err := client.R().SetPathParams(map[string]string{
						"name":  k,
						"value": fmt.Sprint(v),
					}).Post("http://localhost:8080/update/counter/{name}/{value}")
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

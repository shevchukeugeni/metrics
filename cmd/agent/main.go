package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shevchukeugeni/metrics/internal/utils"
)

var (
	flagRunAddr                  string
	pollInterval, reportInterval int
)

func init() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&reportInterval, "r", 10, "report interval in seconds")
	flag.IntVar(&pollInterval, "p", 2, "poll interval in econds")
}

func main() {
	flag.Parse()

	_, err := net.DialTimeout("tcp", flagRunAddr, 1*time.Second)
	if err != nil {
		log.Fatalf("%s %s %s\n", flagRunAddr, "not responding", err.Error())
	}

	metrics := utils.NewRuntimeMetrics()

	pollTicker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	reportTicker := time.NewTicker(time.Duration(reportInterval) * time.Second)

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
					}).Post("http://" + flagRunAddr + "/update/gauge/{name}/{value}")
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

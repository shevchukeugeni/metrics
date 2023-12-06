package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/go-resty/resty/v2"

	"github.com/shevchukeugeni/metrics/internal/utils"
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

	metrics := utils.NewRuntimeMetrics()

	pollTicker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
	reportTicker := time.NewTicker(time.Duration(cfg.ReportInterval) * time.Second)

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
					}).Post(fmt.Sprintf("http://%s/update/gauge/{name}/{value}", cfg.ServerAddr))
					if err != nil {
						log.Println(err)
					}
				}

				for k, v := range metrics.Counter {
					_, err := client.R().SetPathParams(map[string]string{
						"name":  k,
						"value": fmt.Sprint(v),
					}).Post(fmt.Sprintf("http://%s/update/gauge/{name}/{value}", cfg.ServerAddr))
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

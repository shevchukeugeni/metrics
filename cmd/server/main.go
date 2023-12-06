package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/shevchukeugeni/metrics/internal/server"
	"github.com/shevchukeugeni/metrics/internal/utils"
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

	memStorage := utils.NewMemStorage()

	router := server.SetupRouter(memStorage)

	log.Println("Running server on", flagRunAddr)
	err := http.ListenAndServe(flagRunAddr, router)
	if err != nil {
		log.Fatal(err)
	}
}

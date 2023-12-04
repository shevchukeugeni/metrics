package main

import (
	"log"
	"net/http"

	"github.com/shevchukeugeni/metrics/internal/server"
	"github.com/shevchukeugeni/metrics/internal/utils"
)

func main() {
	memStorage := utils.NewMemStorage()

	router := server.SetupRouter(memStorage)

	err := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatal(err)
	}
}

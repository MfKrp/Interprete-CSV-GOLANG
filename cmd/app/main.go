package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/MfKrp/Interprete-CSV-GOLANG/config"
	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/handlers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /health", handlers.Health())

	addr := ":" + strconv.Itoa(cfg.Port)
	log.Printf("CSV Copilot listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("HTTP server: %v", err)
	}
}

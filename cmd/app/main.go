package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/MfKrp/Interprete-CSV-GOLANG/config"
	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/handlers"
	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/repository"
	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.Dir("web")))
	mux.Handle("GET /health", handlers.Health())
	repo := repository.NewDatasetRepository()
	maxBytes := int64(10 * 1024 * 1024)
	if cfg.MaxUploadSizeMB < 10 {
		maxBytes = cfg.MaxUploadSizeMB * 1024 * 1024
	}
	handlers.NewCSVHandler(repo, maxBytes).Register(mux)
	analysisService := service.NewAnalysisService(repo)
	ollamaClient := service.NewOllamaClient(cfg.OllamaURL, cfg.OllamaModel, &http.Client{Timeout: cfg.OllamaTimeout})
	handlers.NewQueryHandler(repo, ollamaClient, analysisService).Register(mux)

	addr := ":" + strconv.Itoa(cfg.Port)
	log.Printf("CSV Copilot listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("HTTP server: %v", err)
	}
}

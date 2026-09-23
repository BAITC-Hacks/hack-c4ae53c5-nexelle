package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/config"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/explanation"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/importer"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/service"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/store"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/transport/httpapi"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	dataset, report, err := importer.LoadDir(context.Background(), cfg.DataDir)
	if err != nil {
		if errors.Is(err, importer.ErrDatasetInvalid) {
			for _, issue := range report.Errors {
				log.Printf("dataset validation error: %s", issue)
			}
		}
		log.Fatalf("load dataset from %q: %v", cfg.DataDir, err)
	}

	memoryStore := store.NewMemoryStore()
	if err := memoryStore.ReplaceDataset(context.Background(), dataset); err != nil {
		log.Fatalf("store dataset: %v", err)
	}
	careerService, err := service.NewCareerService(memoryStore, cfg.SnapshotDate)
	if err != nil {
		log.Fatalf("create career service: %v", err)
	}
	if endpoint := os.Getenv("LLM_EXPLANATION_URL"); endpoint != "" {
		careerService.SetExplainer(explanation.New(explanation.HTTPProvider{URL: endpoint, Token: os.Getenv("LLM_API_KEY")}))
	}
	webDir := os.Getenv("WEB_DIR")
	if webDir == "" {
		webDir = "./web/dist"
	}

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           httpapi.WithFrontend(httpapi.NewHandler(memoryStore, careerService), webDir),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("career-quest API listens on %s", cfg.Address())
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve HTTP: %v", err)
	}
}

package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/config"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/importer"
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

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           httpapi.NewHandler(memoryStore),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	log.Printf("career-quest API listens on %s", cfg.Address())
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve HTTP: %v", err)
	}
}

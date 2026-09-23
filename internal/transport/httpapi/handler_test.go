package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/store"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/transport/httpapi"
)

func TestHealthReturnsDatasetStats(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	dataset := domain.NewDataset()
	dataset.Skills["SK_GO"] = domain.Skill{SkillID: "SK_GO"}
	if err := memoryStore.ReplaceDataset(context.Background(), dataset); err != nil {
		t.Fatalf("ReplaceDataset() error = %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	httpapi.NewHandler(memoryStore).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestDatasetStatsRequiresHRRole(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	if err := memoryStore.ReplaceDataset(context.Background(), domain.NewDataset()); err != nil {
		t.Fatalf("ReplaceDataset() error = %v", err)
	}

	handler := httpapi.NewHandler(memoryStore)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dataset/stats", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status without role = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/dataset/stats", nil)
	request.Header.Set("X-Demo-Role", "hr")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status with HR role = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

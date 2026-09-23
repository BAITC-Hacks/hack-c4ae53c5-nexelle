package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestEmployeeProfileLimitsEmployeeToOwnData(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	dataset := domain.NewDataset()
	dataset.Employees["E0001"] = domain.Employee{EmployeeID: "E0001", FullName: "Own Profile", Skills: map[string]int{}}
	dataset.Employees["E0002"] = domain.Employee{EmployeeID: "E0002", FullName: "Other Profile", Skills: map[string]int{}}
	if err := memoryStore.ReplaceDataset(context.Background(), dataset); err != nil {
		t.Fatalf("ReplaceDataset() error = %v", err)
	}

	handler := httpapi.NewHandler(memoryStore)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/employees/E0001/profile", nil)
	request.Header.Set("X-Demo-Role", "employee")
	request.Header.Set("X-Employee-ID", "E0001")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Own Profile") {
		t.Fatalf("own profile: status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/employees/E0002/profile", nil)
	request.Header.Set("X-Demo-Role", "employee")
	request.Header.Set("X-Employee-ID", "E0001")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("other profile: status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

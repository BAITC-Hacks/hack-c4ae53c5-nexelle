package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/service"
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
	newHandler(t, memoryStore).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestDatasetStatsRequiresHRRole(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	if err := memoryStore.ReplaceDataset(context.Background(), domain.NewDataset()); err != nil {
		t.Fatalf("ReplaceDataset() error = %v", err)
	}

	handler := newHandler(t, memoryStore)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dataset/stats", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
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
	dataset.RoleProfiles[domain.RoleGradeKey("Backend Engineer", domain.GradeMiddle)] = domain.RoleProfile{
		Role: "Backend Engineer", Grade: domain.GradeMiddle, RequiredSkills: map[string]int{},
	}
	dataset.Employees["E0001"] = domain.Employee{EmployeeID: "E0001", FullName: "Own Profile", Role: "Backend Engineer", Grade: domain.GradeJunior, LastReviewDate: "2026-09-01", Skills: map[string]int{}}
	dataset.Employees["E0002"] = domain.Employee{EmployeeID: "E0002", FullName: "Other Profile", Role: "Backend Engineer", Grade: domain.GradeJunior, LastReviewDate: "2026-09-01", Skills: map[string]int{}}
	if err := memoryStore.ReplaceDataset(context.Background(), dataset); err != nil {
		t.Fatalf("ReplaceDataset() error = %v", err)
	}

	handler := newHandler(t, memoryStore)
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

func TestCORSAllowsConfiguredLocalFrontend(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	if err := memoryStore.ReplaceDataset(context.Background(), domain.NewDataset()); err != nil {
		t.Fatalf("ReplaceDataset() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/employees/E0001/profile", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	recorder := httptest.NewRecorder()
	newHandler(t, memoryStore).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("CORS status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q", origin)
	}
}

func newHandler(t *testing.T, memoryStore *store.MemoryStore) http.Handler {
	t.Helper()
	careerService, err := service.NewCareerService(memoryStore, time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewCareerService() error = %v", err)
	}
	return httpapi.NewHandler(memoryStore, careerService)
}

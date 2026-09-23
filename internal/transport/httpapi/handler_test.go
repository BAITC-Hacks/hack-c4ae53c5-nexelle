package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
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

func TestCollectionRoutesRespectRolesAndExposeCatalog(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	dataset := apiTestDataset()
	if err := memoryStore.ReplaceDataset(context.Background(), dataset); err != nil {
		t.Fatalf("ReplaceDataset() error = %v", err)
	}
	handler := newHandler(t, memoryStore)

	request := httptest.NewRequest(http.MethodGet, "/employees", nil)
	request.Header.Set("X-Demo-Role", "employee")
	request.Header.Set("X-Employee-ID", "E0001")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden || !strings.Contains(recorder.Body.String(), "Сотрудник имеет доступ только к собственному профилю") {
		t.Fatalf("employee list: status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/employees", nil)
	request.Header.Set("X-Demo-Role", "hr")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Test Employee") {
		t.Fatalf("HR employee list: status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/events", nil)
	request.Header.Set("X-Demo-Role", "employee")
	request.Header.Set("X-Employee-ID", "E0001")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "EV_001") {
		t.Fatalf("event catalog: status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/employees/E0002/history", nil)
	request.Header.Set("X-Demo-Role", "employee")
	request.Header.Set("X-Employee-ID", "E0001")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden || !strings.Contains(recorder.Body.String(), "Сотрудник имеет доступ только к собственному профилю") {
		t.Fatalf("other history: status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestCompletionIsVisibleInProfileAndHistory(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	if err := memoryStore.ReplaceDataset(context.Background(), apiTestDataset()); err != nil {
		t.Fatalf("ReplaceDataset() error = %v", err)
	}
	handler := newHandler(t, memoryStore)

	request := httptest.NewRequest(http.MethodPost, "/api/employees/E0001/complete", strings.NewReader(`{"eventId":"EV_001"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Demo-Role", "employee")
	request.Header.Set("X-Employee-ID", "E0001")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("complete: status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/employees/E0001/history", nil)
	request.Header.Set("X-Demo-Role", "employee")
	request.Header.Set("X-Employee-ID", "E0001")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"status":"completed"`) {
		t.Fatalf("history after completion: status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/employees/E0001", nil)
	request.Header.Set("X-Demo-Role", "employee")
	request.Header.Set("X-Employee-ID", "E0001")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"SK_GO":2`) {
		t.Fatalf("profile after completion: status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestImportReplacesDatasetWithoutRestart(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	if err := memoryStore.ReplaceDataset(context.Background(), domain.NewDataset()); err != nil {
		t.Fatalf("ReplaceDataset() error = %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, content := range importFixture() {
		part, err := writer.CreateFormFile(name, name)
		if err != nil {
			t.Fatalf("CreateFormFile(%q): %v", name, err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatalf("write %q: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/import", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-Demo-Role", "hr")
	recorder := httptest.NewRecorder()
	newHandler(t, memoryStore).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("import: status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil || response.Status != "ok" {
		t.Fatalf("import response = %s, error = %v", recorder.Body.String(), err)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/events", nil)
	request.Header.Set("X-Demo-Role", "hr")
	recorder = httptest.NewRecorder()
	newHandler(t, memoryStore).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "EV_001") {
		t.Fatalf("catalog after import: status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func apiTestDataset() domain.Dataset {
	dataset := domain.NewDataset()
	dataset.Skills["SK_GO"] = domain.Skill{SkillID: "SK_GO", Type: "hard"}
	dataset.RoleProfiles[domain.RoleGradeKey("Backend Engineer", domain.GradeMiddle)] = domain.RoleProfile{
		Role: "Backend Engineer", Grade: domain.GradeMiddle, RequiredSkills: map[string]int{"SK_GO": 2}, CriticalSkills: []string{"SK_GO"},
	}
	dataset.Employees["E0001"] = domain.Employee{
		EmployeeID: "E0001", FullName: "Test Employee", Department: "Engineering", Role: "Backend Engineer", Grade: domain.GradeJunior,
		LastReviewDate: "2026-09-01", Skills: map[string]int{"SK_GO": 1},
	}
	dataset.Employees["E0002"] = domain.Employee{
		EmployeeID: "E0002", FullName: "Other Employee", Department: "Engineering", Role: "Backend Engineer", Grade: domain.GradeJunior,
		LastReviewDate: "2026-09-01", Skills: map[string]int{"SK_GO": 1},
	}
	dataset.Events["EV_001"] = domain.Event{
		EventID: "EV_001", Title: "Go basics", Format: "self_paced", DurationHours: 1,
		TargetRoles: []string{"Backend Engineer"}, TargetGrades: []string{domain.GradeJunior},
		DevelopsSkills: []domain.EventSkillGain{{SkillID: "SK_GO", Gain: 1, MaxLevel: 3}}, Prerequisites: map[string]int{},
	}
	return dataset
}

func importFixture() map[string]string {
	return map[string]string{
		"skills.json":          `{"proficiency_scale":{"0":"None"},"skills":[{"skill_id":"SK_GO","name":"Go","type":"hard","category":"engineering","description":"Go"}],"role_profiles":[{"role":"Backend Engineer","grade":"Junior","required_skills":{"SK_GO":1},"critical_skills":["SK_GO"]}]}`,
		"employees.json":       `{"employees":[{"employee_id":"E0001","full_name":"Imported Employee","department":"Engineering","role":"Backend Engineer","grade":"Junior","manager_id":null,"hire_date":"2026-01-01","tenure_months":9,"work_format":"hybrid","preferred_language":"ru","career_goal":null,"skills":{"SK_GO":1},"last_review_date":"2026-08-01"}]}`,
		"events.json":          `{"events":[{"event_id":"EV_001","title":"Go Basics","description":"Course","type":"course","format":"self_paced","duration_hours":2,"mandatory":false,"target_roles":["Backend Engineer"],"target_grades":["Junior"],"develops_skills":[{"skill_id":"SK_GO","gain":1,"max_level":3}],"prerequisites":{},"upcoming_sessions":[]}]}`,
		"activity_history.csv": "record_id,employee_id,event_id,date,due_date,status,completion_pct,score,feedback_rating,assigned_by\n",
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

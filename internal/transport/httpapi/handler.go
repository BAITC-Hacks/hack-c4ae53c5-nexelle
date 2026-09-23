// Package httpapi предоставляет REST API Career Quest и demo-разграничение ролей.
package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/explanation"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/importer"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/service"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/store"
)

// CareerService описывает use cases, нужные HTTP-обработчикам.
// Контракт позволяет заменить реализацию без изменения роутов и UI.
type CareerService interface {
	GetProfile(ctx context.Context, employeeID string) (domain.ProfileResponse, error)
	CompleteEvent(ctx context.Context, employeeID, eventID string) (domain.CompletionResponse, error)
	GetHRAnalytics(ctx context.Context) (domain.HRAnalytics, error)
}

// Handler хранит зависимости HTTP-слоя.
type Handler struct {
	store  store.DatasetStore
	career CareerService
}

// NewHandler создаёт маршруты API и добавляет защиту от panic и CORS для локального UI.
func NewHandler(datasetStore store.DatasetStore, careerService CareerService) http.Handler {
	handler := Handler{store: datasetStore, career: careerService}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.health)
	mux.HandleFunc("GET /api/v1/dataset/stats", handler.datasetStats)

	mux.HandleFunc("GET /employees", handler.listEmployees)
	mux.HandleFunc("GET /api/employees", handler.listEmployees)
	mux.HandleFunc("GET /employees/{id}", handler.employeeProfile)
	mux.HandleFunc("GET /api/employees/{id}", handler.employeeProfile)
	mux.HandleFunc("GET /employees/{id}/history", handler.employeeHistory)
	mux.HandleFunc("GET /api/employees/{id}/history", handler.employeeHistory)
	mux.HandleFunc("GET /events", handler.listEvents)
	mux.HandleFunc("GET /api/events", handler.listEvents)
	mux.HandleFunc("GET /api/skills", handler.listSkills)

	mux.HandleFunc("GET /hr/dashboard", handler.hrAnalytics)
	mux.HandleFunc("GET /api/hr/dashboard", handler.hrAnalytics)
	mux.HandleFunc("GET /api/v1/hr/analytics", handler.hrAnalytics)
	mux.HandleFunc("GET /api/v1/employees/{id}/profile", handler.employeeProfile)
	mux.HandleFunc("POST /employees/{id}/complete", handler.completeEvent)
	mux.HandleFunc("POST /api/v1/employees/{id}/complete", handler.completeEvent)
	mux.HandleFunc("POST /api/employees/{id}/complete", handler.completeEvent)
	mux.HandleFunc("POST /import", handler.importDataset)
	mux.HandleFunc("POST /api/import", handler.importDataset)
	return handler.recover(handler.cors(localeMiddleware(mux)))
}

func localeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := r.URL.Query().Get("locale")
		if locale == "" {
			locale = "ru"
		}
		if locale != "ru" && locale != "kk" {
			writeError(w, http.StatusBadRequest, "invalid_locale", "Поддерживаются ru и kk")
			return
		}
		next.ServeHTTP(w, r.WithContext(explanation.WithLocale(r.Context(), locale)))
	})
}

func (h Handler) listSkills(w http.ResponseWriter, r *http.Request) {
	if _, status, code, message := actorFromRequest(r); status != 0 {
		writeError(w, status, code, message)
		return
	}
	dataset, err := h.store.GetDataset(r.Context())
	if err != nil {
		writeCareerError(w, err)
		return
	}
	result := make([]domain.Skill, 0, len(dataset.Skills))
	for _, skill := range dataset.Skills {
		result = append(result, skill)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].SkillID < result[j].SkillID })
	writeJSON(w, http.StatusOK, result)
}

func (h Handler) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	dataset, err := h.store.GetDataset(ctx)
	if err == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"dataset": dataset.Stats(),
		})
		return
	}
	if errors.Is(err, store.ErrDatasetNotLoaded) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "dataset_not_loaded"})
		return
	}
	writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось проверить состояние приложения")
}

func (h Handler) datasetStats(w http.ResponseWriter, r *http.Request) {
	actor, status, code, message := actorFromRequest(r)
	if status != 0 {
		writeError(w, status, code, message)
		return
	}
	if actor.Role != "hr" {
		if actor.Role == "employee" {
			writeError(w, http.StatusForbidden, "forbidden", "Сотрудник имеет доступ только к собственному профилю")
			return
		}
		writeError(w, http.StatusForbidden, "forbidden", "Требуется роль HR")
		return
	}

	dataset, err := h.store.GetDataset(r.Context())
	if errors.Is(err, store.ErrDatasetNotLoaded) {
		writeError(w, http.StatusConflict, "dataset_not_loaded", "Набор данных не загружен")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить данные")
		return
	}
	writeJSON(w, http.StatusOK, dataset.Stats())
}

type employeeListItem struct {
	ID         string `json:"id"`
	FullName   string `json:"full_name"`
	Role       string `json:"role"`
	Grade      string `json:"grade"`
	Department string `json:"department"`
}

func (h Handler) listEmployees(w http.ResponseWriter, r *http.Request) {
	if !requireHR(w, r) {
		return
	}
	employees, err := h.store.ListEmployees(r.Context())
	if err != nil {
		writeCareerError(w, err)
		return
	}
	sort.Slice(employees, func(i, j int) bool {
		return employees[i].EmployeeID < employees[j].EmployeeID
	})
	result := make([]employeeListItem, 0, len(employees))
	for _, employee := range employees {
		result = append(result, employeeListItem{
			ID:         employee.EmployeeID,
			FullName:   employee.FullName,
			Role:       employee.Role,
			Grade:      employee.Grade,
			Department: employee.Department,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (h Handler) employeeProfile(w http.ResponseWriter, r *http.Request) {
	employeeID, ok := authorizeEmployeeResource(w, r)
	if !ok {
		return
	}
	profile, err := h.career.GetProfile(r.Context(), employeeID)
	if err != nil {
		writeCareerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

// employeeHistory reads the current store state, so a newly completed event is
// visible immediately without restarting the server.
func (h Handler) employeeHistory(w http.ResponseWriter, r *http.Request) {
	employeeID, ok := authorizeEmployeeResource(w, r)
	if !ok {
		return
	}
	activities, err := h.store.ListEmployeeActivities(r.Context(), employeeID)
	if err != nil {
		writeCareerError(w, err)
		return
	}
	sort.SliceStable(activities, func(i, j int) bool {
		if activities[i].Date.Equal(activities[j].Date) {
			return activities[i].RecordID < activities[j].RecordID
		}
		return activities[i].Date.Before(activities[j].Date)
	})
	writeJSON(w, http.StatusOK, activities)
}

func (h Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	if _, status, code, message := actorFromRequest(r); status != 0 {
		writeError(w, status, code, message)
		return
	}
	events, err := h.store.ListEvents(r.Context())
	if err != nil {
		writeCareerError(w, err)
		return
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].EventID < events[j].EventID
	})
	writeJSON(w, http.StatusOK, events)
}

func (h Handler) hrAnalytics(w http.ResponseWriter, r *http.Request) {
	actor, status, code, message := actorFromRequest(r)
	if status != 0 {
		writeError(w, status, code, message)
		return
	}
	if actor.Role != "hr" {
		writeError(w, http.StatusForbidden, "forbidden", "Требуется роль HR")
		return
	}
	analytics, err := h.career.GetHRAnalytics(r.Context())
	if err != nil {
		writeCareerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, analytics)
}

func requireHR(w http.ResponseWriter, r *http.Request) bool {
	actor, status, code, message := actorFromRequest(r)
	if status != 0 {
		writeError(w, status, code, message)
		return false
	}
	if actor.Role != "hr" {
		if actor.Role == "employee" {
			writeError(w, http.StatusForbidden, "forbidden", "Сотрудник имеет доступ только к собственному профилю")
			return false
		}
		writeError(w, http.StatusForbidden, "forbidden", "Требуется роль HR")
		return false
	}
	return true
}

func (h Handler) completeEvent(w http.ResponseWriter, r *http.Request) {
	employeeID, ok := authorizeEmployeeResource(w, r)
	if !ok {
		return
	}
	request, ok := decodeCompletionRequest(w, r)
	if !ok {
		return
	}
	response, err := h.career.CompleteEvent(r.Context(), employeeID, request.EventID())
	if err != nil {
		writeCareerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

const maxImportSize = 32 << 20

type datasetImportResponse struct {
	Status   string                     `json:"status"`
	Imported domain.DatasetStats        `json:"imported"`
	Errors   []importer.ValidationIssue `json:"errors"`
}

func (h Handler) importDataset(w http.ResponseWriter, r *http.Request) {
	if !requireHR(w, r) {
		return
	}
	dataset, report, err := loadImportDataset(r, w)
	if err != nil {
		if errors.Is(err, importer.ErrDatasetInvalid) {
			writeJSON(w, http.StatusUnprocessableEntity, datasetImportResponse{
				Status:   "invalid",
				Imported: dataset.Stats(),
				Errors:   report.Errors,
			})
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_import", "Не удалось прочитать набор данных: "+err.Error())
		return
	}
	if err := h.store.ReplaceDataset(r.Context(), dataset); err != nil {
		writeCareerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, datasetImportResponse{
		Status:   "ok",
		Imported: dataset.Stats(),
		Errors:   make([]importer.ValidationIssue, 0),
	})
}

func loadImportDataset(r *http.Request, w http.ResponseWriter) (domain.Dataset, importer.ValidationReport, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportSize)
	if err := r.ParseMultipartForm(maxImportSize); err != nil {
		return domain.Dataset{}, importer.ValidationReport{}, err
	}
	defer r.MultipartForm.RemoveAll()

	files := r.MultipartForm.File
	if archive, found, err := readUploadedFile(files, "archive", "dataset", "dataset.zip"); err != nil {
		return domain.Dataset{}, importer.ValidationReport{}, err
	} else if found {
		return loadDatasetArchive(r.Context(), archive)
	}

	skills, found, err := readUploadedFile(files, "skills", "skills.json")
	if err != nil {
		return domain.Dataset{}, importer.ValidationReport{}, err
	}
	if !found {
		return domain.Dataset{}, importer.ValidationReport{}, errors.New("не найден файл skills.json")
	}
	events, found, err := readUploadedFile(files, "events", "events.json")
	if err != nil {
		return domain.Dataset{}, importer.ValidationReport{}, err
	}
	if !found {
		return domain.Dataset{}, importer.ValidationReport{}, errors.New("не найден файл events.json")
	}
	employees, found, err := readUploadedFile(files, "employees", "employees.json")
	if err != nil {
		return domain.Dataset{}, importer.ValidationReport{}, err
	}
	if !found {
		return domain.Dataset{}, importer.ValidationReport{}, errors.New("не найден файл employees.json")
	}
	history, found, err := readUploadedFile(files, "activity_history", "activity_history.csv", "history")
	if err != nil {
		return domain.Dataset{}, importer.ValidationReport{}, err
	}
	if !found {
		return domain.Dataset{}, importer.ValidationReport{}, errors.New("не найден файл activity_history.csv")
	}
	return importer.LoadFiles(r.Context(), bytes.NewReader(skills), bytes.NewReader(events), bytes.NewReader(employees), bytes.NewReader(history))
}

func readUploadedFile(files map[string][]*multipart.FileHeader, names ...string) ([]byte, bool, error) {
	for _, name := range names {
		headers := files[name]
		if len(headers) == 0 {
			continue
		}
		file, err := headers[0].Open()
		if err != nil {
			return nil, false, err
		}
		content, readErr := io.ReadAll(io.LimitReader(file, maxImportSize+1))
		closeErr := file.Close()
		if readErr != nil {
			return nil, false, readErr
		}
		if closeErr != nil {
			return nil, false, closeErr
		}
		if len(content) > maxImportSize {
			return nil, false, errors.New("размер файла превышает допустимый лимит")
		}
		return content, true, nil
	}
	return nil, false, nil
}

func loadDatasetArchive(ctx context.Context, content []byte) (domain.Dataset, importer.ValidationReport, error) {
	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return domain.Dataset{}, importer.ValidationReport{}, err
	}
	contents := make(map[string][]byte, 4)
	for _, file := range archive.File {
		name := path.Base(file.Name)
		if name != "skills.json" && name != "events.json" && name != "employees.json" && name != "activity_history.csv" {
			continue
		}
		if _, exists := contents[name]; exists {
			return domain.Dataset{}, importer.ValidationReport{}, errors.New("архив содержит несколько файлов " + name)
		}
		reader, err := file.Open()
		if err != nil {
			return domain.Dataset{}, importer.ValidationReport{}, err
		}
		item, readErr := io.ReadAll(io.LimitReader(reader, maxImportSize+1))
		closeErr := reader.Close()
		if readErr != nil {
			return domain.Dataset{}, importer.ValidationReport{}, readErr
		}
		if closeErr != nil {
			return domain.Dataset{}, importer.ValidationReport{}, closeErr
		}
		if len(item) > maxImportSize {
			return domain.Dataset{}, importer.ValidationReport{}, errors.New("размер распакованного файла превышает допустимый лимит")
		}
		contents[name] = item
	}
	for _, name := range []string{"skills.json", "events.json", "employees.json", "activity_history.csv"} {
		if _, exists := contents[name]; !exists {
			return domain.Dataset{}, importer.ValidationReport{}, errors.New("в архиве не найден файл " + name)
		}
	}
	return importer.LoadFiles(ctx,
		bytes.NewReader(contents["skills.json"]),
		bytes.NewReader(contents["events.json"]),
		bytes.NewReader(contents["employees.json"]),
		bytes.NewReader(contents["activity_history.csv"]),
	)
}

func authorizeEmployeeResource(w http.ResponseWriter, r *http.Request) (string, bool) {
	actor, status, code, message := actorFromRequest(r)
	if status != 0 {
		writeError(w, status, code, message)
		return "", false
	}
	employeeID := r.PathValue("id")
	if actor.Role == "employee" && actor.EmployeeID != employeeID {
		writeError(w, http.StatusForbidden, "forbidden", "Сотрудник имеет доступ только к собственному профилю")
		return "", false
	}
	return employeeID, true
}

type completionRequest struct {
	CamelEventID string `json:"eventId"`
	SnakeEventID string `json:"event_id"`
}

func (r completionRequest) EventID() string {
	if r.CamelEventID != "" {
		return r.CamelEventID
	}
	return r.SnakeEventID
}

func decodeCompletionRequest(w http.ResponseWriter, r *http.Request) (completionRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	var request completionRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Тело запроса должно содержать eventId")
		return completionRequest{}, false
	}
	if strings.TrimSpace(request.EventID()) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Укажите eventId")
		return completionRequest{}, false
	}
	return request, true
}

type actor struct {
	Role       string
	EmployeeID string
}

func actorFromRequest(r *http.Request) (actor, int, string, string) {
	role := r.Header.Get("X-Demo-Role")
	switch role {
	case "hr":
		return actor{Role: role}, 0, "", ""
	case "employee":
		employeeID := r.Header.Get("X-Employee-ID")
		if employeeID == "" {
			return actor{}, http.StatusUnauthorized, "unauthenticated", "Для роли employee требуется X-Employee-ID"
		}
		return actor{Role: role, EmployeeID: employeeID}, 0, "", ""
	case "":
		return actor{}, http.StatusUnauthorized, "unauthenticated", "Не указана demo-роль"
	default:
		return actor{}, http.StatusForbidden, "forbidden", "Неизвестная demo-роль"
	}
}

func writeCareerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrDatasetNotLoaded):
		writeError(w, http.StatusConflict, "dataset_not_loaded", "Набор данных не загружен")
	case errors.Is(err, store.ErrEmployeeNotFound):
		writeError(w, http.StatusNotFound, "employee_not_found", "Сотрудник не найден")
	case errors.Is(err, store.ErrEventNotFound):
		writeError(w, http.StatusNotFound, "event_not_found", "Мероприятие не найдено")
	case errors.Is(err, store.ErrMandatoryEvent):
		writeError(w, http.StatusUnprocessableEntity, "mandatory_event", "Обязательные мероприятия не входят в рекомендации")
	case errors.Is(err, store.ErrEventCompleted):
		writeError(w, http.StatusConflict, "event_already_completed", "Мероприятие уже завершено")
	case errors.Is(err, service.ErrPrerequisitesNotMet):
		writeError(w, http.StatusUnprocessableEntity, "event_prerequisites_not_met", "Не выполнены prerequisites мероприятия")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить операцию")
	}
}

func (h Handler) cors(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"http://localhost:3000": true,
		"http://localhost:5173": true,
		"http://localhost:8080": true,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Demo-Role, X-Employee-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			if !allowedOrigins[origin] {
				writeError(w, http.StatusForbidden, "cors_forbidden", "Источник CORS не разрешён")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h Handler) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				writeError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

// Package httpapi предоставляет REST API Career Quest и demo-разграничение ролей.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
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
	mux.HandleFunc("GET /api/v1/hr/analytics", handler.hrAnalytics)
	mux.HandleFunc("GET /api/v1/employees/{id}/profile", handler.employeeProfile)
	mux.HandleFunc("POST /api/v1/employees/{id}/complete", handler.completeEvent)
	mux.HandleFunc("POST /api/employees/{id}/complete", handler.completeEvent)
	return handler.recover(handler.cors(mux))
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

func authorizeEmployeeResource(w http.ResponseWriter, r *http.Request) (string, bool) {
	actor, status, code, message := actorFromRequest(r)
	if status != 0 {
		writeError(w, status, code, message)
		return "", false
	}
	employeeID := r.PathValue("id")
	if actor.Role == "employee" && actor.EmployeeID != employeeID {
		writeError(w, http.StatusForbidden, "forbidden", "Сотрудник может работать только со своим профилем")
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
			"details": []any{},
		},
	})
}

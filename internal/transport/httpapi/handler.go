package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/store"
)

type Handler struct {
	store store.DatasetStore
}

func NewHandler(datasetStore store.DatasetStore) http.Handler {
	handler := Handler{store: datasetStore}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.health)
	mux.HandleFunc("GET /api/v1/dataset/stats", handler.datasetStats)
	mux.HandleFunc("GET /api/v1/employees/{id}/profile", handler.employeeProfile)
	return handler.recover(mux)
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
	if r.Header.Get("X-Demo-Role") != "hr" {
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
	actor, status, code, message := actorFromRequest(r)
	if status != 0 {
		writeError(w, status, code, message)
		return
	}

	employeeID := r.PathValue("id")
	if actor.Role == "employee" && actor.EmployeeID != employeeID {
		writeError(w, http.StatusForbidden, "forbidden", "Сотрудник может просматривать только собственный профиль")
		return
	}

	employee, err := h.store.GetEmployee(r.Context(), employeeID)
	if errors.Is(err, store.ErrDatasetNotLoaded) {
		writeError(w, http.StatusConflict, "dataset_not_loaded", "Набор данных не загружен")
		return
	}
	if errors.Is(err, store.ErrEmployeeNotFound) {
		writeError(w, http.StatusNotFound, "employee_not_found", "Сотрудник не найден")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить профиль")
		return
	}

	activities, err := h.store.ListEmployeeActivities(r.Context(), employeeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить историю активностей")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"employee":   employee,
		"activities": activities,
	})
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
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
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

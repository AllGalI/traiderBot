package httpdebug

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	debugApplication "traiderBot/internalDDD/debug/application"
)

type Handler struct {
	debugService *debugApplication.DebugService
}

func NewHandler(
	debugService *debugApplication.DebugService,
) *Handler {
	return &Handler{
		debugService: debugService,
	}
}

func (h *Handler) RegisterRoutes(
	mux *http.ServeMux,
) {
	mux.HandleFunc(
		"/api/debug/providers",
		h.providers,
	)

	mux.HandleFunc(
		"/api/debug/errors",
		h.errors,
	)

	mux.HandleFunc(
		"/api/debug/provider/",
		h.provider,
	)

	mux.HandleFunc(
		"/api/debug/health",
		h.health,
	)
}

func (h *Handler) health(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (h *Handler) providers(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"providers": h.debugService.Providers(),
	})
}

func (h *Handler) errors(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	limit := 50

	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		value, err := strconv.Atoi(rawLimit)
		if err != nil || value <= 0 {
			http.Error(
				w,
				"invalid limit",
				http.StatusBadRequest,
			)
			return
		}

		if value > 500 {
			value = 500
		}

		limit = value
	}

	errors := h.debugService.Errors().List(limit)

	writeJSON(w, http.StatusOK, map[string]any{
		"errors": errors,
		"count":  len(errors),
	})
}

func (h *Handler) provider(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	name := strings.TrimPrefix(
		r.URL.Path,
		"/api/debug/provider/",
	)

	if name == "" {
		http.Error(
			w,
			"provider name is required",
			http.StatusBadRequest,
		)
		return
	}

	provider, err := h.debugService.Provider(name)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusNotFound,
		)
		return
	}

	snapshot, err := provider.Snapshot()
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	metrics, err := provider.Metrics()
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name":     provider.Name(),
		"snapshot": snapshot,
		"metrics":  metrics,
	})
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	_ = encoder.Encode(value)
}
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/amansharma/config-service/internal/model"
	"github.com/amansharma/config-service/internal/repository"
	"github.com/amansharma/config-service/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type ConfigHandler struct {
	svc    *service.ConfigService
	logger zerolog.Logger
}

func NewConfigHandler(svc *service.ConfigService, logger zerolog.Logger) *ConfigHandler {
	return &ConfigHandler{svc: svc, logger: logger.With().Str("component", "handler").Logger()}
}

func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
		return
	}

	cfg, err := h.svc.GetConfig(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "config not found"})
		return
	}
	if err != nil {
		h.logger.Error().Err(err).Str("id", id).Msg("failed to get config")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

func (h *ConfigHandler) UpsertConfig(w http.ResponseWriter, r *http.Request) {
	var req model.UpsertConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"errors": errs})
		return
	}

	cfg, created, err := h.svc.UpsertConfig(r.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Str("id", req.ID).Msg("failed to upsert config")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, cfg)
}

func (h *ConfigHandler) Ping(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func (h *ConfigHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.HealthCheck(r.Context()); err != nil {
		h.logger.Warn().Err(err).Msg("readiness check failed")
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unavailable",
			"error":  "database connection failed",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

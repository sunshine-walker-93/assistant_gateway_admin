package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"github.com/sunshine-walker-93/assistant_gateway_admin/internal/config"
)

// HistoryHandler handles configuration history API requests.
type HistoryHandler struct {
	store  config.Store
	logger *zap.Logger
}

// NewHistoryHandler creates a new HistoryHandler.
func NewHistoryHandler(store config.Store, logger *zap.Logger) *HistoryHandler {
	return &HistoryHandler{
		store:  store,
		logger: logger,
	}
}

// ListHistory returns configuration change history with optional filters.
// GET /api/v1/history?config_type=backend&config_id=1&limit=10&offset=0
func (h *HistoryHandler) ListHistory(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	var configType *string
	var configID *uint

	if typeParam := r.URL.Query().Get("config_type"); typeParam != "" {
		if typeParam != "backend" && typeParam != "route" {
			http.Error(w, "invalid config_type (must be 'backend' or 'route')", http.StatusBadRequest)
			return
		}
		configType = &typeParam
	}

	if idParam := r.URL.Query().Get("config_id"); idParam != "" {
		id, err := strconv.ParseUint(idParam, 10, 32)
		if err != nil {
			http.Error(w, "invalid config_id", http.StatusBadRequest)
			return
		}
		idUint := uint(id)
		configID = &idUint
	}

	limit := 50 // default limit
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	histories, total, err := h.store.GetHistory(configType, configID, limit, offset)
	if err != nil {
		h.logger.Error("failed to get history", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"items":  histories,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Warn("failed to encode history", zap.Error(err))
	}
}

package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/sunshine-walker-93/assistant_gateway_admin/internal/config"
)

// BackendHandler handles backend management API requests.
type BackendHandler struct {
	store  config.Store
	logger *zap.Logger
}

// NewBackendHandler creates a new BackendHandler.
func NewBackendHandler(store config.Store, logger *zap.Logger) *BackendHandler {
	return &BackendHandler{
		store:  store,
		logger: logger,
	}
}

// ListBackends returns all backends, optionally filtered by enabled status.
// GET /api/v1/backends?enabled=true
func (h *BackendHandler) ListBackends(w http.ResponseWriter, r *http.Request) {
	enabledParam := r.URL.Query().Get("enabled")
	var enabled *bool

	if enabledParam != "" {
		enabledVal, err := strconv.ParseBool(enabledParam)
		if err != nil {
			http.Error(w, "invalid enabled parameter", http.StatusBadRequest)
			return
		}
		enabled = &enabledVal
	}

	backends, err := h.store.GetBackends(enabled)
	if err != nil {
		h.logger.Error("failed to get backends", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(backends); err != nil {
		h.logger.Warn("failed to encode backends", zap.Error(err))
	}
}

// GetBackend returns a single backend by name.
// GET /api/v1/backends/{name}
func (h *BackendHandler) GetBackend(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	backend, err := h.store.GetBackendByName(name)
	if err != nil {
		h.logger.Error("failed to get backend", zap.String("name", name), zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if backend == nil {
		http.Error(w, "backend not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(backend); err != nil {
		h.logger.Warn("failed to encode backend", zap.Error(err))
	}
}

// CreateBackend creates a new backend.
// POST /api/v1/backends
func (h *BackendHandler) CreateBackend(w http.ResponseWriter, r *http.Request) {
	var backend config.Backend
	if err := json.NewDecoder(r.Body).Decode(&backend); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validation
	if backend.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if backend.Addr == "" {
		http.Error(w, "addr is required", http.StatusBadRequest)
		return
	}

	// Check if backend already exists
	existing, err := h.store.GetBackendByName(backend.Name)
	if err != nil {
		h.logger.Error("failed to check backend existence", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, "backend already exists", http.StatusConflict)
		return
	}

	// Default enabled to true
	if !r.URL.Query().Has("enabled") {
		backend.Enabled = true
	}

	// Create backend
	if err := h.store.CreateBackend(&backend); err != nil {
		h.logger.Error("failed to create backend", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Record history
	h.recordHistory("backend", &backend.ID, "CREATE", nil, &backend, r)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(backend); err != nil {
		h.logger.Warn("failed to encode backend", zap.Error(err))
	}
}

// UpdateBackend updates an existing backend.
// PUT /api/v1/backends/{name}
func (h *BackendHandler) UpdateBackend(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Get existing backend
	oldBackend, err := h.store.GetBackendByName(name)
	if err != nil {
		h.logger.Error("failed to get backend", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if oldBackend == nil {
		http.Error(w, "backend not found", http.StatusNotFound)
		return
	}

	// Parse update request - first decode to map to check if enabled field is present
	var backendUpdate map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&backendUpdate); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Check if enabled field is present in the request
	enabledPresent := false
	var enabledValue bool
	if val, ok := backendUpdate["enabled"]; ok {
		enabledPresent = true
		if boolVal, ok := val.(bool); ok {
			enabledValue = boolVal
		}
	}

	// Decode to Backend struct
	backendJSON, _ := json.Marshal(backendUpdate)
	var backend config.Backend
	if err := json.Unmarshal(backendJSON, &backend); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// If enabled field was not present in request, preserve the old value
	if !enabledPresent {
		backend.Enabled = oldBackend.Enabled
	} else {
		backend.Enabled = enabledValue
	}

	// Validation
	if backend.Addr == "" {
		http.Error(w, "addr is required", http.StatusBadRequest)
		return
	}

	// Preserve ID and name
	backend.ID = oldBackend.ID
	backend.Name = name

	// Update backend
	if err := h.store.UpdateBackend(name, &backend); err != nil {
		h.logger.Error("failed to update backend", zap.Error(err))
		if err.Error() == "backend not found" {
			http.Error(w, "backend not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Record history
	h.recordHistory("backend", &backend.ID, "UPDATE", oldBackend, &backend, r)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(backend); err != nil {
		h.logger.Warn("failed to encode backend", zap.Error(err))
	}
}

// DeleteBackend soft deletes a backend.
// DELETE /api/v1/backends/{name}
func (h *BackendHandler) DeleteBackend(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Get existing backend
	oldBackend, err := h.store.GetBackendByName(name)
	if err != nil {
		h.logger.Error("failed to get backend", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if oldBackend == nil {
		http.Error(w, "backend not found", http.StatusNotFound)
		return
	}

	// Delete backend (soft delete)
	if err := h.store.DeleteBackend(name); err != nil {
		h.logger.Error("failed to delete backend", zap.Error(err))
		if err.Error() == "backend not found" {
			http.Error(w, "backend not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Record history
	oldBackend.Enabled = false
	h.recordHistory("backend", &oldBackend.ID, "DELETE", oldBackend, nil, r)

	w.WriteHeader(http.StatusNoContent)
}

// recordHistory records a configuration change history.
func (h *BackendHandler) recordHistory(configType string, configID *uint, operation string, oldVal, newVal interface{}, r *http.Request) {
	history := &config.ConfigHistory{
		ConfigType: configType,
		ConfigID:   configID,
		Operation:  operation,
		Operator:   r.Header.Get("X-Operator"), // Future: extract from auth token
	}

	if oldVal != nil {
		if data, err := json.Marshal(oldVal); err == nil {
			history.OldValue = data
		}
	}

	if newVal != nil {
		if data, err := json.Marshal(newVal); err == nil {
			history.NewValue = data
		}
	}

	if err := h.store.CreateHistory(history); err != nil {
		h.logger.Warn("failed to record history", zap.Error(err))
	}
}

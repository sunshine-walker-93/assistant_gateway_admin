package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/sunshine-walker-93/assistant_gateway_admin/internal/config"
)

// RouteHandler handles route management API requests.
type RouteHandler struct {
	store  config.Store
	logger *zap.Logger
}

// NewRouteHandler creates a new RouteHandler.
func NewRouteHandler(store config.Store, logger *zap.Logger) *RouteHandler {
	return &RouteHandler{
		store:  store,
		logger: logger,
	}
}

// ListRoutes returns all routes, optionally filtered by enabled status.
// GET /api/v1/routes?enabled=true&limit=10&offset=0
func (h *RouteHandler) ListRoutes(w http.ResponseWriter, r *http.Request) {
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

	routes, err := h.store.GetRoutes(enabled)
	if err != nil {
		h.logger.Error("failed to get routes", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(routes); err != nil {
		h.logger.Warn("failed to encode routes", zap.Error(err))
	}
}

// GetRoute returns a single route by ID.
// GET /api/v1/routes/{id}
func (h *RouteHandler) GetRoute(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "invalid route id", http.StatusBadRequest)
		return
	}

	route, err := h.store.GetRouteByID(uint(id))
	if err != nil {
		h.logger.Error("failed to get route", zap.Uint64("id", id), zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if route == nil {
		http.Error(w, "route not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(route); err != nil {
		h.logger.Warn("failed to encode route", zap.Error(err))
	}
}

// CreateRoute creates a new route.
// POST /api/v1/routes
func (h *RouteHandler) CreateRoute(w http.ResponseWriter, r *http.Request) {
	var route config.Route
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validation
	if route.HTTPMethod == "" {
		http.Error(w, "http_method is required", http.StatusBadRequest)
		return
	}
	if route.HTTPPattern == "" {
		http.Error(w, "http_pattern is required", http.StatusBadRequest)
		return
	}
	if route.BackendName == "" {
		http.Error(w, "backend_name is required", http.StatusBadRequest)
		return
	}
	if route.BackendService == "" {
		http.Error(w, "backend_service is required", http.StatusBadRequest)
		return
	}
	if route.BackendMethod == "" {
		http.Error(w, "backend_method is required", http.StatusBadRequest)
		return
	}

	// Verify backend exists
	backend, err := h.store.GetBackendByName(route.BackendName)
	if err != nil {
		h.logger.Error("failed to check backend", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if backend == nil || !backend.Enabled {
		http.Error(w, "backend not found or disabled", http.StatusBadRequest)
		return
	}

	// Default values
	if route.TimeoutMS <= 0 {
		route.TimeoutMS = 5000
	}
	if !r.URL.Query().Has("enabled") {
		route.Enabled = true
	}

	// Create route
	if err := h.store.CreateRoute(&route); err != nil {
		h.logger.Error("failed to create route", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Record history
	h.recordHistory("route", &route.ID, "CREATE", nil, &route, r)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(route); err != nil {
		h.logger.Warn("failed to encode route", zap.Error(err))
	}
}

// UpdateRoute updates an existing route.
// PUT /api/v1/routes/{id}
func (h *RouteHandler) UpdateRoute(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "invalid route id", http.StatusBadRequest)
		return
	}

	// Get existing route
	oldRoute, err := h.store.GetRouteByID(uint(id))
	if err != nil {
		h.logger.Error("failed to get route", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if oldRoute == nil {
		http.Error(w, "route not found", http.StatusNotFound)
		return
	}

	// Parse update request - first decode to map to check if enabled field is present
	var routeUpdate map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&routeUpdate); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Check if enabled field is present in the request
	enabledPresent := false
	var enabledValue bool
	if val, ok := routeUpdate["enabled"]; ok {
		enabledPresent = true
		if boolVal, ok := val.(bool); ok {
			enabledValue = boolVal
		}
	}

	// Decode to Route struct
	routeJSON, _ := json.Marshal(routeUpdate)
	var route config.Route
	if err := json.Unmarshal(routeJSON, &route); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// If enabled field was not present in request, preserve the old value
	if !enabledPresent {
		route.Enabled = oldRoute.Enabled
	} else {
		route.Enabled = enabledValue
	}

	// Validation
	if route.HTTPMethod == "" || route.HTTPPattern == "" || route.BackendName == "" ||
		route.BackendService == "" || route.BackendMethod == "" {
		http.Error(w, "required fields cannot be empty", http.StatusBadRequest)
		return
	}

	// Verify backend exists if changed
	if route.BackendName != oldRoute.BackendName {
		backend, err := h.store.GetBackendByName(route.BackendName)
		if err != nil {
			h.logger.Error("failed to check backend", zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if backend == nil || !backend.Enabled {
			http.Error(w, "backend not found or disabled", http.StatusBadRequest)
			return
		}
	}

	// Preserve ID
	route.ID = uint(id)

	// Update route
	if err := h.store.UpdateRoute(uint(id), &route); err != nil {
		h.logger.Error("failed to update route", zap.Error(err))
		if err.Error() == "route not found" {
			http.Error(w, "route not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Record history
	h.recordHistory("route", &route.ID, "UPDATE", oldRoute, &route, r)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(route); err != nil {
		h.logger.Warn("failed to encode route", zap.Error(err))
	}
}

// DeleteRoute soft deletes a route.
// DELETE /api/v1/routes/{id}
func (h *RouteHandler) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "invalid route id", http.StatusBadRequest)
		return
	}

	// Get existing route
	oldRoute, err := h.store.GetRouteByID(uint(id))
	if err != nil {
		h.logger.Error("failed to get route", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if oldRoute == nil {
		http.Error(w, "route not found", http.StatusNotFound)
		return
	}

	// Delete route (soft delete)
	if err := h.store.DeleteRoute(uint(id)); err != nil {
		h.logger.Error("failed to delete route", zap.Error(err))
		if err.Error() == "route not found" {
			http.Error(w, "route not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Record history
	oldRoute.Enabled = false
	h.recordHistory("route", &oldRoute.ID, "DELETE", oldRoute, nil, r)

	w.WriteHeader(http.StatusNoContent)
}

// recordHistory records a configuration change history.
func (h *RouteHandler) recordHistory(configType string, configID *uint, operation string, oldVal, newVal interface{}, r *http.Request) {
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

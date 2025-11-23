package config

import (
	"encoding/json"
	"time"
)

// Backend represents a backend service configuration.
type Backend struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Addr        string    `json:"addr"`
	Description string    `json:"description,omitempty"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Route represents a route configuration.
type Route struct {
	ID             uint      `json:"id"`
	HTTPMethod     string    `json:"http_method"`
	HTTPPattern    string    `json:"http_pattern"`
	BackendName    string    `json:"backend_name"`
	BackendService string    `json:"backend_service"`
	BackendMethod  string    `json:"backend_method"`
	TimeoutMS      int       `json:"timeout_ms"`
	Description    string    `json:"description,omitempty"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ConfigHistory represents a configuration change history record.
type ConfigHistory struct {
	ID         uint64          `json:"id"`
	ConfigType string          `json:"config_type"` // "backend" or "route"
	ConfigID   *uint           `json:"config_id,omitempty"`
	Operation  string          `json:"operation"` // "CREATE", "UPDATE", "DELETE"
	OldValue   json.RawMessage `json:"old_value,omitempty"`
	NewValue   json.RawMessage `json:"new_value,omitempty"`
	Operator   string          `json:"operator,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// Store defines the interface for configuration storage operations.
type Store interface {
	// Backend operations
	GetBackends(enabled *bool) ([]Backend, error)
	GetBackendByName(name string) (*Backend, error)
	CreateBackend(backend *Backend) error
	UpdateBackend(name string, backend *Backend) error
	DeleteBackend(name string) error

	// Route operations
	GetRoutes(enabled *bool) ([]Route, error)
	GetRouteByID(id uint) (*Route, error)
	CreateRoute(route *Route) error
	UpdateRoute(id uint, route *Route) error
	DeleteRoute(id uint) error

	// History operations
	CreateHistory(history *ConfigHistory) error
	GetHistory(configType *string, configID *uint, limit, offset int) ([]ConfigHistory, int, error)
}

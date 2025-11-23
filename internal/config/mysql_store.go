package config

import (
	"database/sql"
	"errors"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLStore implements Store using MySQL database.
type MySQLStore struct {
	db *sql.DB
}

// NewMySQLStore creates a new MySQLStore instance.
func NewMySQLStore(dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &MySQLStore{db: db}, nil
}

// Close closes the database connection.
func (s *MySQLStore) Close() error {
	return s.db.Close()
}

// GetBackends returns all backend configurations, optionally filtered by enabled status.
func (s *MySQLStore) GetBackends(enabled *bool) ([]Backend, error) {
	var query string
	var args []interface{}

	if enabled != nil {
		query = `SELECT id, name, addr, description, enabled, created_at, updated_at 
		         FROM backends WHERE enabled = ? ORDER BY name`
		args = []interface{}{*enabled}
	} else {
		query = `SELECT id, name, addr, description, enabled, created_at, updated_at 
		         FROM backends ORDER BY name`
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backends []Backend
	for rows.Next() {
		var b Backend
		var enabledInt int
		var desc sql.NullString

		if err := rows.Scan(&b.ID, &b.Name, &b.Addr, &desc, &enabledInt, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}

		if desc.Valid {
			b.Description = desc.String
		}
		b.Enabled = enabledInt == 1

		backends = append(backends, b)
	}

	return backends, rows.Err()
}

// GetBackendByName returns a backend configuration by name.
func (s *MySQLStore) GetBackendByName(name string) (*Backend, error) {
	query := `SELECT id, name, addr, description, enabled, created_at, updated_at 
	          FROM backends WHERE name = ? LIMIT 1`

	var b Backend
	var enabledInt int
	var desc sql.NullString

	err := s.db.QueryRow(query, name).Scan(
		&b.ID, &b.Name, &b.Addr, &desc, &enabledInt, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if desc.Valid {
		b.Description = desc.String
	}
	b.Enabled = enabledInt == 1

	return &b, nil
}

// CreateBackend creates a new backend configuration.
func (s *MySQLStore) CreateBackend(backend *Backend) error {
	query := `INSERT INTO backends (name, addr, description, enabled) 
	          VALUES (?, ?, ?, ?)`

	enabledInt := 0
	if backend.Enabled {
		enabledInt = 1
	}

	result, err := s.db.Exec(query, backend.Name, backend.Addr, backend.Description, enabledInt)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	backend.ID = uint(id)
	backend.CreatedAt = time.Now()
	backend.UpdatedAt = time.Now()

	return nil
}

// UpdateBackend updates an existing backend configuration.
func (s *MySQLStore) UpdateBackend(name string, backend *Backend) error {
	query := `UPDATE backends 
	          SET addr = ?, description = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP 
	          WHERE name = ?`

	enabledInt := 0
	if backend.Enabled {
		enabledInt = 1
	}

	result, err := s.db.Exec(query, backend.Addr, backend.Description, enabledInt, name)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("backend not found")
	}

	backend.Name = name
	backend.UpdatedAt = time.Now()

	return nil
}

// DeleteBackend soft deletes a backend by setting enabled=0.
func (s *MySQLStore) DeleteBackend(name string) error {
	query := `UPDATE backends SET enabled = 0, updated_at = CURRENT_TIMESTAMP WHERE name = ?`

	result, err := s.db.Exec(query, name)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("backend not found")
	}

	return nil
}

// GetRoutes returns all route configurations, optionally filtered by enabled status.
func (s *MySQLStore) GetRoutes(enabled *bool) ([]Route, error) {
	var query string
	var args []interface{}

	if enabled != nil {
		query = `SELECT id, http_method, http_pattern, backend_name, backend_service, 
		                backend_method, timeout_ms, description, enabled, created_at, updated_at 
		         FROM routes WHERE enabled = ? ORDER BY http_method, http_pattern`
		args = []interface{}{*enabled}
	} else {
		query = `SELECT id, http_method, http_pattern, backend_name, backend_service, 
		                backend_method, timeout_ms, description, enabled, created_at, updated_at 
		         FROM routes ORDER BY http_method, http_pattern`
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []Route
	for rows.Next() {
		var r Route
		var enabledInt int
		var desc sql.NullString

		if err := rows.Scan(
			&r.ID, &r.HTTPMethod, &r.HTTPPattern, &r.BackendName, &r.BackendService,
			&r.BackendMethod, &r.TimeoutMS, &desc, &enabledInt, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if desc.Valid {
			r.Description = desc.String
		}
		r.Enabled = enabledInt == 1

		routes = append(routes, r)
	}

	return routes, rows.Err()
}

// GetRouteByID returns a route configuration by ID.
func (s *MySQLStore) GetRouteByID(id uint) (*Route, error) {
	query := `SELECT id, http_method, http_pattern, backend_name, backend_service, 
	                 backend_method, timeout_ms, description, enabled, created_at, updated_at 
	          FROM routes WHERE id = ? LIMIT 1`

	var r Route
	var enabledInt int
	var desc sql.NullString

	err := s.db.QueryRow(query, id).Scan(
		&r.ID, &r.HTTPMethod, &r.HTTPPattern, &r.BackendName, &r.BackendService,
		&r.BackendMethod, &r.TimeoutMS, &desc, &enabledInt, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if desc.Valid {
		r.Description = desc.String
	}
	r.Enabled = enabledInt == 1

	return &r, nil
}

// CreateRoute creates a new route configuration.
func (s *MySQLStore) CreateRoute(route *Route) error {
	query := `INSERT INTO routes (http_method, http_pattern, backend_name, backend_service, 
	                              backend_method, timeout_ms, description, enabled) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	enabledInt := 0
	if route.Enabled {
		enabledInt = 1
	}

	result, err := s.db.Exec(
		query, route.HTTPMethod, route.HTTPPattern, route.BackendName,
		route.BackendService, route.BackendMethod, route.TimeoutMS,
		route.Description, enabledInt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	route.ID = uint(id)
	route.CreatedAt = time.Now()
	route.UpdatedAt = time.Now()

	return nil
}

// UpdateRoute updates an existing route configuration.
func (s *MySQLStore) UpdateRoute(id uint, route *Route) error {
	query := `UPDATE routes 
	          SET http_method = ?, http_pattern = ?, backend_name = ?, backend_service = ?, 
	              backend_method = ?, timeout_ms = ?, description = ?, enabled = ?, 
	              updated_at = CURRENT_TIMESTAMP 
	          WHERE id = ?`

	enabledInt := 0
	if route.Enabled {
		enabledInt = 1
	}

	result, err := s.db.Exec(
		query, route.HTTPMethod, route.HTTPPattern, route.BackendName,
		route.BackendService, route.BackendMethod, route.TimeoutMS,
		route.Description, enabledInt, id,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("route not found")
	}

	route.ID = id
	route.UpdatedAt = time.Now()

	return nil
}

// DeleteRoute soft deletes a route by setting enabled=0.
func (s *MySQLStore) DeleteRoute(id uint) error {
	query := `UPDATE routes SET enabled = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("route not found")
	}

	return nil
}

// CreateHistory creates a new configuration change history record.
func (s *MySQLStore) CreateHistory(history *ConfigHistory) error {
	query := `INSERT INTO config_history (config_type, config_id, operation, old_value, new_value, operator) 
	          VALUES (?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(
		query, history.ConfigType, history.ConfigID, history.Operation,
		history.OldValue, history.NewValue, history.Operator,
	)
	return err
}

// GetHistory returns configuration change history with optional filters.
func (s *MySQLStore) GetHistory(configType *string, configID *uint, limit, offset int) ([]ConfigHistory, int, error) {
	// Build WHERE clause
	where := "1=1"
	args := []interface{}{}

	if configType != nil {
		where += " AND config_type = ?"
		args = append(args, *configType)
	}

	if configID != nil {
		where += " AND config_id = ?"
		args = append(args, *configID)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM config_history WHERE " + where
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := `SELECT id, config_type, config_id, operation, old_value, new_value, operator, created_at 
	          FROM config_history WHERE ` + where + ` 
	          ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var histories []ConfigHistory
	for rows.Next() {
		var h ConfigHistory
		var configIDPtr *uint

		if err := rows.Scan(
			&h.ID, &h.ConfigType, &configIDPtr, &h.Operation,
			&h.OldValue, &h.NewValue, &h.Operator, &h.CreatedAt,
		); err != nil {
			return nil, 0, err
		}

		if configIDPtr != nil {
			h.ConfigID = configIDPtr
		}

		histories = append(histories, h)
	}

	return histories, total, rows.Err()
}

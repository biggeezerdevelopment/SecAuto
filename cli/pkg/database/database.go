package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the database connection
type DB struct {
	conn *sql.DB
	path string
}

// Server represents a SecAuto server configuration
type Server struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	APIKey      string    `json:"api_key"`
	Description string    `json:"description"`
	IsDefault   bool      `json:"is_default"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Execution represents an execution record
type Execution struct {
	ID         int       `json:"id"`
	ServerID   int       `json:"server_id"`
	ServerName string    `json:"server_name"`
	Type       string    `json:"type"` // playbook, automation, integration
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Result     string    `json:"result"`
	Error      string    `json:"error,omitempty"`
	ExecutedAt time.Time `json:"executed_at"`
}

// SyncStatus represents synchronization status for a resource
type SyncStatus struct {
	ID         int       `json:"id"`
	ServerID   int       `json:"server_id"`
	Type       string    `json:"type"` // playbook, automation, integration
	Name       string    `json:"name"`
	Hash       string    `json:"hash"`
	SyncedAt   time.Time `json:"synced_at"`
	Status     string    `json:"status"`
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	// Open database connection
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	db := &DB{
		conn: conn,
		path: dbPath,
	}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %v", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// initSchema creates the database tables if they don't exist
func (db *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS servers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		url TEXT NOT NULL,
		api_key TEXT NOT NULL,
		description TEXT,
		is_default BOOLEAN DEFAULT 0,
		is_active BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS executions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server_id INTEGER NOT NULL,
		server_name TEXT NOT NULL,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		status TEXT NOT NULL,
		result TEXT,
		error TEXT,
		executed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS sync_status (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server_id INTEGER NOT NULL,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		hash TEXT NOT NULL,
		synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		status TEXT NOT NULL,
		UNIQUE(server_id, type, name),
		FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_executions_server_id ON executions(server_id);
	CREATE INDEX IF NOT EXISTS idx_executions_type ON executions(type);
	CREATE INDEX IF NOT EXISTS idx_sync_status_server_id ON sync_status(server_id);
	CREATE INDEX IF NOT EXISTS idx_sync_status_type ON sync_status(type);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// AddServer adds a new server configuration
func (db *DB) AddServer(server *Server) error {
	// If this is the first server or marked as default, unset other defaults
	if server.IsDefault {
		if _, err := db.conn.Exec("UPDATE servers SET is_default = 0"); err != nil {
			return fmt.Errorf("failed to unset default servers: %v", err)
		}
	}

	query := `
		INSERT INTO servers (name, url, api_key, description, is_default, is_active)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query,
		server.Name, server.URL, server.APIKey, server.Description,
		server.IsDefault, server.IsActive)
	if err != nil {
		return fmt.Errorf("failed to add server: %v", err)
	}

	id, _ := result.LastInsertId()
	server.ID = int(id)
	return nil
}

// UpdateServer updates an existing server configuration
func (db *DB) UpdateServer(server *Server) error {
	// If setting as default, unset other defaults
	if server.IsDefault {
		if _, err := db.conn.Exec("UPDATE servers SET is_default = 0 WHERE id != ?", server.ID); err != nil {
			return fmt.Errorf("failed to unset default servers: %v", err)
		}
	}

	query := `
		UPDATE servers 
		SET name = ?, url = ?, api_key = ?, description = ?, 
		    is_default = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := db.conn.Exec(query,
		server.Name, server.URL, server.APIKey, server.Description,
		server.IsDefault, server.IsActive, server.ID)
	if err != nil {
		return fmt.Errorf("failed to update server: %v", err)
	}

	return nil
}

// DeleteServer deletes a server configuration
func (db *DB) DeleteServer(id int) error {
	_, err := db.conn.Exec("DELETE FROM servers WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete server: %v", err)
	}
	return nil
}

// GetServer retrieves a server by ID
func (db *DB) GetServer(id int) (*Server, error) {
	query := `
		SELECT id, name, url, api_key, description, is_default, is_active, created_at, updated_at
		FROM servers WHERE id = ?
	`

	var server Server
	err := db.conn.QueryRow(query, id).Scan(
		&server.ID, &server.Name, &server.URL, &server.APIKey,
		&server.Description, &server.IsDefault, &server.IsActive,
		&server.CreatedAt, &server.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("server not found")
		}
		return nil, fmt.Errorf("failed to get server: %v", err)
	}

	return &server, nil
}

// GetServerByName retrieves a server by name
func (db *DB) GetServerByName(name string) (*Server, error) {
	query := `
		SELECT id, name, url, api_key, description, is_default, is_active, created_at, updated_at
		FROM servers WHERE name = ?
	`

	var server Server
	err := db.conn.QueryRow(query, name).Scan(
		&server.ID, &server.Name, &server.URL, &server.APIKey,
		&server.Description, &server.IsDefault, &server.IsActive,
		&server.CreatedAt, &server.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("server not found")
		}
		return nil, fmt.Errorf("failed to get server: %v", err)
	}

	return &server, nil
}

// GetDefaultServer retrieves the default server
func (db *DB) GetDefaultServer() (*Server, error) {
	query := `
		SELECT id, name, url, api_key, description, is_default, is_active, created_at, updated_at
		FROM servers WHERE is_default = 1 LIMIT 1
	`

	var server Server
	err := db.conn.QueryRow(query).Scan(
		&server.ID, &server.Name, &server.URL, &server.APIKey,
		&server.Description, &server.IsDefault, &server.IsActive,
		&server.CreatedAt, &server.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no default server configured")
		}
		return nil, fmt.Errorf("failed to get default server: %v", err)
	}

	return &server, nil
}

// ListServers retrieves all servers
func (db *DB) ListServers(activeOnly bool) ([]*Server, error) {
	query := `
		SELECT id, name, url, api_key, description, is_default, is_active, created_at, updated_at
		FROM servers
	`
	if activeOnly {
		query += " WHERE is_active = 1"
	}
	query += " ORDER BY is_default DESC, name ASC"

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %v", err)
	}
	defer rows.Close()

	var servers []*Server
	for rows.Next() {
		var server Server
		err := rows.Scan(
			&server.ID, &server.Name, &server.URL, &server.APIKey,
			&server.Description, &server.IsDefault, &server.IsActive,
			&server.CreatedAt, &server.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan server: %v", err)
		}
		servers = append(servers, &server)
	}

	return servers, nil
}

// GetActiveServers retrieves all active servers
func (db *DB) GetActiveServers() ([]*Server, error) {
	return db.ListServers(true)
}

// RecordExecution records an execution result
func (db *DB) RecordExecution(exec *Execution) error {
	query := `
		INSERT INTO executions (server_id, server_name, type, name, status, result, error)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query,
		exec.ServerID, exec.ServerName, exec.Type, exec.Name,
		exec.Status, exec.Result, exec.Error)
	if err != nil {
		return fmt.Errorf("failed to record execution: %v", err)
	}

	id, _ := result.LastInsertId()
	exec.ID = int(id)
	return nil
}

// GetExecutionHistory retrieves execution history
func (db *DB) GetExecutionHistory(serverID int, limit int) ([]*Execution, error) {
	query := `
		SELECT id, server_id, server_name, type, name, status, result, error, executed_at
		FROM executions
	`
	args := []interface{}{}

	if serverID > 0 {
		query += " WHERE server_id = ?"
		args = append(args, serverID)
	}

	query += " ORDER BY executed_at DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution history: %v", err)
	}
	defer rows.Close()

	var executions []*Execution
	for rows.Next() {
		var exec Execution
		var errorStr sql.NullString
		err := rows.Scan(
			&exec.ID, &exec.ServerID, &exec.ServerName, &exec.Type,
			&exec.Name, &exec.Status, &exec.Result, &errorStr, &exec.ExecutedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %v", err)
		}
		if errorStr.Valid {
			exec.Error = errorStr.String
		}
		executions = append(executions, &exec)
	}

	return executions, nil
}

// UpdateSyncStatus updates the sync status for a resource
func (db *DB) UpdateSyncStatus(sync *SyncStatus) error {
	query := `
		INSERT OR REPLACE INTO sync_status (server_id, type, name, hash, status, synced_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err := db.conn.Exec(query,
		sync.ServerID, sync.Type, sync.Name, sync.Hash, sync.Status)
	if err != nil {
		return fmt.Errorf("failed to update sync status: %v", err)
	}

	return nil
}

// GetSyncStatus gets the sync status for a resource
func (db *DB) GetSyncStatus(serverID int, resourceType, name string) (*SyncStatus, error) {
	query := `
		SELECT id, server_id, type, name, hash, synced_at, status
		FROM sync_status
		WHERE server_id = ? AND type = ? AND name = ?
	`

	var sync SyncStatus
	err := db.conn.QueryRow(query, serverID, resourceType, name).Scan(
		&sync.ID, &sync.ServerID, &sync.Type, &sync.Name,
		&sync.Hash, &sync.SyncedAt, &sync.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not synced yet
		}
		return nil, fmt.Errorf("failed to get sync status: %v", err)
	}

	return &sync, nil
}

// NeedsSyncing checks if a resource needs syncing based on hash
func (db *DB) NeedsSyncing(serverID int, resourceType, name, hash string) (bool, error) {
	sync, err := db.GetSyncStatus(serverID, resourceType, name)
	if err != nil {
		return false, err
	}

	// If never synced or hash changed, needs syncing
	if sync == nil || sync.Hash != hash {
		return true, nil
	}

	return false, nil
}

// GetExecutionStats gets execution statistics
func (db *DB) GetExecutionStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total executions
	var total int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM executions").Scan(&total)
	if err != nil {
		return nil, err
	}
	stats["total_executions"] = total

	// By status
	statusQuery := `
		SELECT status, COUNT(*) as count 
		FROM executions 
		GROUP BY status
	`
	rows, err := db.conn.Query(statusQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statusCounts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		statusCounts[status] = count
	}
	stats["by_status"] = statusCounts

	// By type
	typeQuery := `
		SELECT type, COUNT(*) as count 
		FROM executions 
		GROUP BY type
	`
	rows2, err := db.conn.Query(typeQuery)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	typeCounts := make(map[string]int)
	for rows2.Next() {
		var execType string
		var count int
		if err := rows2.Scan(&execType, &count); err != nil {
			return nil, err
		}
		typeCounts[execType] = count
	}
	stats["by_type"] = typeCounts

	// By server
	serverQuery := `
		SELECT server_name, COUNT(*) as count 
		FROM executions 
		GROUP BY server_id, server_name
	`
	rows3, err := db.conn.Query(serverQuery)
	if err != nil {
		return nil, err
	}
	defer rows3.Close()

	serverCounts := make(map[string]int)
	for rows3.Next() {
		var serverName string
		var count int
		if err := rows3.Scan(&serverName, &count); err != nil {
			return nil, err
		}
		serverCounts[serverName] = count
	}
	stats["by_server"] = serverCounts

	return stats, nil
}

// ClearExecutionHistory clears execution history older than days
func (db *DB) ClearExecutionHistory(days int) error {
	query := "DELETE FROM executions WHERE executed_at < datetime('now', '-' || ? || ' days')"
	_, err := db.conn.Exec(query, days)
	if err != nil {
		return fmt.Errorf("failed to clear execution history: %v", err)
	}
	return nil
}
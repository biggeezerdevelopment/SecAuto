package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"SoarAuto/pkg/types"

	"github.com/lib/pq"
)

// ClientRepository handles database operations for clients
type ClientRepository struct {
	db     *sql.DB
	logger types.Logger
}

// NewClientRepository creates a new client repository
func NewClientRepository(db *sql.DB, logger types.Logger) *ClientRepository {
	return &ClientRepository{
		db:     db,
		logger: logger,
	}
}

// Client represents a client in the database
type Client struct {
	ID                      string                 `json:"id"`
	Name                    string                 `json:"name"`
	Description             string                 `json:"description"`
	Active                  bool                   `json:"active"`
	APIKeys                 []string               `json:"api_keys"`
	EncryptionKeyID         string                 `json:"encryption_key_id"`
	RateLimitRequestsPerMin int                    `json:"rate_limit_requests_per_minute"`
	RateLimitBurstLimit     int                    `json:"rate_limit_burst_limit"`
	RateLimitEnabled        bool                   `json:"rate_limit_enabled"`
	Metadata                map[string]interface{} `json:"metadata"`
	CreatedAt               time.Time              `json:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at"`
	LastAccessedAt          *time.Time             `json:"last_accessed_at,omitempty"`
	IntegrationCount        int                    `json:"integration_count"`
}

// CreateClient creates a new client in the database
func (cr *ClientRepository) CreateClient(client *Client) error {
	query := `
		INSERT INTO clients (
			id, name, description, active, api_keys, encryption_key_id,
			rate_limit_requests_per_minute, rate_limit_burst_limit, rate_limit_enabled,
			metadata, created_at, updated_at, integration_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	metadataJSON, err := json.Marshal(client.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	_, err = cr.db.Exec(query,
		client.ID,
		client.Name,
		client.Description,
		client.Active,
		pq.Array(client.APIKeys),
		client.EncryptionKeyID,
		client.RateLimitRequestsPerMin,
		client.RateLimitBurstLimit,
		client.RateLimitEnabled,
		metadataJSON,
		client.CreatedAt,
		client.UpdatedAt,
		client.IntegrationCount,
	)

	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	cr.logger.Info("Client created in database", map[string]interface{}{
		"component": "database",
		"client_id": client.ID,
		"name":      client.Name,
	})

	return nil
}

// GetClient retrieves a client by ID
func (cr *ClientRepository) GetClient(clientID string) (*Client, error) {
	query := `
		SELECT id, name, description, active, api_keys, encryption_key_id,
			   rate_limit_requests_per_minute, rate_limit_burst_limit, rate_limit_enabled,
			   metadata, created_at, updated_at, last_accessed_at, integration_count
		FROM clients 
		WHERE id = $1`

	row := cr.db.QueryRow(query, clientID)

	client := &Client{}
	var metadataJSON []byte
	var apiKeys pq.StringArray

	err := row.Scan(
		&client.ID,
		&client.Name,
		&client.Description,
		&client.Active,
		&apiKeys,
		&client.EncryptionKeyID,
		&client.RateLimitRequestsPerMin,
		&client.RateLimitBurstLimit,
		&client.RateLimitEnabled,
		&metadataJSON,
		&client.CreatedAt,
		&client.UpdatedAt,
		&client.LastAccessedAt,
		&client.IntegrationCount,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("client not found: %s", clientID)
		}
		return nil, fmt.Errorf("failed to get client: %v", err)
	}

	client.APIKeys = []string(apiKeys)

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &client.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %v", err)
		}
	} else {
		client.Metadata = make(map[string]interface{})
	}

	return client, nil
}

// ListClients retrieves all clients with optional filtering
func (cr *ClientRepository) ListClients(activeOnly bool, limit, offset int) ([]*Client, error) {
	query := `
		SELECT id, name, description, active, api_keys, encryption_key_id,
			   rate_limit_requests_per_minute, rate_limit_burst_limit, rate_limit_enabled,
			   metadata, created_at, updated_at, last_accessed_at, integration_count
		FROM clients`

	args := []interface{}{}
	argIndex := 1

	if activeOnly {
		query += " WHERE active = $" + fmt.Sprintf("%d", argIndex)
		args = append(args, true)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += " LIMIT $" + fmt.Sprintf("%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	if offset > 0 {
		query += " OFFSET $" + fmt.Sprintf("%d", argIndex)
		args = append(args, offset)
	}

	rows, err := cr.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %v", err)
	}
	defer rows.Close()

	var clients []*Client
	for rows.Next() {
		client := &Client{}
		var metadataJSON []byte
		var apiKeys pq.StringArray

		err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.Description,
			&client.Active,
			&apiKeys,
			&client.EncryptionKeyID,
			&client.RateLimitRequestsPerMin,
			&client.RateLimitBurstLimit,
			&client.RateLimitEnabled,
			&metadataJSON,
			&client.CreatedAt,
			&client.UpdatedAt,
			&client.LastAccessedAt,
			&client.IntegrationCount,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan client: %v", err)
		}

		client.APIKeys = []string(apiKeys)

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &client.Metadata); err != nil {
				cr.logger.Warning("Failed to unmarshal client metadata", map[string]interface{}{
					"client_id": client.ID,
					"error":     err.Error(),
				})
				client.Metadata = make(map[string]interface{})
			}
		} else {
			client.Metadata = make(map[string]interface{})
		}

		clients = append(clients, client)
	}

	return clients, nil
}

// UpdateClient updates an existing client
func (cr *ClientRepository) UpdateClient(clientID string, updates map[string]interface{}) error {
	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if name, ok := updates["name"].(string); ok {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, name)
		argIndex++
	}

	if description, ok := updates["description"].(string); ok {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, description)
		argIndex++
	}

	if active, ok := updates["active"].(bool); ok {
		setParts = append(setParts, fmt.Sprintf("active = $%d", argIndex))
		args = append(args, active)
		argIndex++
	}

	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %v", err)
		}
		setParts = append(setParts, fmt.Sprintf("metadata = $%d", argIndex))
		args = append(args, metadataJSON)
		argIndex++
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no valid updates provided")
	}

	// Always update the timestamp
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add client ID as the final parameter
	args = append(args, clientID)

	query := fmt.Sprintf("UPDATE clients SET %s WHERE id = $%d",
		fmt.Sprintf("%s", setParts[0]), argIndex)

	for i := 1; i < len(setParts); i++ {
		query = fmt.Sprintf("%s, %s", query, setParts[i])
	}

	result, err := cr.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update client: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("client not found: %s", clientID)
	}

	cr.logger.Info("Client updated in database", map[string]interface{}{
		"component": "database",
		"client_id": clientID,
		"updates":   updates,
	})

	return nil
}

// DeleteClient deletes a client from the database
func (cr *ClientRepository) DeleteClient(clientID string) error {
	query := `DELETE FROM clients WHERE id = $1`

	result, err := cr.db.Exec(query, clientID)
	if err != nil {
		return fmt.Errorf("failed to delete client: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("client not found: %s", clientID)
	}

	cr.logger.Info("Client deleted from database", map[string]interface{}{
		"component": "database",
		"client_id": clientID,
	})

	return nil
}

// GetClientByAPIKey finds a client by their API key
func (cr *ClientRepository) GetClientByAPIKey(apiKey string) (*Client, error) {
	query := `
		SELECT id, name, description, active, api_keys, encryption_key_id,
			   rate_limit_requests_per_minute, rate_limit_burst_limit, rate_limit_enabled,
			   metadata, created_at, updated_at, last_accessed_at, integration_count
		FROM clients 
		WHERE $1 = ANY(api_keys) AND active = true`

	row := cr.db.QueryRow(query, apiKey)

	client := &Client{}
	var metadataJSON []byte
	var apiKeys pq.StringArray

	err := row.Scan(
		&client.ID,
		&client.Name,
		&client.Description,
		&client.Active,
		&apiKeys,
		&client.EncryptionKeyID,
		&client.RateLimitRequestsPerMin,
		&client.RateLimitBurstLimit,
		&client.RateLimitEnabled,
		&metadataJSON,
		&client.CreatedAt,
		&client.UpdatedAt,
		&client.LastAccessedAt,
		&client.IntegrationCount,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no client found for API key")
		}
		return nil, fmt.Errorf("failed to get client by API key: %v", err)
	}

	client.APIKeys = []string(apiKeys)

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &client.Metadata); err != nil {
			client.Metadata = make(map[string]interface{})
		}
	} else {
		client.Metadata = make(map[string]interface{})
	}

	// Update last accessed time
	_, err = cr.db.Exec("UPDATE clients SET last_accessed_at = $1 WHERE id = $2",
		time.Now(), client.ID)
	if err != nil {
		cr.logger.Warning("Failed to update last accessed time", map[string]interface{}{
			"client_id": client.ID,
			"error":     err.Error(),
		})
	}

	return client, nil
}

// UpdateIntegrationCount updates the integration count for a client
func (cr *ClientRepository) UpdateIntegrationCount(clientID string, count int) error {
	query := `UPDATE clients SET integration_count = $1, updated_at = $2 WHERE id = $3`

	_, err := cr.db.Exec(query, count, time.Now(), clientID)
	if err != nil {
		return fmt.Errorf("failed to update integration count: %v", err)
	}

	return nil
}

// SearchClients searches clients by name or metadata
func (cr *ClientRepository) SearchClients(searchTerm string, limit int) ([]*Client, error) {
	query := `
		SELECT id, name, description, active, api_keys, encryption_key_id,
			   rate_limit_requests_per_minute, rate_limit_burst_limit, rate_limit_enabled,
			   metadata, created_at, updated_at, last_accessed_at, integration_count
		FROM clients 
		WHERE name ILIKE $1 OR description ILIKE $1 OR metadata::text ILIKE $1
		ORDER BY 
			CASE WHEN name ILIKE $1 THEN 1 ELSE 2 END,
			created_at DESC
		LIMIT $2`

	searchPattern := "%" + searchTerm + "%"
	rows, err := cr.db.Query(query, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search clients: %v", err)
	}
	defer rows.Close()

	var clients []*Client
	for rows.Next() {
		client := &Client{}
		var metadataJSON []byte
		var apiKeys pq.StringArray

		err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.Description,
			&client.Active,
			&apiKeys,
			&client.EncryptionKeyID,
			&client.RateLimitRequestsPerMin,
			&client.RateLimitBurstLimit,
			&client.RateLimitEnabled,
			&metadataJSON,
			&client.CreatedAt,
			&client.UpdatedAt,
			&client.LastAccessedAt,
			&client.IntegrationCount,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan client: %v", err)
		}

		client.APIKeys = []string(apiKeys)

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &client.Metadata); err != nil {
				client.Metadata = make(map[string]interface{})
			}
		} else {
			client.Metadata = make(map[string]interface{})
		}

		clients = append(clients, client)
	}

	return clients, nil
}

package clients

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"SoarAuto/pkg/database"
	"SoarAuto/pkg/security"
	"SoarAuto/pkg/types"
)

// DatabaseClientManager manages client configurations using PostgreSQL database
type DatabaseClientManager struct {
	repository *database.ClientRepository
	logger     types.Logger
	encryptor  *security.Encryptor
}

// NewDatabaseClientManager creates a new database-based client manager
func NewDatabaseClientManager(db *sql.DB, logger types.Logger) (*DatabaseClientManager, error) {
	repository := database.NewClientRepository(db, logger)

	// Initialize encryptor for client-specific encryption
	encryptor, err := security.NewEncryptor()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize encryptor: %v", err)
	}

	return &DatabaseClientManager{
		repository: repository,
		logger:     logger,
		encryptor:  encryptor,
	}, nil
}

// ListClients returns all clients
func (dcm *DatabaseClientManager) ListClients() []*Client {
	dbClients, err := dcm.repository.ListClients(false, 0, 0)
	if err != nil {
		dcm.logger.Error("Failed to list clients from database", map[string]interface{}{
			"component": "clients",
			"error":     err.Error(),
		})
		return []*Client{} // Return empty slice on error
	}

	clients := make([]*Client, 0, len(dbClients))
	for _, dbClient := range dbClients {
		client := dcm.convertFromDB(dbClient)
		clients = append(clients, client)
	}

	return clients
}

// GetClient retrieves a specific client
func (dcm *DatabaseClientManager) GetClient(clientID string) (*Client, error) {
	dbClient, err := dcm.repository.GetClient(clientID)
	if err != nil {
		return nil, err
	}

	return dcm.convertFromDB(dbClient), nil
}

// CreateClient creates a new client
func (dcm *DatabaseClientManager) CreateClient(name, description string) (*Client, error) {
	// Generate unique client ID
	clientID := dcm.generateClientID()

	// Generate API key for the client
	apiKey := dcm.generateAPIKey()

	// Generate encryption key ID for client-specific encryption
	encryptionKeyID := dcm.generateEncryptionKeyID()

	// Create database client
	dbClient := &database.Client{
		ID:                      clientID,
		Name:                    name,
		Description:             description,
		Active:                  true,
		APIKeys:                 []string{apiKey},
		EncryptionKeyID:         encryptionKeyID,
		RateLimitRequestsPerMin: 100,
		RateLimitBurstLimit:     20,
		RateLimitEnabled:        true,
		Metadata:                make(map[string]interface{}),
		CreatedAt:               time.Now().UTC(),
		UpdatedAt:               time.Now().UTC(),
		IntegrationCount:        0,
	}

	// Create client in database
	if err := dcm.repository.CreateClient(dbClient); err != nil {
		return nil, err
	}

	// TODO: Create client directory structure if still needed
	// TODO: Store encryption key if still needed

	dcm.logger.Info("Client created", map[string]interface{}{
		"component":   "clients",
		"client_id":   clientID,
		"client_name": name,
	})

	return dcm.convertFromDB(dbClient), nil
}

// UpdateClient updates an existing client
func (dcm *DatabaseClientManager) UpdateClient(clientID string, updates map[string]interface{}) error {
	err := dcm.repository.UpdateClient(clientID, updates)
	if err != nil {
		return err
	}

	dcm.logger.Info("Client updated", map[string]interface{}{
		"component": "clients",
		"client_id": clientID,
		"updates":   updates,
	})

	return nil
}

// DeleteClient deletes a client and all associated data
func (dcm *DatabaseClientManager) DeleteClient(clientID string) error {
	// Delete from database (this will cascade to related tables due to foreign key)
	err := dcm.repository.DeleteClient(clientID)
	if err != nil {
		return err
	}

	// TODO: Delete client directories if still needed

	dcm.logger.Info("Client deleted", map[string]interface{}{
		"component": "clients",
		"client_id": clientID,
	})

	return nil
}

// ValidateClientAPIKey validates if an API key belongs to a specific client
func (dcm *DatabaseClientManager) ValidateClientAPIKey(clientID, apiKey string) bool {
	client, err := dcm.repository.GetClient(clientID)
	if err != nil || !client.Active {
		return false
	}

	for _, key := range client.APIKeys {
		if key == apiKey {
			return true
		}
	}

	return false
}

// GetClientByAPIKey finds a client by their API key
func (dcm *DatabaseClientManager) GetClientByAPIKey(apiKey string) (*Client, error) {
	dbClient, err := dcm.repository.GetClientByAPIKey(apiKey)
	if err != nil {
		return nil, err
	}

	return dcm.convertFromDB(dbClient), nil
}

// SearchClients searches clients by name or metadata
func (dcm *DatabaseClientManager) SearchClients(searchTerm string, limit int) ([]*Client, error) {
	dbClients, err := dcm.repository.SearchClients(searchTerm, limit)
	if err != nil {
		return nil, err
	}

	clients := make([]*Client, 0, len(dbClients))
	for _, dbClient := range dbClients {
		client := dcm.convertFromDB(dbClient)
		clients = append(clients, client)
	}

	return clients, nil
}

// UpdateIntegrationCount updates the integration count for a client
func (dcm *DatabaseClientManager) UpdateIntegrationCount(clientID string, count int) error {
	return dcm.repository.UpdateIntegrationCount(clientID, count)
}

// AuditLog logs client-related actions for audit purposes
func (dcm *DatabaseClientManager) AuditLog(clientID, action string, details map[string]interface{}) {
	dcm.logger.Info("Client action", map[string]interface{}{
		"component": "audit",
		"client_id": clientID,
		"action":    action,
		"details":   details,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Helper methods

func (dcm *DatabaseClientManager) convertFromDB(dbClient *database.Client) *Client {
	var lastAccessedAt string
	if dbClient.LastAccessedAt != nil {
		lastAccessedAt = dbClient.LastAccessedAt.Format(time.RFC3339)
	}

	return &Client{
		ID:              dbClient.ID,
		Name:            dbClient.Name,
		Description:     dbClient.Description,
		Active:          dbClient.Active,
		APIKeys:         dbClient.APIKeys,
		EncryptionKeyID: dbClient.EncryptionKeyID,
		RateLimits: &RateLimitConfig{
			RequestsPerMinute: dbClient.RateLimitRequestsPerMin,
			BurstLimit:        dbClient.RateLimitBurstLimit,
			Enabled:           dbClient.RateLimitEnabled,
		},
		Metadata:         dbClient.Metadata,
		CreatedAt:        dbClient.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        dbClient.UpdatedAt.Format(time.RFC3339),
		LastAccessedAt:   lastAccessedAt,
		IntegrationCount: dbClient.IntegrationCount,
	}
}

func (dcm *DatabaseClientManager) generateClientID() string {
	// Generate a random 20-character hex string
	bytes := make([]byte, 10)
	rand.Read(bytes)
	return "client_" + hex.EncodeToString(bytes)
}

func (dcm *DatabaseClientManager) generateAPIKey() string {
	// Generate a random 32-character hex string for API key
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "secauto_client_" + hex.EncodeToString(bytes)
}

func (dcm *DatabaseClientManager) generateEncryptionKeyID() string {
	// Generate a random 16-character hex string for encryption key ID
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

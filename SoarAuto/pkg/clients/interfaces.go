package clients

// ClientManagerInterface defines the interface for client management operations
// This allows for different implementations (file-based, database-based, etc.)
type ClientManagerInterface interface {
	// ListClients returns all clients
	ListClients() []*Client

	// GetClient retrieves a specific client by ID
	GetClient(clientID string) (*Client, error)

	// CreateClient creates a new client
	CreateClient(name, description string) (*Client, error)

	// UpdateClient updates an existing client
	UpdateClient(clientID string, updates map[string]interface{}) error

	// DeleteClient deletes a client and all associated data
	DeleteClient(clientID string) error

	// ValidateClientAPIKey validates if an API key belongs to a specific client
	ValidateClientAPIKey(clientID, apiKey string) bool

	// GetClientByAPIKey finds a client by their API key
	GetClientByAPIKey(apiKey string) (*Client, error)

	// AuditLog logs client-related actions for audit purposes
	AuditLog(clientID, action string, details map[string]interface{})
}

// Ensure both implementations satisfy the interface
var _ ClientManagerInterface = (*ClientManager)(nil)
var _ ClientManagerInterface = (*DatabaseClientManager)(nil)

# Client-Specific Integration Configurations

This document describes the client-specific integration configuration system that allows each client to have their own integration settings while maintaining global configurations.

## Overview

The integration configuration system now supports client-specific configurations, allowing each client to have their own API keys, URLs, and settings for integrations like VirusTotal, Slack, Email, etc. This ensures data isolation and allows for client-specific customization.

## Key Features

- **Client Isolation**: Each client can have their own integration configurations
- **Global Fallback**: Global configurations are available when client-specific ones don't exist
- **Encrypted Storage**: All configurations are encrypted at rest
- **RESTful API**: Full CRUD operations for client-specific integrations
- **Backward Compatibility**: Existing global integrations continue to work

## Storage Structure

Configurations are stored using a key format that includes the client name:

- **Global Configurations**: `integration_name` (e.g., `virustotal`)
- **Client-Specific Configurations**: `client:integration_name` (e.g., `acme-corp:virustotal`)

## API Endpoints

### Client-Specific Integration Management

#### List All Integrations for a Client
```
GET /clients/{client}/integrations
```

**Response:**
```json
{
  "success": true,
  "message": "Client-specific integrations retrieved successfully for client: acme-corp",
  "integrations": [
    {
      "name": "virustotal",
      "type": "virustotal",
      "client": "acme-corp",
      "url": "https://www.virustotal.com/vtapi/v2",
      "apikey": "client-specific-api-key",
      "enabled": true,
      "description": "Client-specific VirusTotal integration",
      "version": "1.0.0",
      "settings": {
        "timeout": 60,
        "retries": 3
      },
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### Get Client-Specific Integration
```
GET /clients/{client}/integrations/{integration}
```

#### Create Client-Specific Integration
```
POST /clients/{client}/integrations/{integration}
```

**Request Body:**
```json
{
  "name": "virustotal",
  "type": "virustotal",
  "url": "https://www.virustotal.com/vtapi/v2",
  "apikey": "client-specific-api-key",
  "enabled": true,
  "description": "Client-specific VirusTotal integration",
  "version": "1.0.0",
  "settings": {
    "timeout": 60,
    "retries": 3
  }
}
```

#### Update Client-Specific Integration
```
PUT /clients/{client}/integrations/{integration}
```

#### Delete Client-Specific Integration
```
DELETE /clients/{client}/integrations/{integration}
```

## Code Examples

### Creating Client-Specific Configurations

```go
// Initialize the integration config manager
manager, err := NewIntegrationConfigManager("data/integration_configs.enc", "your-encryption-key")
if err != nil {
    log.Fatalf("Failed to initialize integration config manager: %v", err)
}

// Create a VirusTotal integration for a specific client
clientVTConfig := &IntegrationConfig{
    Name:        "virustotal",
    Type:        "virustotal",
    URL:         "https://www.virustotal.com/vtapi/v2",
    APIKey:      "client-specific-api-key",
    Enabled:     true,
    Description: "Client-specific VirusTotal integration",
    Version:     "1.0.0",
    Settings: map[string]interface{}{
        "timeout": 60,
        "retries": 3,
    },
}

// Set the client-specific configuration
if err := manager.SetConfigByClient("acme-corp", "virustotal", clientVTConfig); err != nil {
    log.Fatalf("Failed to set client-specific config: %v", err)
}
```

### Retrieving Client-Specific Configurations

```go
// Get client-specific configuration
config, exists := manager.GetConfigByClient("acme-corp", "virustotal")
if exists {
    fmt.Printf("Client-specific VirusTotal config found for acme-corp\n")
    fmt.Printf("API Key: %s\n", config.APIKey)
    fmt.Printf("Enabled: %t\n", config.Enabled)
}

// Get a specific value from the client configuration
if apiKey, exists := manager.GetConfigValueByClient("acme-corp", "virustotal", "apikey"); exists {
    fmt.Printf("API Key for acme-corp: %s\n", apiKey)
}
```

### Listing Configurations

```go
// List all configurations for a specific client
clientConfigs := manager.ListConfigsByClient("acme-corp")
for name, config := range clientConfigs {
    fmt.Printf("  - %s: %s (Enabled: %t)\n", name, config.Name, config.Enabled)
}

// List global configurations
globalConfigs := manager.ListGlobalConfigs()
for key, config := range globalConfigs {
    fmt.Printf("  - %s: %s (Enabled: %t)\n", key, config.Name, config.Enabled)
}
```

## Use Cases

### 1. Multi-Tenant Environment
Each client can have their own API keys and settings for integrations:

```go
// Client A has their own VirusTotal API key
manager.SetConfigByClient("client-a", "virustotal", &IntegrationConfig{
    APIKey: "client-a-api-key",
    // ... other settings
})

// Client B has a different VirusTotal API key
manager.SetConfigByClient("client-b", "virustotal", &IntegrationConfig{
    APIKey: "client-b-api-key",
    // ... other settings
})
```

### 2. Different Settings per Client
Clients can have different timeout settings, retry configurations, etc.:

```go
// Client A prefers faster timeouts
manager.SetConfigByClient("client-a", "virustotal", &IntegrationConfig{
    Settings: map[string]interface{}{
        "timeout": 30,
        "retries": 2,
    },
})

// Client B prefers more reliable but slower settings
manager.SetConfigByClient("client-b", "virustotal", &IntegrationConfig{
    Settings: map[string]interface{}{
        "timeout": 120,
        "retries": 5,
    },
})
```

### 3. Integration Availability
Some clients might have access to certain integrations while others don't:

```go
// Client A has access to premium integrations
manager.SetConfigByClient("client-a", "premium-scanner", &IntegrationConfig{
    Enabled: true,
    // ... premium settings
})

// Client B doesn't have access to premium integrations
// (no configuration set, so it won't be available)
```

## Migration from Global Configurations

Existing global configurations continue to work without any changes. To migrate to client-specific configurations:

1. **Identify clients** that need their own configurations
2. **Create client-specific configurations** using the new API endpoints
3. **Update automation scripts** to use client-specific configurations
4. **Test thoroughly** to ensure proper isolation

## Security Considerations

- All configurations are encrypted at rest using AES-256-GCM
- Client-specific configurations are isolated from each other
- API keys and sensitive data are encrypted
- Access control should be implemented at the application level

## Testing

Run the test file to verify functionality:

```bash
cd SoarAuto
go run test_client_integrations.go
```

This will demonstrate:
- Creating global and client-specific configurations
- Listing configurations by client
- Retrieving specific configuration values
- Testing configuration isolation

## Best Practices

1. **Use descriptive client names** that are easy to identify
2. **Document client-specific settings** for each integration
3. **Implement proper access control** to ensure clients can only access their own configurations
4. **Regularly audit configurations** to ensure they're up to date
5. **Test client isolation** to prevent data leakage between clients

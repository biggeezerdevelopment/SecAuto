# SecAuto Cache API

The SecAuto Cache API provides Redis-based caching functionality for storing and retrieving key-value pairs. This is useful for caching automation results, sharing data between playbooks, and storing temporary values.

## 🚀 Getting Started

The cache API is automatically available when SecAuto starts, using the Redis configuration from `config.yaml`.

### Prerequisites

- Redis server running (configured in `config.yaml`)
- Valid API key for authentication

## 📚 API Endpoints

### 1. Get Cache Information

**GET** `/cache`

Returns general information about the cache system.

```bash
curl -X GET http://localhost:8000/cache \
  -H "X-API-Key: your-api-key"
```

**Response:**
```json
{
  "success": true,
  "message": "Redis cache is available",
  "operations": [
    {"method": "GET", "path": "/cache/{key}", "description": "Get value from cache"},
    {"method": "POST", "path": "/cache/{key}", "description": "Set value in cache"},
    {"method": "DELETE", "path": "/cache/{key}", "description": "Delete value from cache"}
  ],
  "timestamp": "2025-01-24T12:00:00Z"
}
```

### 2. Get Cache Value

**GET** `/cache/{key}`

Retrieves a value from the cache by key.

```bash
curl -X GET http://localhost:8000/cache/my-key \
  -H "X-API-Key: your-api-key"
```

**Success Response:**
```json
{
  "success": true,
  "key": "my-key",
  "value": {"data": "cached value"},
  "message": "Value retrieved successfully",
  "timestamp": "2025-01-24T12:00:00Z"
}
```

**Not Found Response:**
```json
{
  "success": false,
  "key": "my-key",
  "error_message": "Key not found in cache",
  "timestamp": "2025-01-24T12:00:00Z"
}
```

### 3. Set Cache Value

**POST** `/cache/{key}`

Stores a value in the cache with the specified key.

```bash
curl -X POST http://localhost:8000/cache/my-key \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "value": {
      "data": "some value",
      "timestamp": "2025-01-24T12:00:00Z"
    }
  }'
```

**Request Body:**
```json
{
  "value": "any JSON value - string, number, object, array"
}
```

**Response:**
```json
{
  "success": true,
  "key": "my-key",
  "value": {"data": "some value", "timestamp": "2025-01-24T12:00:00Z"},
  "message": "Value set successfully",
  "timestamp": "2025-01-24T12:00:00Z"
}
```

### 4. Delete Cache Value

**DELETE** `/cache/{key}`

Removes a value from the cache.

```bash
curl -X DELETE http://localhost:8000/cache/my-key \
  -H "X-API-Key: your-api-key"
```

**Success Response:**
```json
{
  "success": true,
  "key": "my-key",
  "message": "Value deleted successfully",
  "timestamp": "2025-01-24T12:00:00Z"
}
```

**Not Found Response:**
```json
{
  "success": false,
  "key": "my-key",
  "error_message": "Key not found in cache",
  "timestamp": "2025-01-24T12:00:00Z"
}
```

## 🔧 Data Types

The cache API supports all JSON data types:

### String Values
```json
{"value": "Hello World"}
```

### Numeric Values
```json
{"value": 42}
{"value": 3.14}
```

### Boolean Values
```json
{"value": true}
```

### Object Values
```json
{
  "value": {
    "user_id": 123,
    "username": "john_doe",
    "preferences": {
      "theme": "dark",
      "notifications": true
    }
  }
}
```

### Array Values
```json
{
  "value": [
    {"id": 1, "name": "Item 1"},
    {"id": 2, "name": "Item 2"}
  ]
}
```

## 🔄 Use Cases

### 1. Caching Automation Results
Store the results of expensive operations for reuse:

```bash
# Store VirusTotal scan results
curl -X POST http://localhost:8000/cache/virustotal-scan-123 \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "value": {
      "scan_id": "123",
      "results": {"malicious": 0, "clean": 67},
      "cached_at": "2025-01-24T12:00:00Z"
    }
  }'
```

### 2. Sharing Data Between Playbooks
Pass data from one playbook execution to another:

```bash
# Store incident context
curl -X POST http://localhost:8000/cache/incident-456-context \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "value": {
      "incident_id": "456",
      "severity": "high",
      "affected_systems": ["server1", "server2"],
      "timeline": []
    }
  }'
```

### 3. Rate Limiting and Deduplication
Store request timestamps to implement rate limiting:

```bash
# Store last API call timestamp
curl -X POST http://localhost:8000/cache/api-limit-virustotal \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "value": {
      "last_call": "2025-01-24T12:00:00Z",
      "call_count": 5
    }
  }'
```

## 🛡️ Security

- **Authentication Required**: All cache endpoints require a valid API key
- **Rate Limiting**: Cache operations are rate-limited (200 requests per minute by default)
- **Input Validation**: All input is validated before storage
- **No Expiration**: Values persist until explicitly deleted (consider implementing TTL if needed)

## ⚡ Performance

- **Redis Backend**: Uses Redis for high-performance storage
- **JSON Serialization**: Automatic JSON encoding/decoding
- **Connection Pooling**: Efficient Redis connection management
- **Error Handling**: Comprehensive error handling and logging

## 🔧 Configuration

Cache settings are configured in `config.yaml`:

```yaml
database:
  redis_url: "redis://localhost:6379/0"

security:
  rate_limiting:
    endpoints:
      cache: 200  # requests per minute
```

## 📝 Examples

### Python Integration
```python
import requests

# Set cache value
response = requests.post(
    "http://localhost:8000/cache/my-data",
    headers={"X-API-Key": "your-api-key"},
    json={"value": {"processed": True, "count": 42}}
)

# Get cache value
response = requests.get(
    "http://localhost:8000/cache/my-data",
    headers={"X-API-Key": "your-api-key"}
)
data = response.json()
if data["success"]:
    cached_value = data["value"]
```

### Automation Script Usage
Use the cache in your Python automation scripts:

```python
import requests
import json

def cache_set(key, value):
    response = requests.post(
        f"http://localhost:8000/cache/{key}",
        headers={"X-API-Key": "your-api-key"},
        json={"value": value}
    )
    return response.json()

def cache_get(key):
    response = requests.get(
        f"http://localhost:8000/cache/{key}",
        headers={"X-API-Key": "your-api-key"}
    )
    data = response.json()
    return data["value"] if data["success"] else None

# Example usage in automation
scan_results = {"clean": 10, "malicious": 0}
cache_set("last-scan-results", scan_results)

# Later in another automation
previous_results = cache_get("last-scan-results")
if previous_results:
    print(f"Previous scan: {previous_results}")
```

## 🐛 Troubleshooting

### Common Issues

1. **Connection Failed**: Ensure Redis is running and accessible
2. **Authentication Failed**: Verify your API key is correct
3. **Key Not Found**: The key doesn't exist in cache
4. **Rate Limited**: Too many requests, wait before retrying
5. **Invalid JSON**: Request body must be valid JSON

### Error Responses

All error responses follow this format:
```json
{
  "success": false,
  "key": "optional-key",
  "error_message": "Description of the error",
  "timestamp": "2025-01-24T12:00:00Z"
}
```

## 🔗 Related Features

- **Redis Job Store**: Uses the same Redis instance for job persistence
- **Distributed System**: Cache can be shared across cluster nodes
- **Webhook Integration**: Cache webhook delivery status
- **Plugin System**: Plugins can use cache for state management
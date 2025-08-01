# Redis Job Store Implementation

This document describes the Redis job store implementation for SecAuto SOAR, which provides an alternative to the SQLite job store with better performance and scalability.

## Overview

The Redis job store eliminates the deadlock issues present in the SQLite implementation and provides:
- **No deadlocks**: Redis operations are atomic and don't require complex locking
- **Better performance**: In-memory operations with optional persistence
- **Scalability**: Can be shared across multiple SecAuto instances
- **TTL support**: Automatic cleanup of old jobs
- **Backup capabilities**: Built-in backup and recovery features

## Configuration

### Basic Configuration

Update your `config.yaml` to use Redis:

```yaml
database:
  type: "redis"  # Use Redis instead of SQLite
  redis_url: "redis://localhost:6379/0"  # Redis connection URL
```

### Redis URL Format

The Redis URL supports the following formats:
- `redis://localhost:6379/0` - Basic connection
- `redis://user:password@localhost:6379/0` - With authentication
- `redis://localhost:6379/0?pool_size=10&pool_timeout=30s` - With connection pool options

### Advanced Configuration

```yaml
database:
  type: "redis"
  redis_url: "redis://localhost:6379/0"
  # Other database settings remain the same
  connection_pool:
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: "1h"
    conn_max_idle_time: "30m"
```

## Installation

### 1. Install Redis

**Windows:**
```powershell
# Using Chocolatey
choco install redis-64

# Or download from https://github.com/microsoftarchive/redis/releases
```

**Linux:**
```bash
# Ubuntu/Debian
sudo apt-get install redis-server

# CentOS/RHEL
sudo yum install redis
```

**macOS:**
```bash
# Using Homebrew
brew install redis
```

### 2. Start Redis Server

**Windows:**
```powershell
redis-server
```

**Linux/macOS:**
```bash
sudo systemctl start redis
# or
redis-server
```

### 3. Test Redis Connection

```bash
redis-cli ping
# Should return: PONG
```

## Features

### Job Storage

Jobs are stored in Redis with the following structure:
- **Job data**: `job:{job_id}` - Contains serialized job JSON
- **Job list**: `jobs:list` - Sorted set with job IDs and creation timestamps
- **TTL**: Jobs automatically expire after 24 hours

### Performance Benefits

1. **No Deadlocks**: Redis operations are atomic
2. **Faster Queries**: In-memory operations
3. **Automatic Cleanup**: TTL-based expiration
4. **Scalable**: Can handle thousands of concurrent operations

### Backup and Recovery

The Redis job store includes:
- **Automatic backups**: Jobs are backed up to Redis with 7-day TTL
- **Crash recovery**: Running jobs are marked as failed on restart
- **Manual backup**: API endpoint for manual backup creation

## API Compatibility

The Redis job store implements the same interface as the SQLite store, so no API changes are required:

```go
// Same interface for both implementations
type JobStoreInterface interface {
    SaveJob(job *Job) error
    LoadJob(jobID string) (*Job, bool)
    ListJobs(status string, limit int) []*Job
    UpdateJobStatus(jobID, status string) error
    UpdateJobResults(jobID string, results []interface{}, errorMsg string) error
    DeleteJob(jobID string) error
    // ... other methods
}
```

## Migration from SQLite

### Automatic Migration

The system will automatically use the configured job store type. To migrate:

1. **Backup existing data**:
   ```bash
   # Backup SQLite database
   cp data/jobs.db data/jobs.db.backup
   ```

2. **Update configuration**:
   ```yaml
   database:
     type: "redis"
     redis_url: "redis://localhost:6379/0"
   ```

3. **Restart SecAuto**:
   ```bash
   ./secauto
   ```

### Manual Migration

For large datasets, you can implement a custom migration script:

```go
// Example migration function
func migrateFromSQLiteToRedis(sqlitePath, redisURL string) error {
    // Load SQLite store
    sqliteStore, err := NewSQLiteJobStore("data")
    if err != nil {
        return err
    }
    
    // Create Redis store
    redisStore, err := NewRedisJobStore(redisURL)
    if err != nil {
        return err
    }
    
    // Migrate jobs
    jobs := sqliteStore.ListJobs("", 10000)
    for _, job := range jobs {
        if err := redisStore.SaveJob(job); err != nil {
            return err
        }
    }
    
    return nil
}
```

## Monitoring

### Redis Metrics

The Redis job store provides metrics through the `/api/jobs/metrics` endpoint:

```json
{
  "success": true,
  "metrics": {
    "type": "redis",
    "info": "Redis server information...",
    "job_count": 150,
    "memory_usage": "2.5MB"
  }
}
```

### Health Checks

Redis connectivity is checked during:
- Server startup
- Job operations
- Health check endpoint (`/api/health`)

## Troubleshooting

### Common Issues

1. **Connection Refused**:
   ```
   Error: failed to connect to Redis: dial tcp localhost:6379: connect: connection refused
   ```
   **Solution**: Start Redis server

2. **Authentication Failed**:
   ```
   Error: failed to connect to Redis: NOAUTH Authentication required
   ```
   **Solution**: Update Redis URL with password or disable authentication

3. **Memory Issues**:
   ```
   Error: OOM command not allowed when used memory > 'maxmemory'
   ```
   **Solution**: Increase Redis memory limit or enable persistence

### Redis Configuration

Recommended Redis configuration (`redis.conf`):

```conf
# Memory management
maxmemory 256mb
maxmemory-policy allkeys-lru

# Persistence (optional)
save 900 1
save 300 10
save 60 10000

# Network
bind 127.0.0.1
port 6379

# Security
requirepass your_password_here
```

## Performance Tuning

### Redis Optimization

1. **Memory Usage**: Monitor with `redis-cli info memory`
2. **Connection Pool**: Adjust pool size based on load
3. **TTL Settings**: Modify job expiration time as needed
4. **Persistence**: Configure RDB/AOF for data durability

### Monitoring Commands

```bash
# Check Redis info
redis-cli info

# Monitor Redis operations
redis-cli monitor

# Check memory usage
redis-cli info memory

# List all keys
redis-cli keys "*"
```

## Security Considerations

1. **Network Security**: Bind Redis to localhost only
2. **Authentication**: Use Redis password for production
3. **Firewall**: Block Redis port from external access
4. **Encryption**: Use SSL/TLS for remote Redis connections

## Production Deployment

### Docker Compose Example

```yaml
version: '3.8'
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes
    restart: unless-stopped

  secauto:
    build: .
    ports:
      - "8080:8080"
    environment:
      - REDIS_URL=redis://redis:6379/0
    depends_on:
      - redis
    restart: unless-stopped

volumes:
  redis_data:
```

### Kubernetes Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: secauto-config
data:
  config.yaml: |
    database:
      type: "redis"
      redis_url: "redis://redis-service:6379/0"

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: secauto
spec:
  replicas: 3
  selector:
    matchLabels:
      app: secauto
  template:
    metadata:
      labels:
        app: secauto
    spec:
      containers:
      - name: secauto
        image: secauto:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
      volumes:
      - name: config
        configMap:
          name: secauto-config
```

## Redis Cache API

SecAuto now includes a comprehensive Redis Cache API that allows automations and external applications to store and retrieve data efficiently.

### Cache API Features

- **RESTful Interface**: Simple HTTP endpoints for cache operations
- **JSON Support**: Automatic serialization/deserialization of complex data
- **High Performance**: Redis-backed for optimal speed
- **Authentication**: API key authentication required
- **Rate Limiting**: Configurable rate limits for cache operations

### Cache API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/cache` | GET | Get cache system information |
| `/cache/{key}` | GET | Retrieve value from cache |
| `/cache/{key}` | POST | Store value in cache |
| `/cache/{key}` | DELETE | Remove value from cache |

### Usage Examples

#### Store Data in Cache
```bash
curl -X POST http://localhost:8000/cache/incident-123 \
  -H "X-API-Key: secauto-api-key-2024-07-14" \
  -H "Content-Type: application/json" \
  -d '{
    "value": {
      "incident_id": "123",
      "severity": "high",
      "status": "investigating",
      "timestamp": "2025-01-24T12:00:00Z"
    }
  }'
```

#### Retrieve Data from Cache
```bash
curl -X GET http://localhost:8000/cache/incident-123 \
  -H "X-API-Key: secauto-api-key-2024-07-14"
```

#### Delete Data from Cache
```bash
curl -X DELETE http://localhost:8000/cache/incident-123 \
  -H "X-API-Key: secauto-api-key-2024-07-14"
```

### Integration with Automations

Automations can use the cache API to:
- **Share Data**: Pass data between different automation executions
- **Cache Results**: Store expensive computation results for reuse
- **State Management**: Maintain state across playbook executions
- **Rate Limiting**: Track API call timestamps to implement rate limiting

#### Python Example
```python
import requests
import json

# Cache automation results
def cache_scan_results(scan_id, results):
    response = requests.post(
        f"http://localhost:8000/cache/scan-{scan_id}",
        headers={"X-API-Key": "secauto-api-key-2024-07-14"},
        json={"value": results}
    )
    return response.json()

# Retrieve cached results
def get_cached_scan(scan_id):
    response = requests.get(
        f"http://localhost:8000/cache/scan-{scan_id}",
        headers={"X-API-Key": "secauto-api-key-2024-07-14"}
    )
    data = response.json()
    return data["value"] if data["success"] else None
```

### Configuration

Cache API settings are configured in `config.yaml`:

```yaml
security:
  rate_limiting:
    endpoints:
      cache: 200  # requests per minute for cache operations

database:
  redis_url: "redis://localhost:6379/0"  # Same Redis instance used for cache
```

### Documentation

For complete Cache API documentation, see: [CACHE_API_README.md](CACHE_API_README.md)

## Conclusion

The Redis infrastructure in SecAuto provides both robust job storage and high-performance caching capabilities. The Redis job store eliminates SQLite deadlock issues while the Cache API enables efficient data sharing and state management across automation workflows. This dual-purpose Redis integration makes SecAuto well-suited for production environments with high throughput and complex automation requirements. 
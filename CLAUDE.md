# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SecAuto is a Security Orchestration, Automation & Response (SOAR) platform written in Go with Python automation scripts. It provides playbook-based security automation, integration capabilities, and a Redis-based distributed architecture for cybersecurity operations.

## Build and Development Commands

### Go Server (SoarAuto)
```bash
# Build the main server
cd SoarAuto
go mod tidy
go build -o soarauto.exe .  # Windows
go build -o secauto .        # macOS/Linux

# Run the server
./soarauto.exe  # Windows
./secauto       # macOS/Linux

# Build with specific target
GOOS=linux GOARCH=amd64 go build -o secauto
GOOS=windows GOARCH=amd64 go build -o soarauto.exe

# List all Go packages
go list ./...

# Format Go code
go fmt ./...

# Run Go vet for static analysis
go vet ./...
```

### Python Environment
```bash
# Set up Python virtual environment (required for automations)
python3 -m venv Venv
source Venv/bin/activate  # On Windows: Venv\Scripts\activate

# Note: No requirements.txt exists - dependencies are handled per automation script
```

### Redis Setup (Required)
```bash
# Start Redis server
redis-server

# Test Redis connection
redis-cli ping

# Monitor Redis operations (debugging)
redis-cli monitor
```

## Architecture

### Package Structure
The codebase follows a modular architecture with packages in `SoarAuto/pkg/`:

- **auth**: API key authentication system (`pkg/auth/auth.go`)
- **automations**: Python script automation management (`pkg/automations/automations.go`)
- **cache**: Context caching with TTL and lazy evaluation (`pkg/cache/cache.go`)
- **clients**: Client management and tracking (`pkg/clients/clients.go`)
- **cluster**: Distributed node coordination (`pkg/cluster/cluster.go`)
- **config**: YAML configuration management (`pkg/config/config.go`)
- **errors**: Custom error handling (`pkg/errors/errors.go`)
- **integrations**: External service integrations (`pkg/integrations/integrations.go`)
- **jobs**: Asynchronous job execution and persistence (`pkg/jobs/jobs.go`)
- **logger**: Structured JSON logging with rotation (`pkg/logger/logger.go`)
- **middleware**: HTTP middleware components
- **performance**: Performance optimization utilities (`pkg/performance/`)
  - Async operations, caching, pooling, and profiling capabilities
- **playbooks**: JSON playbook management (`pkg/playbooks/playbooks.go`)
- **plugins**: Plugin system management
- **recovery**: Panic recovery and error handling (`pkg/recovery/recovery.go`)
- **redis**: Redis client and connection pooling (`pkg/redis/redis.go`)
- **rules**: Core playbook execution engine (`pkg/rules/engine.go`)
- **schedules**: Cron-based job scheduling (`pkg/schedules/schedules.go`)
- **security**: Security components (`pkg/security/`)
  - Audit logging (`audit.go`)
  - Security middleware (`middleware.go`)
  - Rate limiting (`ratelimit.go`)
  - Input validation (`validation.go`)
- **swagger**: API documentation handler (`pkg/swagger/swagger.go`)
- **testutil**: Testing utilities (`pkg/testutil/testutil.go`)
- **tls**: TLS/HTTPS configuration (`pkg/tls/tls.go`)
- **types**: Shared type definitions (`pkg/types/types.go`)
- **validator**: Input validation system (`pkg/validator/validator.go`)
- **webhooks**: Webhook management system

### Key Components

1. **Main Entry Point**: `SoarAuto/main.go` - HTTP server with middleware chain
2. **Configuration**: `SoarAuto/config.yaml` - All settings including ports, Redis, logging
3. **Python Automations**: `automations/` - Scripts using `SoarBaseAPI.py` helper
4. **Playbooks**: `playbooks/` - JSON workflow definitions
5. **Integrations**: `integrations/` - External service connectors
6. **Plugins**: Multi-platform plugin system (`plugins/` with go/python/windows/linux subdirs)

### API Endpoints

All endpoints require `X-API-Key` header except `/health` and `/docs`:

| Endpoint | Method | Handler Function | Description |
|----------|--------|-----------------|-------------|
| `/health` | GET | `healthHandler` | Health check (no auth) |
| `/playbook` | POST | `playbookHandler` | Execute playbook synchronously |
| `/playbook/async` | POST | `playbookAsyncHandler` | Execute playbook asynchronously |
| `/playbook/upload` | POST | `playbookUploadHandler` | Upload new playbook |
| `/playbook/{name}` | DELETE | `playbookDeleteHandler` | Delete playbook |
| `/playbooks` | GET | `playbooksHandler` | List all playbooks |
| `/cache` | GET | `cacheHandler` | Get cache info |
| `/cache/stats` | GET | `cacheStatsHandler` | Cache statistics |
| `/cache/clear` | POST | `cacheClearHandler` | Clear cache |
| `/cache/{key}` | GET/POST/DELETE | `cacheKeyHandler` | Cache operations |
| `/lists/{key}` | GET/POST | `listHandler` | Redis list operations |
| `/integrations` | GET | `integrationsHandler` | List integrations |
| `/integrations/{name}` | POST | `integrationHandler` | Execute integration |
| `/integrations/upload` | POST | `integrationUploadHandler` | Upload integration |
| `/cluster` | GET | `clusterHandler` | Cluster status |
| `/cluster/jobs` | GET | `clusterJobsHandler` | List cluster jobs |
| `/cluster/jobs/{id}` | GET | `clusterJobHandler` | Get cluster job |
| `/automation` | POST | `automationUploadHandler` | Upload automation |
| `/automations` | GET | `automationsListHandler` | List automations |
| `/automation/{name}` | DELETE | `automationDeleteHandler` | Delete automation |
| `/automation/metadata` | GET/POST | `automationMetadataHandler` | Manage metadata |
| `/automation/metadata/{key}` | GET/PUT/DELETE | `automationMetadataItemHandler` | Metadata operations |
| `/jobs` | GET | `jobsHandler` | List all jobs |
| `/jobs/stats` | GET | `jobsStatsHandler` | Job statistics |
| `/job/{id}` | GET/DELETE | `jobHandler` | Get/delete job |
| `/schedules` | GET/POST | `schedulesHandler` | Manage schedules |
| `/schedules/stats` | GET | `scheduleStatsHandler` | Schedule statistics |
| `/schedule/{id}` | GET/PUT/DELETE | `scheduleHandler` | Manage schedule |
| `/schedule/execute/{id}` | POST | `scheduleExecuteHandler` | Execute schedule |
| `/api-keys` | GET/POST/DELETE | `apiKeysHandler` | Manage API keys |
| `/api-keys/stats` | GET | `apiKeyStatsHandler` | API key statistics |
| `/clients` | GET/POST | `clientsHandler` | Manage clients |
| `/clients/{id}` | GET/PUT/DELETE | `clientHandler` | Manage client |
| `/docs` | GET | Swagger UI | API documentation (no auth) |
| `/docs/` | GET | Swagger UI | API documentation (no auth) |
| `/api-docs` | GET | Swagger UI | API documentation (no auth) |

## Configuration

Primary configuration is in `SoarAuto/config.yaml`:

```yaml
server:
  port: 9090  # API port
  host: "localhost"
  workers: 5  # Concurrent workers
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "60s"
  max_header_bytes: 1048576
  # TLS/HTTPS Configuration
  tls:
    enabled: false
    port: 9443
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"
    auto_redirect: true
    min_version: "1.2"
    max_version: "1.3"
    
database:
  redis_url: "redis://localhost:6379/0"
  cache_ttl: 3600  # 1 hour
  job_ttl: 86400   # 24 hours
  temp_data_ttl: 300  # 5 minutes
  
logging:
  level: "DEBUG"  # DEBUG, INFO, WARNING, ERROR
  destination: "both"  # console, file, or both
  file: "logs/secauto.log"
  format: "json"
  rotation:
    max_size_mb: 10
    max_backups: 5
    max_age_days: 30
    compress: true
  component_levels:  # Per-component log levels
    rules_engine: "WARNING"
    redis_integration: "ERROR"
    job_manager: "INFO"
    webhook_system: "INFO"
    plugin_system: "WARNING"
    cluster_manager: "INFO"
    default: "INFO"
    
cluster:
  enabled: false
  redis_url: "redis://localhost:6379/1"
  node_id: "node-1"
  cluster_name: "secauto-cluster"
  heartbeat_interval: 30
  job_timeout: 3600
  max_retries: 3
  
security:
  api_keys:
    - "your-api-key-here"
  rate_limiting:
    enabled: true
    
python:
  venv_path: "../Venv"  # Path to Python virtual environment
  
plugins:
  enabled: true
  directory: "../plugins"
  platforms:
    python:
      directory: "../plugins/python"
      interpreter: "../Venv/Scripts/python.exe"  # Windows path
```

## Development Patterns

### Adding New Features

1. **Create Package**: Add to `SoarAuto/pkg/{feature}/`
2. **Implement Manager**: Create struct with `New{Feature}Manager()` function
3. **Wire in main.go**: Initialize in `NewSecAutoServer()` and add handlers
4. **Update Config**: Add to `config.yaml` and `pkg/config/config.go` structs

### Python Automation Scripts

Standard pattern for automations:
```python
import json
import sys
from server.SoarBaseAPI import load_context, return_context

def main():
    # Load context from stdin or argv
    context = load_context()
    if not context and len(sys.argv) > 1:
        context = json.loads(sys.argv[1])
    
    # Process automation logic
    result = {"success": True, "data": process_data(context)}
    
    # Return JSON result
    return_context(result)

if __name__ == "__main__":
    main()
```

### Playbook Structure

JSON array format with actions:
```json
[
  {"run": "data_enrichment"},
  {
    "if": {
      "conditions": [["==", {"var": "severity"}, "high"]],
      "true": {"run": "escalate"},
      "false": {"run": "log_incident"}
    }
  },
  {"cache": {"key": "result", "value": {"var": "output"}}}
]
```

## Redis Usage

Redis serves as the central data store for:
- **Job Storage**: Persistent job queue with TTL management
- **Cache API**: Key-value storage for automation data sharing
- **Distributed Locks**: Coordination between cluster nodes
- **Schedule Storage**: Cron job definitions and state
- **Context Cache**: Rule engine expression caching

Default connection: `redis://localhost:6379/0` (DB 0 for main, DB 1 for cluster)

## Important Files and Directories

### Core Files
- `SoarAuto/main.go`: HTTP server and all endpoint handlers
- `SoarAuto/config.yaml`: Main configuration file
- `server/SoarBaseAPI.py`: Python helper functions for automations
- `SoarAuto/legacy-archived-20250815/`: Old implementation (reference only)

### Data Directories
- `automations/`: Python automation scripts
- `playbooks/`: JSON playbook definitions
- `integrations/`: Integration modules and configs
- `plugins/`: Multi-platform plugin system
- `logs/`: Application and standalone logs
- `SoarAuto/data/`: Runtime data (metadata, configs, security)

## Common Development Tasks

### Adding New API Endpoint
1. Add handler function in `main.go` (follow pattern: `func (s *SecAutoServer) myHandler`)
2. Register route in server initialization (around line 2865-2923 in `main.go`)
3. Add middleware chain: `http.HandleFunc("/myendpoint", s.middleware(s.myHandler))` for auth-required endpoints
4. Use `s.publicMiddleware()` for public endpoints (no auth required)
5. Update Swagger documentation if needed

### Creating Python Automation
1. Create script in `automations/my_script.py`
2. Use `SoarBaseAPI` helpers for context handling
3. Reference in playbook as `{"run": "my_script"}`
4. Test: `curl -X POST localhost:9090/playbook -H "X-API-Key: ..." -d '{"playbook": [{"run": "my_script"}]}'`

### Working with Plugins
- Go plugins: Build as `.so` files, place in `plugins/go/`
- Python plugins: Place `.py` files in `plugins/python/`
- Platform-specific: Use `plugins/windows/` or `plugins/linux/`
- Configuration in `config.yaml` under `plugins.platforms`

### Debugging Tips
- Enable DEBUG logging: Set `logging.level: "DEBUG"` in config
- Monitor Redis: `redis-cli monitor` to see all Redis operations
- Check component logs: Adjust `logging.component_levels` for specific packages
- Test endpoints: Use `/docs` for Swagger UI testing
- Job status: Check `/jobs` and `/job/{id}` for async execution status
- Performance profiling: Use the performance package utilities for optimization
- Check client status: Use `/clients` endpoint to monitor connected clients
- Schedule monitoring: Use `/schedules/stats` for scheduling system health

## Sample Automation Scripts

The `automations/` directory contains various security automation scripts:
- **client_virustotal_scanner.py**: VirusTotal integration for threat scanning
- **data_enrichment.py**: Enrich security events with additional context
- **threat_analyzer.py**: Analyze threat intelligence data
- **email_notification.py**: Send security alert notifications
- **qualysauto.py**: Qualys vulnerability scanner integration
- **tenableauto.py**: Tenable.io integration for vulnerability management

## Testing

Test files are located alongside their respective packages:
- Unit tests: `*_test.go` files in each package
- Test utilities: `pkg/testutil/testutil.go`
- Test playbooks: `playbooks/test_*.json`
- Test automations: `automations/test_*.py`
# SecAuto Modular SOAR Platform

A modular Security Orchestration, Automation & Response (SOAR) platform built with Go, featuring distributed processing, comprehensive API management, and enterprise-grade security features.

## 🚀 Overview

SecAuto has been completely rewritten with a modular architecture that provides:

- **Modular Design**: Clean separation of concerns with independent packages
- **API Key Authentication**: Enterprise-grade security with dynamic key management
- **CORS Support**: Cross-origin resource sharing for web applications
- **Distributed Processing**: Cluster-based job execution with Redis backend
- **Comprehensive APIs**: RESTful endpoints with OpenAPI 3.0 documentation
- **Real-time Scheduling**: Cron-based job scheduling with advanced management
- **Integration Management**: Dynamic integration with Python script uploads
- **Advanced Caching**: Redis-based caching with TTL management

## 📁 Project Structure

```
SoarAuto/
├── main.go                    # Main server entry point
├── config.yaml               # Main configuration file
├── go.mod                     # Go module dependencies
├── 
├── pkg/                       # Modular packages
│   ├── auth/                  # API key authentication system
│   │   └── auth.go           # JWT-like API key management
│   ├── automations/           # Automation script management
│   │   └── automations.go    # Python script upload & analysis
│   ├── cache/                 # Redis caching layer
│   │   └── cache.go          # TTL-based cache operations
│   ├── cluster/               # Distributed processing
│   │   └── cluster.go        # Worker pool & job distribution
│   ├── config/                # Configuration management
│   │   └── config.go         # YAML-based config with defaults
│   ├── integrations/          # Integration management
│   │   └── integrations.go   # Dynamic integration configs
│   ├── jobs/                  # Job management
│   │   └── jobs.go           # Persistent job storage
│   ├── logger/                # Structured logging
│   │   └── logger.go         # Component-based logging
│   ├── playbooks/             # Playbook management
│   │   └── playbooks.go      # JSON playbook execution
│   ├── redis/                 # Redis operations
│   │   └── redis.go          # Connection pool management
│   ├── rules/                 # Rules engine
│   │   └── engine.go         # Expression evaluation engine
│   ├── schedules/             # Job scheduling
│   │   └── schedules.go      # Cron-based job scheduling
│   ├── swagger/               # API documentation
│   │   └── swagger.go        # OpenAPI 3.0 spec generation
│   ├── types/                 # Shared types
│   │   └── types.go          # Common data structures
│   └── validator/             # Input validation
│       └── validator.go      # Request validation system
│
├── data/                      # Persistent data storage
│   ├── automations/           # Automation scripts & metadata
│   ├── integrations/          # Integration configs & scripts
│   ├── playbooks/             # Stored playbooks
│   ├── schedules/             # Schedule definitions
│   └── security/              # API keys & security data
│       └── api_keys.json     # Generated API keys
│
├── legacy-archived-20250815/  # Previous implementation (archived)
├── legacy-codebase-20250815.zip # Legacy code archive
├── LEGACY_ARCHIVE_INFO.md     # Archive documentation
├── READMES/                   # Detailed documentation
├── docker/                    # Containerization configs
├── logs/                      # Application logs
└── bin/                       # Compiled binaries
```

## 🔧 Installation & Setup

### Prerequisites
- Go 1.21 or higher
- Redis server
- Python 3.8+ (for automation scripts)

### Quick Start

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd SoarAuto
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Configure Redis**:
   ```bash
   # Start Redis server
   redis-server
   ```

4. **Start the server**:
   ```bash
   go run main.go
   ```

5. **Access the API**:
   - Server: http://localhost:9090
   - Health: http://localhost:9090/health
   - Documentation: http://localhost:9090/docs

## 🔐 Authentication

### API Key Management

SecAuto uses API key authentication for all endpoints (except `/health` and `/docs`).

**Default API Keys** (from config):
- `secauto-api-key-2024-07-14`
- `another-api-key-if-needed`

**Create New API Keys**:
```bash
curl -X POST "http://localhost:9090/api-keys" \
  -H "X-API-Key: secauto-api-key-2024-07-14" \
  -H "Content-Type: application/json" \
  -d '{"name": "my-key", "description": "My API key"}'
```

**Authentication Methods**:
- Header: `X-API-Key: your-api-key-here`
- Query param: `?api_key=your-api-key-here`

## 🌐 API Endpoints

### Core APIs

| Category | Endpoint | Method | Description |
|----------|----------|---------|-------------|
| **Health** | `/health` | GET | System health check |
| **Playbooks** | `/playbook` | POST | Execute playbook |
| | `/playbook/async` | POST | Async playbook execution |
| | `/playbook/upload` | POST | Upload playbook file |
| | `/playbooks` | GET | List all playbooks |
| **Jobs** | `/jobs` | GET | List jobs with filtering |
| | `/jobs/stats` | GET | Job statistics |
| | `/job/{id}` | GET/PUT/DELETE | Job management |
| **Schedules** | `/schedules` | GET/POST | Schedule management |
| | `/schedule/{id}` | GET/PUT/DELETE | Individual schedules |
| | `/schedule/execute/{id}` | POST | Manual execution |
| **Cache** | `/cache` | GET | List cache keys |
| | `/cache/{key}` | GET/POST/DELETE | Cache operations |
| | `/cache/stats` | GET | Cache statistics |
| **API Keys** | `/api-keys` | GET/POST | Key management |
| | `/api-keys/stats` | GET | Key statistics |

### Integration Management

| Endpoint | Method | Description |
|----------|---------|-------------|
| `/integrations` | GET/POST | List/create integrations |
| `/integrations/{id}` | GET/PUT/DELETE | Manage integrations |
| `/integrations/upload` | POST | Upload integration scripts |

## 🔧 Configuration

The `config.yaml` file provides comprehensive configuration options:

```yaml
# Server Settings
server:
  port: 9090
  host: "localhost"

# Security Settings
security:
  api_keys:
    - "secauto-api-key-2024-07-14"
  api_keys_file: "data/security/api_keys.json"
  cors:
    enabled: true
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers: ["Content-Type", "Authorization", "X-API-Key"]

# Database Settings
database:
  redis_url: "redis://localhost:6379/0"
  cache_ttl: 3600
  job_ttl: 86400

# Integration Paths
integrations:
  configs_path: "data/integrations/configs"
  scripts_path: "data/integrations/scripts"
```

## 🚀 New Features

### 🔐 Enterprise Security
- **API Key Authentication**: Cryptographically secure key generation
- **Dynamic Key Management**: Create, list, and manage keys via API
- **Key Persistence**: Automatic saving/loading of generated keys
- **Usage Tracking**: Monitor API key usage patterns

### 🌐 CORS Support
- **Cross-Origin Requests**: Full CORS implementation
- **Configurable Origins**: Support for wildcard and specific domains
- **Preflight Handling**: Automatic OPTIONS request processing
- **Custom Headers**: Support for API key headers

### 📊 Enhanced APIs
- **OpenAPI 3.0**: Complete API documentation with Swagger UI
- **Filtering & Pagination**: Advanced query capabilities
- **Comprehensive Responses**: Detailed success/error responses
- **Real-time Statistics**: Live metrics and analytics

### 🔧 Modular Architecture
- **Package-based Design**: Clean separation of concerns
- **Independent Modules**: Loosely coupled components
- **Easy Extension**: Simple to add new features
- **Test-friendly**: Modular structure supports testing

## 📖 Documentation

Detailed documentation is available in the `READMES/` directory:

- **[API Documentation](READMES/apidoc.json)**: Complete API specification
- **[Cache System](READMES/CACHE_API_README.md)**: Redis caching details
- **[CORS Configuration](READMES/README_CORS.md)**: Cross-origin setup
- **[Plugin Development](READMES/PLUGIN_SYSTEM_README.md)**: Plugin system guide
- **[Docker Setup](READMES/README_DOCKER.md)**: Containerization guide
- **[Redis Integration](READMES/README_REDIS.md)**: Redis configuration

## 🔄 Migration from Legacy

The legacy codebase has been completely refactored into a modular architecture:

### Key Changes:
1. **Monolithic → Modular**: Split into independent packages
2. **File-based Config → YAML**: Centralized configuration management
3. **Basic Auth → API Keys**: Enterprise-grade authentication
4. **No CORS → Full CORS**: Web application support
5. **Limited APIs → Comprehensive**: Full RESTful API suite

### Breaking Changes:
- All endpoints now require API key authentication
- Configuration file format changed to YAML
- API response formats standardized
- Job management completely rewritten

## 🏃‍♂️ Development

### Building
```bash
# Build the binary
go build -o secauto main.go

# Run with hot reload (install air first)
air
```

### Testing
```bash
# Run tests
go test ./...

# Test specific package
go test ./pkg/auth
```

### Contributing
1. Follow the modular architecture patterns
2. Add tests for new functionality
3. Update API documentation
4. Use structured logging

## 🎯 Roadmap

- [ ] Web UI Dashboard
- [ ] Advanced Plugin System
- [ ] Multi-tenant Support
- [ ] Advanced Monitoring
- [ ] GraphQL API
- [ ] Webhook Management UI

## 📄 License

[Add your license information here]

## 🤝 Support

For support and questions:
- Check the documentation in `READMES/`
- Review the API documentation at `/docs`
- Check the health endpoint at `/health`

---

**Version**: 2.0.0 (Modular Architecture)  
**Build Date**: 2025-08-15  
**Go Version**: 1.21+
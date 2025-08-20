# SecAuto - Security Orchestration, Automation & Response Platform

![SecAuto Logo](https://img.shields.io/badge/SecAuto-SOAR-blue)
![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8)
![Python Version](https://img.shields.io/badge/Python-3.9+-3776AB)
![Redis](https://img.shields.io/badge/Redis-7.0+-DC382D)
![License](https://img.shields.io/badge/License-MIT-green)

**SecAuto** is a powerful, enterprise-grade Security Orchestration, Automation, and Response (SOAR) platform designed to streamline cybersecurity operations through intelligent automation, flexible playbooks, and comprehensive integration capabilities.

## 🚀 Key Features

### Core Capabilities
- **🎭 Advanced Playbook Engine**: JSON-based workflow automation with conditional logic, loops, and parallel execution
- **🐍 Python Automation Framework**: Extensible Python scripts with built-in helper libraries
- **🔌 Multi-Platform Plugin System**: Support for Go, Python, Windows, and Linux plugins
- **⚡ High-Performance Caching**: Redis-based cache API with TTL and lazy evaluation
- **🌐 Distributed Architecture**: Multi-node clustering with Redis coordination
- **📊 Comprehensive Job Management**: Async execution, persistence, retry logic, and monitoring
- **⏰ Cron-Based Scheduling**: Automated workflow scheduling with timezone support
- **🔗 Rich Integration Ecosystem**: Pre-built integrations for security tools
- **🪝 Event-Driven Automation**: Webhook system for real-time event processing
- **🛡️ Enterprise Security**: API keys, rate limiting, TLS/HTTPS, client certificates

### Management & Monitoring
- **📈 Performance Optimization**: Async operations, connection pooling, and profiling
- **📝 Advanced Logging**: JSON structured logs with rotation, compression, and filtering
- **🔍 Input Validation**: Comprehensive validation for APIs and playbooks
- **📚 Interactive API Docs**: Swagger UI with live testing
- **🌍 CORS Configuration**: Flexible cross-origin resource sharing
- **🔒 TLS/HTTPS**: Full encryption with auto-certificate management

## 🏗️ System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Client Layer                         │
├──────────────┬──────────────┬──────────────┬──────────────┤
│   Web UI     │   CLI Tools  │  External    │  Monitoring  │
│              │              │  Systems     │   Tools      │
└──────┬───────┴──────┬───────┴──────┬───────┴──────┬───────┘
       │              │              │              │
       └──────────────┴──────────────┴──────────────┘
                             │
                    ┌────────▼────────┐
                    │   API Gateway   │
                    │   (Port 9090)   │
                    └────────┬────────┘
                             │
          ┌──────────────────┼──────────────────┐
          │          Middleware Stack           │
          ├─────────────┬────┴────┬─────────────┤
          │   Auth      │  Rate   │  CORS &     │
          │   System    │  Limit  │  Security   │
          └─────────────┴────┬────┴─────────────┘
                             │
     ┌───────────────────────┼───────────────────────┐
     │                Core Services                  │
     ├──────────┬──────────┬─┴──────────┬──────────┤
     │ Playbook │  Rules   │   Job      │  Cache   │
     │  Engine  │  Engine  │  Manager   │   API    │
     └──────┬───┴──────┬───┴──────┬─────┴──────────┘
            │          │          │
     ┌──────▼──────────▼──────────▼──────┐
     │        Redis Data Store           │
     │   • Jobs  • Cache  • Schedules    │
     │   • Locks • State  • Metadata     │
     └───────────────────────────────────┘
            │                    │
     ┌──────▼──────┐      ┌─────▼──────┐
     │  Python     │      │   Plugin   │
     │ Automations │      │   System   │
     └─────────────┘      └────────────┘
```

## 📦 Installation

### Prerequisites
- **Go 1.22+** - [Download](https://golang.org/dl/)
- **Python 3.9+** - [Download](https://www.python.org/downloads/)
- **Redis 7.0+** - [Download](https://redis.io/download)
- **Git** - [Download](https://git-scm.com/downloads)

### Quick Start

#### 1. Clone Repository
```bash
git clone https://github.com/your-org/secauto.git
cd SecAuto
```

#### 2. Setup Python Environment
```bash
python3 -m venv Venv
source Venv/bin/activate  # On Windows: Venv\Scripts\activate
# Note: No requirements.txt needed - dependencies handled per automation
```

#### 3. Install & Configure Redis
```bash
# macOS
brew install redis
brew services start redis

# Ubuntu/Debian
sudo apt update && sudo apt install redis-server
sudo systemctl start redis-server
sudo systemctl enable redis-server

# Verify Redis
redis-cli ping  # Should return PONG
```

#### 4. Build SecAuto
```bash
cd SoarAuto
go mod tidy
go build -o secauto .        # macOS/Linux
go build -o soarauto.exe .   # Windows
```

#### 5. Configure SecAuto
Edit `SoarAuto/config.yaml`:
```yaml
server:
  port: 9090
  host: "localhost"
  
database:
  redis_url: "redis://localhost:6379/0"
  
security:
  api_keys:
    - "your-secure-api-key-here"
```

#### 6. Run SecAuto
```bash
./secauto       # macOS/Linux
./soarauto.exe  # Windows
```

API available at: `http://localhost:9090`
Documentation at: `http://localhost:9090/docs`

## 🔧 Configuration

### Complete Configuration Example
```yaml
# Server Configuration
server:
  port: 9090
  host: "localhost"
  workers: 5
  read_timeout: "30s"
  write_timeout: "30s"
  
  # TLS/HTTPS Configuration
  tls:
    enabled: false
    port: 9443
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"
    auto_redirect: true
    min_version: "1.2"

# Database Configuration
database:
  redis_url: "redis://localhost:6379/0"
  cache_ttl: 3600      # 1 hour
  job_ttl: 86400       # 24 hours
  temp_data_ttl: 300   # 5 minutes

# Cluster Configuration
cluster:
  enabled: false
  redis_url: "redis://localhost:6379/1"
  node_id: "node-1"
  heartbeat_interval: 30
  job_timeout: 3600

# Security Configuration
security:
  api_keys:
    - "your-api-key-here"
  rate_limiting:
    enabled: true
    requests_per_minute: 100

# Python Environment
python:
  venv_path: "../Venv"

# Logging Configuration
logging:
  level: "INFO"
  destination: "both"  # console, file, or both
  file: "logs/secauto.log"
  format: "json"
  rotation:
    max_size_mb: 10
    max_backups: 5
    max_age_days: 30
    compress: true
  component_levels:
    rules_engine: "WARNING"
    redis_integration: "ERROR"
    job_manager: "INFO"

# Plugin System
plugins:
  enabled: true
  directory: "../plugins"
```

## 🎯 API Reference

### Authentication
All API endpoints (except `/health` and `/docs`) require the `X-API-Key` header:
```bash
curl -H "X-API-Key: your-api-key-here" http://localhost:9090/endpoint
```

### Core Endpoints

#### Playbook Execution
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/playbook` | POST | Execute playbook synchronously |
| `/playbook/async` | POST | Execute playbook asynchronously |
| `/playbook/upload` | POST | Upload new playbook |
| `/playbook/{name}` | DELETE | Delete playbook |
| `/playbooks` | GET | List all playbooks |

#### Job Management
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/jobs` | GET | List all jobs |
| `/jobs/stats` | GET | Job statistics |
| `/job/{id}` | GET | Get job details |
| `/job/{id}` | DELETE | Cancel/delete job |

#### Cache API
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/cache` | GET | Cache information |
| `/cache/stats` | GET | Cache statistics |
| `/cache/clear` | POST | Clear all cache |
| `/cache/{key}` | GET | Retrieve value |
| `/cache/{key}` | POST | Store value |
| `/cache/{key}` | DELETE | Delete value |

#### Automation Management
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/automations` | GET | List automations |
| `/automation` | POST | Upload automation |
| `/automation/{name}` | DELETE | Delete automation |
| `/automation/metadata` | GET/POST | Manage metadata |

#### Schedule Management
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/schedules` | GET/POST | List/create schedules |
| `/schedules/stats` | GET | Schedule statistics |
| `/schedule/{id}` | GET/PUT/DELETE | Manage schedule |
| `/schedule/execute/{id}` | POST | Execute schedule now |

#### Client Management
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/clients` | GET/POST | List/create clients |
| `/clients/{id}` | GET/PUT/DELETE | Manage client |

#### System Endpoints
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check (no auth) |
| `/cluster` | GET | Cluster status |
| `/integrations` | GET | List integrations |
| `/api-keys` | GET/POST/DELETE | Manage API keys |
| `/docs` | GET | Swagger UI (no auth) |

## 🔥 Usage Examples

### Execute a Simple Playbook
```bash
curl -X POST http://localhost:9090/playbook \
  -H "X-API-Key: your-api-key-here" \
  -H "Content-Type: application/json" \
  -d '{
    "playbook": [
      {"run": "data_enrichment"},
      {"run": "threat_analyzer"}
    ],
    "context": {
      "incident_id": "INC-001",
      "severity": "high"
    }
  }'
```

### Conditional Playbook Execution
```bash
curl -X POST http://localhost:9090/playbook \
  -H "X-API-Key: your-api-key-here" \
  -H "Content-Type: application/json" \
  -d '{
    "playbook": [
      {"run": "virustotal_url_scanner", "url": "suspicious.com"},
      {
        "if": {
          "conditions": [[">=", {"var": "malicious_score"}, 5]],
          "true": {"run": "escalate_to_soc"},
          "false": {"run": "log_as_safe"}
        }
      }
    ]
  }'
```

### Async Job Execution
```bash
# Start async job
response=$(curl -X POST http://localhost:9090/playbook/async \
  -H "X-API-Key: your-api-key-here" \
  -H "Content-Type: application/json" \
  -d '{"playbook": [{"run": "long_running_scan"}]}')

job_id=$(echo $response | jq -r '.job_id')

# Check job status
curl -X GET "http://localhost:9090/job/$job_id" \
  -H "X-API-Key: your-api-key-here"
```

### Cache API Usage
```bash
# Store incident data
curl -X POST http://localhost:9090/cache/incident-INC001 \
  -H "X-API-Key: your-api-key-here" \
  -H "Content-Type: application/json" \
  -d '{
    "value": {
      "status": "investigating",
      "assigned_to": "soc-team",
      "priority": "high"
    },
    "ttl": 3600
  }'

# Retrieve incident data
curl -X GET http://localhost:9090/cache/incident-INC001 \
  -H "X-API-Key: your-api-key-here"

# Delete incident data
curl -X DELETE http://localhost:9090/cache/incident-INC001 \
  -H "X-API-Key: your-api-key-here"
```

### Schedule Automation
```bash
curl -X POST http://localhost:9090/schedules \
  -H "X-API-Key: your-api-key-here" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "daily-vulnerability-scan",
    "cron": "0 2 * * *",
    "playbook": [
      {"run": "qualysauto"},
      {"run": "email_notification", "template": "vuln_report"}
    ],
    "enabled": true
  }'
```

## 🏢 Multi-Tenant Support

SecAuto provides enterprise-grade multi-tenant capabilities with complete client isolation:

### Client-Specific Execution
```bash
# Execute playbook for specific client
curl -X POST http://localhost:9090/playbook \
  -H "X-API-Key: your-api-key-here" \
  -H "X-Client-ID: acme-corp" \
  -H "Content-Type: application/json" \
  -d '{
    "playbook": [{"run": "client_aware_demo"}]
  }'
```

### Features
- **🔐 Isolated Credentials**: Per-client API keys and secrets
- **📁 Separate Storage**: Client-specific data directories
- **⚙️ Custom Configuration**: Per-client thresholds and policies
- **🔄 Automatic Fallback**: Global config when client-specific unavailable
- **📊 Audit Trails**: Complete per-client activity logging

## 🛠️ Development

### Creating Custom Automations

#### Python Automation Template
```python
#!/usr/bin/env python3
import json
import sys
from server.SoarBaseAPI import load_context, return_context

def main():
    # Load context from SecAuto
    context = load_context()
    if not context and len(sys.argv) > 1:
        context = json.loads(sys.argv[1])
    
    # Your automation logic
    result = process_security_event(context)
    
    # Return results to SecAuto
    return_context(result)

def process_security_event(context):
    # Implementation here
    return {"success": True, "data": processed_data}

if __name__ == "__main__":
    main()
```

#### Playbook Definition
```json
[
  {
    "run": "data_enrichment",
    "timeout": 30
  },
  {
    "parallel": [
      {"run": "virustotal_scan"},
      {"run": "threat_intel_lookup"}
    ]
  },
  {
    "if": {
      "conditions": [["==", {"var": "threat_level"}, "critical"]],
      "true": {
        "sequential": [
          {"run": "isolate_host"},
          {"run": "notify_soc"},
          {"cache": {"key": "critical_incident", "value": {"var": "incident_data"}}}
        ]
      },
      "false": {"run": "log_event"}
    }
  }
]
```

### Project Structure
```
SecAuto/
├── SoarAuto/                    # Go server application
│   ├── main.go                  # Main entry point & API handlers
│   ├── config.yaml              # Configuration file
│   ├── pkg/                     # Go packages
│   │   ├── auth/               # Authentication system
│   │   ├── automations/        # Automation management
│   │   ├── cache/              # Cache implementation
│   │   ├── clients/            # Client management
│   │   ├── cluster/            # Clustering support
│   │   ├── config/             # Configuration management
│   │   ├── errors/             # Error handling
│   │   ├── integrations/       # Integration framework
│   │   ├── jobs/               # Job management
│   │   ├── logger/             # Logging system
│   │   ├── performance/        # Performance utilities
│   │   ├── playbooks/          # Playbook management
│   │   ├── recovery/           # Panic recovery
│   │   ├── redis/              # Redis client
│   │   ├── rules/              # Rules engine
│   │   ├── schedules/          # Scheduling system
│   │   ├── security/           # Security middleware
│   │   ├── swagger/            # API documentation
│   │   ├── tls/                # TLS configuration
│   │   ├── types/              # Type definitions
│   │   └── validator/          # Input validation
│   └── data/                    # Runtime data
├── automations/                 # Python automation scripts
│   ├── client_virustotal_scanner.py
│   ├── data_enrichment.py
│   ├── email_notification.py
│   ├── threat_analyzer.py
│   └── ...
├── playbooks/                   # JSON playbook definitions
├── integrations/                # Integration modules
├── plugins/                     # Plugin system
│   ├── go/                     # Go plugins
│   ├── python/                  # Python plugins
│   ├── windows/                 # Windows-specific
│   └── linux/                   # Linux-specific
├── server/                      # Python support libraries
│   └── SoarBaseAPI.py          # Helper functions
├── logs/                        # Application logs
├── Venv/                        # Python virtual environment
└── CLAUDE.md                    # AI assistant instructions
```

## 🔒 Security Features

### Authentication & Authorization
- **API Key Management**: Secure key generation and rotation
- **Client Certificates**: Mutual TLS authentication
- **Role-Based Access**: Granular permission system

### Network Security
- **TLS/HTTPS**: Full encryption with TLS 1.2+
- **Rate Limiting**: DDoS protection and API throttling
- **CORS Protection**: Configurable origin policies
- **Security Headers**: HSTS, CSP, X-Frame-Options

### Data Protection
- **Input Validation**: Comprehensive request validation
- **Secure Storage**: Encrypted sensitive data
- **Audit Logging**: Complete activity tracking

## 📊 Monitoring & Observability

### Logging
- JSON structured logging
- Component-level log filtering
- Automatic log rotation and compression
- Centralized log aggregation support

### Metrics
- Job execution metrics
- API response times
- Cache hit/miss rates
- Error tracking and alerting

### Health Monitoring
- `/health` endpoint for liveness checks
- Dependency health verification
- Cluster node status tracking

## 🚀 Production Deployment

### Docker Deployment
```bash
# Build Docker image
docker build -t secauto:latest .

# Run with Docker Compose
docker-compose up -d
```

### Kubernetes Deployment
```yaml
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
        - containerPort: 9090
        env:
        - name: REDIS_URL
          value: "redis://redis-service:6379"
```

### High Availability Setup
- Multiple SecAuto nodes
- Redis Sentinel for failover
- Load balancer configuration
- Shared storage for plugins/automations

## 📚 Documentation

- **[API Documentation](http://localhost:9090/docs)** - Interactive Swagger UI
- **[CLAUDE.md](CLAUDE.md)** - Development guide for AI assistants
- **[Cache API Guide](SoarAuto/READMES/CACHE_API_README.md)** - Redis cache usage
- **[HTTPS Setup](SoarAuto/READMES/HTTPS_SETUP_README.md)** - TLS configuration
- **[Plugin Development](SoarAuto/READMES/PLUGIN_SYSTEM_DEVELOPEMENT_README.md)** - Creating plugins
- **[Distributed Systems](SoarAuto/READMES/DISTRIBUTED_SYSTEM_README.md)** - Clustering guide

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Go community for excellent libraries and tools
- Redis team for high-performance data storage
- Python community for automation capabilities
- Security community for best practices and inspiration

## 📞 Support

- **Documentation**: Check `/docs` endpoint when running
- **Issues**: [GitHub Issues](https://github.com/your-org/secauto/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/secauto/discussions)
- **Security**: Report vulnerabilities to security@your-org.com

---

**Built with ❤️ for the Cybersecurity Community**

*SecAuto - Automate Today, Secure Tomorrow*
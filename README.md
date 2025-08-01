# SecAuto - Security Orchestration, Automation & Response Platform

![SecAuto Logo](https://img.shields.io/badge/SecAuto-SOAR-blue)
![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8)
![Python Version](https://img.shields.io/badge/Python-3.9+-3776AB)
![Redis](https://img.shields.io/badge/Redis-Cache-DC382D)
![License](https://img.shields.io/badge/License-MIT-green)

**SecAuto** is a powerful, scalable Security Orchestration, Automation, and Response (SOAR) platform designed to streamline cybersecurity operations through intelligent automation, flexible playbooks, and comprehensive integration capabilities.

## 🚀 Features

### Core Capabilities
- **🎭 Playbook Engine**: JSON-based workflow automation with complex logic support
- **🔧 Automation Scripts**: Python-based automation with rich integration libraries
- **🔌 Plugin System**: Extensible plugin architecture for Go, Python, and platform-specific plugins
- **⚡ Redis Cache API**: High-performance caching for automation results and data sharing
- **🌐 Distributed Clustering**: Multi-node deployment with Redis-based coordination
- **📊 Job Management**: Asynchronous job execution with persistence and monitoring
- **⏰ Job Scheduling**: Cron-based scheduling system for automated workflows
- **🔗 Integration Framework**: Seamless integration with external security tools
- **🪝 Webhook System**: Real-time notifications and event-driven automation
- **🛡️ Security Features**: API key authentication, rate limiting, input validation

### Management & Monitoring
- **📈 Performance Metrics**: Comprehensive monitoring and performance tracking
- **📝 Structured Logging**: Advanced logging with rotation and filtering
- **🔍 Validation System**: Input validation and playbook verification
- **📚 API Documentation**: Interactive Swagger UI documentation
- **🌍 CORS Support**: Full cross-origin resource sharing support

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web UI/CLI    │    │   External      │    │   Automation    │
│                 │    │   Integrations  │    │   Scripts       │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
          ┌─────────────────────────────────────────────┐
          │             SecAuto API Layer               │
          │  ┌─────────┐ ┌─────────┐ ┌─────────────────┐│
          │  │  Auth   │ │  Rate   │ │   Validation    ││
          │  │ System  │ │Limiting │ │    System       ││
          │  └─────────┘ └─────────┘ └─────────────────┘│
          └─────────────────────┬───────────────────────┘
                                │
    ┌───────────────────────────┼───────────────────────────┐
    │                          │                           │
┌───▼────┐              ┌──────▼──────┐              ┌─────▼─────┐
│Playbook│              │    Rules    │              │   Cache   │
│Engine  │◄────────────►│   Engine    │◄────────────►│   API     │
└────────┘              └─────┬───────┘              └───────────┘
    │                         │                           │
    │                    ┌────▼────┐                      │
    │                    │   Job   │                      │
    │                    │ Manager │                      │
    │                    └────┬────┘                      │
    │                         │                           │
    └─────────────────────────┼───────────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │      Redis        │
                    │   Data Store      │
                    └───────────────────┘
```

## 📦 Installation

### Prerequisites
- **Go 1.22+**
- **Python 3.9+**
- **Redis Server**
- **Git**

### Quick Start

1. **Clone the Repository**
```bash
git clone https://github.com/your-org/secauto.git
cd secauto
```

2. **Setup Python Virtual Environment**
```bash
python3 -m venv Venv
source Venv/bin/activate  # On Windows: Venv\Scripts\activate
pip install -r requirements.txt
```

3. **Configure Redis**
```bash
# Install Redis (Ubuntu/Debian)
sudo apt update && sudo apt install redis-server

# Start Redis
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

4. **Build SecAuto**
```bash
cd SoarAuto
go mod tidy
go build -o soarauto.exe .
```

5. **Configure SecAuto**
```bash
# Edit configuration
nano config.yaml
```

6. **Run SecAuto**
```bash
./soarauto.exe
```

The API will be available at `http://localhost:8000`

## 🔧 Configuration

SecAuto uses a comprehensive YAML configuration file (`config.yaml`):

```yaml
# Server Configuration
server:
  port: "8000"
  
# Database Configuration  
database:
  redis_url: "redis://localhost:6379/0"
  
# Security Configuration
security:
  api_keys:
    - "secauto-api-key-2024-07-14"
  rate_limiting:
    enabled: true
    requests_per_minute: 100
    endpoints:
      cache: 200
      playbook: 50
      
# Python Environment
python:
  venv_path: "../Venv"
  
# Logging
logging:
  level: "info"
  file: "logs/secauto.log"
  max_size: 100
  max_backups: 5
```

## 🎯 API Endpoints

### Core APIs

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/playbook` | POST | Execute playbook (sync) |
| `/playbook/async` | POST | Execute playbook (async) |
| `/jobs` | GET | List all jobs |
| `/job/{id}` | GET | Get job status |

### Cache API (🆕)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/cache` | GET | Get cache information |
| `/cache/{key}` | GET | Retrieve cached value |
| `/cache/{key}` | POST | Store value in cache |
| `/cache/{key}` | DELETE | Delete cached value |

### Management APIs

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/automations` | GET | List automations |
| `/automation` | POST | Upload automation |
| `/playbooks` | GET | List playbooks |
| `/integrations` | GET | List integrations |
| `/plugins` | GET | List plugins |

### Documentation
- **`/docs`** - Interactive Swagger UI
- **`/api-docs`** - OpenAPI specification

## 🔥 Quick Examples

### 1. Execute a Playbook
```bash
curl -X POST http://localhost:8000/playbook \
  -H "X-API-Key: secauto-api-key-2024-07-14" \
  -H "Content-Type: application/json" \
  -d '{
    "playbook": [
      {"run": "data_enrichment"},
      {"run": "virustotal_url_scanner", "urls": ["malicious.com"]}
    ],
    "context": {
      "incident_id": "INC-001",
      "severity": "high"
    }
  }'
```

### 2. Cache API Usage
```bash
# Store data in cache
curl -X POST http://localhost:8000/cache/incident-data \
  -H "X-API-Key: secauto-api-key-2024-07-14" \
  -H "Content-Type: application/json" \
  -d '{"value": {"incident_id": "INC-001", "status": "active"}}'

# Retrieve data from cache
curl -X GET http://localhost:8000/cache/incident-data \
  -H "X-API-Key: secauto-api-key-2024-07-14"
```

### 3. Job Management
```bash
# Get all jobs
curl -X GET http://localhost:8000/jobs \
  -H "X-API-Key: secauto-api-key-2024-07-14"

# Get specific job status
curl -X GET http://localhost:8000/job/job-123 \
  -H "X-API-Key: secauto-api-key-2024-07-14"
```

## 📚 Documentation

### Comprehensive Guides
- **[Cache API Documentation](SoarAuto/READMES/CACHE_API_README.md)** - Complete Redis cache API guide
- **[Plugin Development](SoarAuto/READMES/PLUGIN_SYSTEM_DEVELOPEMENT_README.md)** - Building custom plugins
- **[Integration Development](SoarAuto/READMES/CONFIG_FILE_INTEGRATION.md)** - Creating integrations
- **[Distributed System](SoarAuto/READMES/DISTRIBUTED_SYSTEM_README.md)** - Multi-node deployment
- **[Docker Deployment](SoarAuto/READMES/README_DOCKER.md)** - Container deployment guide

### API Documentation
- **Interactive Docs**: Visit `http://localhost:8000/docs` when running
- **OpenAPI Spec**: Available at `http://localhost:8000/api-docs`

## 🔌 Integrations

SecAuto supports a wide range of security tool integrations:

### Threat Intelligence
- **VirusTotal** - URL/file scanning and reputation checks
- **Qualys** - Vulnerability management and scanning
- **Custom APIs** - Extensible integration framework

### Communication & Notifications
- **Webhooks** - Real-time event notifications
- **Custom Integrations** - Build your own integration modules

### Data Storage & Caching
- **Redis** - High-performance caching and job storage
- **File System** - Local storage for automations and playbooks

## 🛠️ Development

### Project Structure
```
SecAuto/
├── SoarAuto/                 # Go server source
│   ├── main.go              # Main application entry
│   ├── config.yaml          # Configuration file
│   ├── redis_integration.go # Cache API implementation
│   ├── rules_engine.go      # Playbook execution engine
│   └── READMES/             # Documentation
├── automations/             # Python automation scripts
├── integrations/            # Integration modules
├── playbooks/              # JSON playbook definitions
├── plugins/                # Plugin system
└── Venv/                   # Python virtual environment
```

### Adding New Features

1. **Create Automation Scripts**
```python
# automations/my_automation.py
import json
import sys

def main():
    context = json.loads(sys.argv[1])
    
    # Your automation logic here
    result = {"processed": True, "data": context}
    
    print(json.dumps(result))

if __name__ == "__main__":
    main()
```

2. **Create Playbooks**
```json
[
  {"run": "data_enrichment"},
  {"run": "my_automation"},
  {
    "if": {
      "conditions": [["==", {"var": "severity"}, "high"]],
      "true": {"run": "escalate_incident"},
      "false": {"run": "log_incident"}
    }
  }
]
```

3. **Add Integrations**
```python
# integrations/my_service_integration.py
class MyServiceIntegration:
    def __init__(self, config_name="my_service"):
        # Integration initialization
        pass
    
    def query_api(self, query):
        # API interaction logic
        return {"success": True, "data": []}
```

## 🔒 Security

- **API Key Authentication**: Required for all endpoints
- **Rate Limiting**: Configurable per-endpoint rate limits
- **Input Validation**: Comprehensive request validation
- **CORS Protection**: Configurable cross-origin policies
- **Secure Headers**: Security-focused HTTP headers

## 📊 Monitoring

### Logging
- **Structured Logging**: JSON-formatted logs with context
- **Log Rotation**: Automatic log file rotation
- **Multiple Levels**: Debug, Info, Warning, Error levels

### Metrics
- **Job Metrics**: Execution times, success rates, error tracking
- **Performance Monitoring**: Response times, throughput monitoring
- **Health Checks**: System health and dependency status

## 🚀 Production Deployment

### Docker Deployment
```bash
# Build container
docker build -t secauto .

# Run with Redis
docker-compose up -d
```

### Distributed Deployment
```yaml
# Multiple nodes with shared Redis
cluster:
  enabled: true
  node_id: "node-1"
  redis_url: "redis://redis-cluster:6379/0"
```

### Load Balancing
```nginx
upstream secauto {
    server 127.0.0.1:8000;
    server 127.0.0.1:8001;
    server 127.0.0.1:8002;
}
```

## 🤝 Contributing

1. **Fork the Repository**
2. **Create Feature Branch** (`git checkout -b feature/amazing-feature`)
3. **Commit Changes** (`git commit -m 'Add amazing feature'`)
4. **Push to Branch** (`git push origin feature/amazing-feature`)
5. **Open Pull Request**

### Development Guidelines
- Follow Go best practices and idioms
- Include comprehensive tests
- Update documentation for new features
- Ensure backward compatibility

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Go Community** - For excellent libraries and tools
- **Redis Team** - For high-performance data storage
- **Security Community** - For inspiration and best practices

## 📞 Support

- **Documentation**: Check the `/docs` endpoint when running
- **Issues**: Use GitHub Issues for bug reports
- **Community**: Join our discussion forums

---

**Built with ❤️ for the cybersecurity community**

*SecAuto - Automate Today, Secure Tomorrow*
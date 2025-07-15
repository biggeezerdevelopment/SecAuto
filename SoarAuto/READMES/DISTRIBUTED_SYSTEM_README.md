# SecAuto Distributed/Cluster Mode

SecAuto now supports distributed operation with multiple nodes for high availability, load balancing, and fault tolerance. The distributed system uses Redis as a centralized job queue and coordination mechanism.

## 🏗️ Architecture Overview

### Key Components

- **Redis-based Job Queue**: Centralized job dispatching and storage
- **Node Discovery**: Automatic discovery of cluster nodes
- **Health Monitoring**: Real-time node health and load monitoring
- **Load Balancing**: Intelligent job distribution based on node load
- **Fault Tolerance**: Automatic failover and job retry mechanisms

### Cluster Topology

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   SecAuto Node  │    │   SecAuto Node  │    │   SecAuto Node  │
│   (Primary)     │    │   (Secondary)   │    │   (Secondary)   │
│                 │    │                 │    │                 │
│ • Job Processor │    │ • Job Processor │    │ • Job Processor │
│ • Health Check  │    │ • Health Check  │    │ • Health Check  │
│ • Node Discovery│    │ • Node Discovery│    │ • Node Discovery│
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │      Redis      │
                    │                 │
                    │ • Job Queue     │
                    │ • Node Registry │
                    │ • State Storage │
                    └─────────────────┘
```

## ⚙️ Configuration

### Enable Cluster Mode

Add the following to your `config.yaml`:

```yaml
cluster:
  enabled: true
  redis_url: "localhost:6379"
  redis_db: 0
  redis_password: ""
  node_id: ""  # Auto-generated if empty
  cluster_name: "secauto-cluster"
  heartbeat_interval: 30  # seconds
  election_timeout: 60    # seconds
  job_timeout: 300        # seconds
  max_retries: 3
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `enabled` | Enable cluster mode | `false` |
| `redis_url` | Redis server address | `localhost:6379` |
| `redis_db` | Redis database number | `0` |
| `redis_password` | Redis password | `""` |
| `node_id` | Unique node identifier | Auto-generated |
| `cluster_name` | Cluster name for isolation | `secauto-cluster` |
| `heartbeat_interval` | Node heartbeat interval (seconds) | `30` |
| `election_timeout` | Leader election timeout (seconds) | `60` |
| `job_timeout` | Job execution timeout (seconds) | `300` |
| `max_retries` | Maximum job retry attempts | `3` |

## 🚀 Getting Started

### 1. Install Redis

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install redis-server
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

**Windows:**
```bash
# Download Redis for Windows from https://github.com/microsoftarchive/redis/releases
# Or use WSL with Ubuntu or docker
```

**macOS:**
```bash
brew install redis
brew services start redis
```

### 2. Configure SecAuto for Cluster Mode

Create or update your `config.yaml`:

```yaml
server:
  port: 8080
  host: "localhost"

cluster:
  enabled: true
  redis_url: "localhost:6379"
  cluster_name: "my-secauto-cluster"
  heartbeat_interval: 30
  job_timeout: 300
  max_retries: 3

logging:
  level: "info"
  destination: "stdout"
```

### 3. Start Multiple Nodes

**Node 1 (Primary):**
```bash
./secauto.exe
```

**Node 2 (Secondary):**
```bash
# Update config.yaml to use different port
server:
  port: 8081

# Start the second node
./secauto.exe
```

**Node 3 (Secondary):**
```bash
# Update config.yaml to use different port
server:
  port: 8082

# Start the third node
./secauto.exe
```

## 📊 Cluster Management API

### Get Cluster Information

```bash
curl -X GET http://localhost:8080/cluster \
  -H "Authorization: Bearer YOUR_API_KEY"
```

Response:
```json
{
  "success": true,
  "cluster": {
    "cluster_name": "secauto-cluster",
    "node_id": "node-1",
    "total_nodes": 3,
    "nodes": [
      {
        "id": "node-1",
        "host": "localhost",
        "port": 8080,
        "status": "active",
        "started_at": "2024-01-15T10:30:00Z",
        "last_seen": "2024-01-15T10:35:00Z",
        "capabilities": ["playbook_execution", "job_processing"],
        "load": 0.3,
        "jobs_running": 3,
        "jobs_completed": 150,
        "jobs_failed": 2
      }
    ],
    "queue_size": 5
  },
  "timestamp": "2024-01-15T10:35:00Z"
}
```

### Submit Job to Distributed Queue

```bash
curl -X POST http://localhost:8080/cluster/jobs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "playbook": [
      {
        "action": "run_script",
        "script": "baseit.py",
        "parameters": {
          "target": "192.168.1.1"
        }
      }
    ],
    "context": {
      "environment": "production"
    }
  }'
```

Response:
```json
{
  "success": true,
  "job_id": "job-12345",
  "status": "submitted",
  "timestamp": "2024-01-15T10:35:00Z"
}
```

### Get Distributed Job Status

```bash
curl -X GET http://localhost:8080/cluster/jobs/job-12345 \
  -H "Authorization: Bearer YOUR_API_KEY"
```

Response:
```json
{
  "success": true,
  "job": {
    "id": "job-12345",
    "playbook": [...],
    "context": {...},
    "status": "completed",
    "submitted_at": "2024-01-15T10:35:00Z",
    "started_at": "2024-01-15T10:35:05Z",
    "completed_at": "2024-01-15T10:35:30Z",
    "assigned_to": "node-2",
    "results": [...],
    "retry_count": 0,
    "priority": 0,
    "tags": []
  },
  "timestamp": "2024-01-15T10:35:30Z"
}
```

## 🔄 Job Processing Flow

### 1. Job Submission
1. Client submits job to any cluster node
2. Job is validated and stored in Redis queue
3. Job is assigned a unique ID and priority

### 2. Job Distribution
1. Available nodes poll the Redis queue
2. Jobs are distributed based on node load and priority
3. Node with lowest load gets the job

### 3. Job Execution
1. Selected node picks up the job
2. Job status is updated to "running"
3. Playbook is executed using the rule engine
4. Results are stored back to Redis

### 4. Job Completion
1. Job status is updated to "completed" or "failed"
2. Results are stored in Redis
3. Node statistics are updated
4. Webhook notifications are sent (if configured)

## 🛡️ Fault Tolerance

### Node Failure Handling

- **Automatic Detection**: Nodes are marked inactive after 3 missed heartbeats
- **Job Recovery**: Failed jobs are automatically retried on other nodes
- **Load Redistribution**: Jobs from failed nodes are redistributed

### Job Retry Logic

- **Retry Count**: Configurable maximum retry attempts
- **Exponential Backoff**: Increasing delays between retries
- **Dead Letter Queue**: Jobs that exceed max retries are moved to DLQ

### Data Persistence

- **Redis Persistence**: Jobs and node state persist across restarts
- **Job History**: Completed jobs are retained for 24 hours
- **Node Registry**: Node information is automatically cleaned up

## 📈 Monitoring and Observability

### Cluster Metrics

- **Node Health**: Real-time status of all nodes
- **Queue Size**: Number of pending jobs
- **Load Distribution**: Jobs per node and load balancing
- **Job Statistics**: Success/failure rates and execution times

### Health Checks

```bash
# Check cluster health
curl -X GET http://localhost:8080/cluster \
  -H "Authorization: Bearer YOUR_API_KEY"

# Check individual node health
curl -X GET http://localhost:8080/health \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Logging

Cluster events are logged with structured logging:

```json
{
  "timestamp": "2024-01-15T10:35:00Z",
  "level": "info",
  "component": "cluster_manager",
  "message": "Job submitted to distributed queue",
  "job_id": "job-12345",
  "node_id": "node-1"
}
```

## 🔧 Troubleshooting

### Common Issues

1. **Redis Connection Failed**
   - Verify Redis is running: `redis-cli ping`
   - Check Redis URL in config
   - Ensure network connectivity

2. **Node Not Joining Cluster**
   - Check node ID uniqueness
   - Verify cluster name matches
   - Check Redis permissions

3. **Jobs Not Processing**
   - Check node health status
   - Verify job queue size
   - Check node load and capacity

4. **High Job Failure Rate**
   - Review job timeout settings
   - Check node resources
   - Verify playbook syntax

### Debug Commands

```bash
# Check Redis queue
redis-cli ZCARD secauto:jobs:my-secauto-cluster

# List cluster nodes
redis-cli KEYS "secauto:nodes:my-secauto-cluster:*"

# Check job data
redis-cli GET "secauto:job:job-12345"
```

## 🚀 Performance Optimization

### Scaling Recommendations

1. **Node Sizing**
   - CPU: 2-4 cores per node
   - Memory: 4-8 GB per node
   - Storage: SSD for better I/O

2. **Redis Optimization**
   - Enable Redis persistence
   - Configure appropriate memory limits
   - Use Redis clustering for high availability

3. **Network Configuration**
   - Low-latency network between nodes
   - Dedicated network for Redis
   - Proper firewall configuration

### Load Balancing

- **Round Robin**: Jobs distributed evenly
- **Load-Based**: Jobs sent to least loaded nodes
- **Priority Queue**: High-priority jobs processed first
- **Sticky Sessions**: Related jobs sent to same node

## 🔮 Future Enhancements

### Planned Features

- **Leader Election**: Automatic cluster leader selection
- **Job Scheduling**: Advanced job scheduling with cron-like syntax
- **Resource Quotas**: Per-node resource limits and quotas
- **Job Dependencies**: Job chaining and dependency management
- **Advanced Monitoring**: Prometheus metrics and Grafana dashboards
- **Multi-Region**: Cross-region cluster support
- **Auto-Scaling**: Automatic node scaling based on load

### Integration Possibilities

- **Kubernetes**: Native Kubernetes deployment
- **Docker Swarm**: Docker Swarm integration
- **Cloud Providers**: AWS, Azure, GCP integration
- **Service Mesh**: Istio/Linkerd integration

## 📚 Examples

### Multi-Node Deployment

```yaml
# docker-compose.yml
version: '3.8'
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  secauto-node1:
    build: .
    ports:
      - "8080:8080"
    environment:
      - REDIS_URL=redis:6379
    depends_on:
      - redis

  secauto-node2:
    build: .
    ports:
      - "8081:8080"
    environment:
      - REDIS_URL=redis:6379
    depends_on:
      - redis

  secauto-node3:
    build: .
    ports:
      - "8082:8080"
    environment:
      - REDIS_URL=redis:6379
    depends_on:
      - redis

volumes:
  redis_data:
```

### Load Balancer Configuration

```nginx
# nginx.conf
upstream secauto_cluster {
    server localhost:8080;
    server localhost:8081;
    server localhost:8082;
}

server {
    listen 80;
    server_name secauto.example.com;

    location / {
        proxy_pass http://secauto_cluster;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 🤝 Contributing

When contributing to the distributed system:

1. Follow the existing code patterns
2. Add comprehensive tests for new features
3. Update documentation for API changes
4. Consider backward compatibility
5. Test with multiple nodes
6. Monitor performance impact

## 📄 License

This distributed system is part of SecAuto and follows the same license terms. 
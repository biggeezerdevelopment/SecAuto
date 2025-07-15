# SecAuto Docker Setup

This document describes how to run SecAuto with Redis using Docker for development and testing.

## Quick Start

### Option 1: Redis Only in Docker (Recommended for Development)

1. **Start Redis in Docker:**
   ```powershell
   .\start_redis_docker.ps1 -StartRedis
   ```

2. **Build and run SecAuto locally:**
   ```powershell
   go build -o secauto.exe .
   .\secauto.exe
   ```

### Option 2: Full Docker Compose Setup

1. **Start both Redis and SecAuto:**
   ```powershell
   docker-compose up -d
   ```

2. **View logs:**
   ```powershell
   docker-compose logs -f
   ```

3. **Stop services:**
   ```powershell
   docker-compose down
   ```

## Docker Commands

### Redis Management

```powershell
# Start Redis
.\start_redis_docker.ps1 -StartRedis

# Stop Redis
.\start_redis_docker.ps1 -StopRedis

# Restart Redis
.\start_redis_docker.ps1 -RestartRedis

# Check status
.\start_redis_docker.ps1 -CheckStatus

# View logs
.\start_redis_docker.ps1 -ViewLogs
```

### Docker Compose Commands

```bash
# Start all services
docker-compose up -d

# Start with logs
docker-compose up

# Stop all services
docker-compose down

# View logs
docker-compose logs

# View specific service logs
docker-compose logs redis
docker-compose logs secauto

# Rebuild and start
docker-compose up -d --build

# Remove volumes (clean slate)
docker-compose down -v
```

## Configuration

### Redis Configuration

The Redis container is configured with:
- **Port**: 6379 (mapped to localhost:6379)
- **Memory**: 256MB max
- **Persistence**: AOF enabled
- **Policy**: LRU eviction

### SecAuto Configuration

The application is configured to connect to Redis at `localhost:6379`:
```yaml
database:
  type: "redis"
  redis_url: "redis://localhost:6379/0"
```

## Development Workflow

### 1. Start Redis
```powershell
.\start_redis_docker.ps1 -StartRedis
```

### 2. Build and Run SecAuto
```powershell
go build -o secauto.exe .
.\secauto.exe
```

### 3. Test the API
```powershell
# Test health endpoint
curl http://localhost:8080/health

# Test async playbook
curl -X POST http://localhost:8080/playbook/async \
  -H "X-API-Key: your-secure-api-key-here" \
  -H "Content-Type: application/json" \
  -d '{
    "playbook": [{"run": "baseit"}],
    "context": {"test": true}
  }'
```

### 4. Monitor Jobs
```powershell
# List jobs
curl http://localhost:8080/jobs

# Get job stats
curl http://localhost:8080/jobs/stats
```

## Troubleshooting

### Redis Connection Issues

1. **Check if Redis is running:**
   ```powershell
   .\start_redis_docker.ps1 -CheckStatus
   ```

2. **View Redis logs:**
   ```powershell
   .\start_redis_docker.ps1 -ViewLogs
   ```

3. **Restart Redis:**
   ```powershell
   .\start_redis_docker.ps1 -RestartRedis
   ```

### Docker Issues

1. **Check Docker status:**
   ```powershell
   docker --version
   docker ps
   ```

2. **Clean up containers:**
   ```powershell
   docker stop redis-secauto
   docker rm redis-secauto
   ```

3. **Check Docker logs:**
   ```powershell
   docker logs redis-secauto
   ```

### SecAuto Issues

1. **Check application logs:**
   ```powershell
   Get-Content logs/secauto.log -Tail 50
   ```

2. **Test Redis connection:**
   ```powershell
   docker exec redis-secauto redis-cli ping
   ```

3. **Check Redis data:**
   ```powershell
   docker exec redis-secauto redis-cli keys "*"
   ```

## Production Deployment

For production, consider:

1. **Redis Configuration:**
   - Use Redis Cluster for high availability
   - Configure proper authentication
   - Set up monitoring and alerting

2. **SecAuto Configuration:**
   - Use environment variables for sensitive data
   - Configure proper logging
   - Set up health checks

3. **Docker Compose Production:**
   ```yaml
   version: '3.8'
   services:
     redis:
       image: redis:7-alpine
       ports:
         - "6379:6379"
       volumes:
         - redis_data:/data
       command: redis-server --requirepass your_password
       restart: unless-stopped
       
     secauto:
       build: .
       ports:
         - "8080:8080"
       environment:
         - REDIS_URL=redis://:your_password@redis:6379/0
       depends_on:
         - redis
       restart: unless-stopped
   ```

## Benefits of Docker Setup

1. **Isolation**: Redis runs in its own container
2. **Consistency**: Same environment across development and production
3. **Easy Management**: Simple commands to start/stop services
4. **No Installation**: No need to install Redis locally
5. **Clean Environment**: Easy to reset and start fresh

## Performance Considerations

1. **Memory**: Redis container is limited to 256MB
2. **Persistence**: Data is persisted in Docker volumes
3. **Networking**: Minimal overhead for local development
4. **Scaling**: Can easily scale with Docker Compose

## Security Notes

1. **Development Only**: This setup is for development/testing
2. **No Authentication**: Redis runs without password in development
3. **Local Access**: Only accessible from localhost
4. **Production**: Use proper authentication and security measures 
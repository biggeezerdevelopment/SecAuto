# SecAuto Redis Docker Setup Script
# This script helps set up Redis in Docker for SecAuto development

param(
    [switch]$StartRedis,
    [switch]$StopRedis,
    [switch]$RestartRedis,
    [switch]$CheckStatus,
    [switch]$ViewLogs
)

Write-Host "SecAuto Redis Docker Setup" -ForegroundColor Green
Write-Host "=========================" -ForegroundColor Green

# Function to check if Docker is available
function Test-DockerAvailable {
    try {
        $dockerVersion = docker --version 2>$null
        if ($dockerVersion) {
            Write-Host "✓ Docker is available: $dockerVersion" -ForegroundColor Green
            return $true
        }
    }
    catch {
        Write-Host "✗ Docker is not available" -ForegroundColor Red
        return $false
    }
    return $false
}

# Function to check if Redis container exists
function Test-RedisContainer {
    $container = docker ps -a --filter "name=redis-secauto" --format "{{.Names}}" 2>$null
    return $container -eq "redis-secauto"
}

# Function to check if Redis container is running
function Test-RedisRunning {
    $container = docker ps --filter "name=redis-secauto" --format "{{.Names}}" 2>$null
    return $container -eq "redis-secauto"
}

# Function to start Redis container
function Start-RedisContainer {
    Write-Host "Starting Redis container..." -ForegroundColor Yellow
    
    if (Test-RedisContainer) {
        # Container exists, start it
        docker start redis-secauto 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Redis container started" -ForegroundColor Green
            return $true
        } else {
            Write-Host "✗ Failed to start Redis container" -ForegroundColor Red
            return $false
        }
    } else {
        # Create and start new container
        docker run --name redis-secauto -p 6379:6379 -d redis:7-alpine redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Redis container created and started" -ForegroundColor Green
            return $true
        } else {
            Write-Host "✗ Failed to create Redis container" -ForegroundColor Red
            return $false
        }
    }
}

# Function to stop Redis container
function Stop-RedisContainer {
    Write-Host "Stopping Redis container..." -ForegroundColor Yellow
    
    if (Test-RedisRunning) {
        docker stop redis-secauto 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Redis container stopped" -ForegroundColor Green
            return $true
        } else {
            Write-Host "✗ Failed to stop Redis container" -ForegroundColor Red
            return $false
        }
    } else {
        Write-Host "Redis container is not running" -ForegroundColor Yellow
        return $true
    }
}

# Function to restart Redis container
function Restart-RedisContainer {
    Write-Host "Restarting Redis container..." -ForegroundColor Yellow
    
    if (Test-RedisContainer) {
        docker restart redis-secauto 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Redis container restarted" -ForegroundColor Green
            return $true
        } else {
            Write-Host "✗ Failed to restart Redis container" -ForegroundColor Red
            return $false
        }
    } else {
        Write-Host "Redis container does not exist, creating new one..." -ForegroundColor Yellow
        return Start-RedisContainer
    }
}

# Function to check Redis status
function Show-RedisStatus {
    Write-Host "Redis Container Status:" -ForegroundColor Yellow
    
    if (Test-RedisContainer) {
        $containerInfo = docker ps -a --filter "name=redis-secauto" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>$null
        Write-Host $containerInfo -ForegroundColor Cyan
        
        if (Test-RedisRunning) {
            Write-Host "✓ Redis is running and accessible on localhost:6379" -ForegroundColor Green
            
            # Test Redis connection
            $testResult = docker exec redis-secauto redis-cli ping 2>$null
            if ($testResult -eq "PONG") {
                Write-Host "✓ Redis connection test successful" -ForegroundColor Green
            } else {
                Write-Host "✗ Redis connection test failed" -ForegroundColor Red
            }
        } else {
            Write-Host "✗ Redis container is not running" -ForegroundColor Red
        }
    } else {
        Write-Host "✗ Redis container does not exist" -ForegroundColor Red
    }
}

# Function to view Redis logs
function Show-RedisLogs {
    Write-Host "Redis Container Logs:" -ForegroundColor Yellow
    
    if (Test-RedisContainer) {
        docker logs redis-secauto --tail 20 2>$null
    } else {
        Write-Host "Redis container does not exist" -ForegroundColor Red
    }
}

# Function to update config for Docker Redis
function Update-ConfigForDocker {
    Write-Host "Updating config.yaml for Docker Redis..." -ForegroundColor Yellow
    
    if (Test-Path "config.yaml") {
        $config = Get-Content "config.yaml" -Raw
        
        # Update Redis URL to use localhost (since we're running SecAuto outside Docker)
        $config = $config -replace 'redis_url:\s*".*?"', 'redis_url: "redis://localhost:6379/0"'
        
        Set-Content -Path "config.yaml" -Value $config
        
        Write-Host "✓ Configuration updated for Docker Redis" -ForegroundColor Green
        Write-Host "  - Redis URL: redis://localhost:6379/0" -ForegroundColor Cyan
    } else {
        Write-Host "✗ config.yaml not found" -ForegroundColor Red
    }
}

# Main execution
if (-not (Test-DockerAvailable)) {
    Write-Host "✗ Docker is not available. Please install Docker Desktop first." -ForegroundColor Red
    Write-Host "Download from: https://www.docker.com/products/docker-desktop" -ForegroundColor Cyan
    exit 1
}

if ($StartRedis) {
    if (Start-RedisContainer) {
        Update-ConfigForDocker
        Start-Sleep -Seconds 3
        Show-RedisStatus
    }
}

if ($StopRedis) {
    Stop-RedisContainer
}

if ($RestartRedis) {
    Restart-RedisContainer
    Start-Sleep -Seconds 3
    Show-RedisStatus
}

if ($CheckStatus) {
    Show-RedisStatus
}

if ($ViewLogs) {
    Show-RedisLogs
}

# Show usage if no parameters provided
if (-not ($StartRedis -or $StopRedis -or $RestartRedis -or $CheckStatus -or $ViewLogs)) {
    Write-Host "Usage:" -ForegroundColor Yellow
    Write-Host "  .\start_redis_docker.ps1 -StartRedis" -ForegroundColor Cyan
    Write-Host "  .\start_redis_docker.ps1 -StopRedis" -ForegroundColor Cyan
    Write-Host "  .\start_redis_docker.ps1 -RestartRedis" -ForegroundColor Cyan
    Write-Host "  .\start_redis_docker.ps1 -CheckStatus" -ForegroundColor Cyan
    Write-Host "  .\start_redis_docker.ps1 -ViewLogs" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Commands:" -ForegroundColor Yellow
    Write-Host "  -StartRedis    Start Redis container" -ForegroundColor Cyan
    Write-Host "  -StopRedis     Stop Redis container" -ForegroundColor Cyan
    Write-Host "  -RestartRedis  Restart Redis container" -ForegroundColor Cyan
    Write-Host "  -CheckStatus   Show Redis container status" -ForegroundColor Cyan
    Write-Host "  -ViewLogs      Show Redis container logs" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Docker Commands:" -ForegroundColor Yellow
    Write-Host "  docker-compose up -d    # Start both Redis and SecAuto" -ForegroundColor Cyan
    Write-Host "  docker-compose down     # Stop all services" -ForegroundColor Cyan
    Write-Host "  docker-compose logs     # View all logs" -ForegroundColor Cyan
} 
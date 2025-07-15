# SecAuto Redis Setup Script
# This script helps set up Redis for use with SecAuto SOAR

param(
    [string]$RedisUrl = "redis://localhost:6379/0",
    [switch]$InstallRedis,
    [switch]$TestConnection,
    [switch]$UpdateConfig,
    [string]$ConfigPath = "config.yaml"
)

Write-Host "SecAuto Redis Setup Script" -ForegroundColor Green
Write-Host "=========================" -ForegroundColor Green

# Function to check if Redis is installed
function Test-RedisInstalled {
    try {
        $redisVersion = redis-cli --version 2>$null
        if ($redisVersion) {
            Write-Host "✓ Redis is installed: $redisVersion" -ForegroundColor Green
            return $true
        }
    }
    catch {
        Write-Host "✗ Redis is not installed" -ForegroundColor Red
        return $false
    }
    return $false
}

# Function to install Redis on Windows
function Install-RedisWindows {
    Write-Host "Installing Redis on Windows..." -ForegroundColor Yellow
    
    # Check if Chocolatey is available
    if (Get-Command choco -ErrorAction SilentlyContinue) {
        Write-Host "Using Chocolatey to install Redis..." -ForegroundColor Yellow
        choco install redis-64 -y
    }
    else {
        Write-Host "Chocolatey not found. Please install Redis manually:" -ForegroundColor Yellow
        Write-Host "1. Download from: https://github.com/microsoftarchive/redis/releases" -ForegroundColor Cyan
        Write-Host "2. Extract and run redis-server.exe" -ForegroundColor Cyan
        Write-Host "3. Or install Chocolatey first: https://chocolatey.org/install" -ForegroundColor Cyan
        return $false
    }
    
    return $true
}

# Function to start Redis server
function Start-RedisServer {
    Write-Host "Starting Redis server..." -ForegroundColor Yellow
    
    # Check if Redis is already running
    try {
        $response = redis-cli ping 2>$null
        if ($response -eq "PONG") {
            Write-Host "✓ Redis server is already running" -ForegroundColor Green
            return $true
        }
    }
    catch {
        # Redis not running, start it
    }
    
    # Start Redis server in background
    try {
        Start-Process redis-server -WindowStyle Hidden
        Start-Sleep -Seconds 3
        
        # Test connection
        $response = redis-cli ping 2>$null
        if ($response -eq "PONG") {
            Write-Host "✓ Redis server started successfully" -ForegroundColor Green
            return $true
        }
        else {
            Write-Host "✗ Failed to start Redis server" -ForegroundColor Red
            return $false
        }
    }
    catch {
        Write-Host "✗ Failed to start Redis server: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

# Function to test Redis connection
function Test-RedisConnection {
    param([string]$Url)
    
    Write-Host "Testing Redis connection..." -ForegroundColor Yellow
    
    try {
        # Extract host and port from URL
        if ($Url -match "redis://([^:]+):(\d+)/(\d+)") {
            $redisHost = $matches[1]
            $port = $matches[2]
            $db = $matches[3]
            
            Write-Host "Connecting to Redis at $redisHost`:$port (database $db)..." -ForegroundColor Cyan
            
            # Test basic connection
            $response = redis-cli -h $redisHost -p $port ping 2>$null
            if ($response -eq "PONG") {
                Write-Host "✓ Redis connection successful" -ForegroundColor Green
                
                # Test database selection
                $response = redis-cli -h $redisHost -p $port -n $db ping 2>$null
                if ($response -eq "PONG") {
                    Write-Host "✓ Database $db accessible" -ForegroundColor Green
                    return $true
                }
                else {
                    Write-Host "✗ Database $db not accessible" -ForegroundColor Red
                    return $false
                }
            }
            else {
                Write-Host "✗ Redis connection failed" -ForegroundColor Red
                return $false
            }
        }
        else {
            Write-Host "✗ Invalid Redis URL format: $Url" -ForegroundColor Red
            Write-Host "Expected format: redis://host:port/db" -ForegroundColor Cyan
            return $false
        }
    }
    catch {
        Write-Host "✗ Redis connection test failed: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

# Function to update config file
function Update-ConfigFile {
    param([string]$ConfigPath, [string]$RedisUrl)
    
    Write-Host "Updating configuration file..." -ForegroundColor Yellow
    
    if (-not (Test-Path $ConfigPath)) {
        Write-Host "✗ Configuration file not found: $ConfigPath" -ForegroundColor Red
        return $false
    }
    
    try {
        # Read current config
        $config = Get-Content $ConfigPath -Raw
        
        # Update database type and Redis URL
        $config = $config -replace 'type:\s*"sqlite"', 'type: "redis"'
        $config = $config -replace 'redis_url:\s*".*?"', "redis_url: `"$RedisUrl`""
        
        # Write updated config
        Set-Content -Path $ConfigPath -Value $config
        
        Write-Host "✓ Configuration updated successfully" -ForegroundColor Green
        Write-Host "  - Database type: redis" -ForegroundColor Cyan
        Write-Host "  - Redis URL: $RedisUrl" -ForegroundColor Cyan
        
        return $true
    }
    catch {
        Write-Host "✗ Failed to update configuration: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

# Function to show Redis info
function Show-RedisInfo {
    Write-Host "Redis Server Information:" -ForegroundColor Yellow
    
    try {
        # Get Redis info
        $info = redis-cli info 2>$null
        
        # Parse and display key information
        $lines = $info -split "`n"
        $serverInfo = @{}
        
        foreach ($line in $lines) {
            if ($line -match "^(.+):(.+)$") {
                $key = $matches[1]
                $value = $matches[2]
                $serverInfo[$key] = $value
            }
        }
        
        # Display key metrics
        if ($serverInfo.ContainsKey("redis_version")) {
            Write-Host "  Version: $($serverInfo['redis_version'])" -ForegroundColor Cyan
        }
        if ($serverInfo.ContainsKey("used_memory_human")) {
            Write-Host "  Memory Usage: $($serverInfo['used_memory_human'])" -ForegroundColor Cyan
        }
        if ($serverInfo.ContainsKey("connected_clients")) {
            Write-Host "  Connected Clients: $($serverInfo['connected_clients'])" -ForegroundColor Cyan
        }
        if ($serverInfo.ContainsKey("total_commands_processed")) {
            Write-Host "  Total Commands: $($serverInfo['total_commands_processed'])" -ForegroundColor Cyan
        }
        
        return $true
    }
    catch {
        Write-Host "✗ Failed to get Redis info: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

# Main execution
if ($InstallRedis) {
    if (-not (Test-RedisInstalled)) {
        if (Install-RedisWindows) {
            Write-Host "✓ Redis installed successfully" -ForegroundColor Green
        }
        else {
            Write-Host "✗ Redis installation failed" -ForegroundColor Red
            exit 1
        }
    }
}

if ($TestConnection -or $UpdateConfig) {
    if (-not (Test-RedisInstalled)) {
        Write-Host "✗ Redis is not installed. Use -InstallRedis to install it." -ForegroundColor Red
        exit 1
    }
    
    if (-not (Start-RedisServer)) {
        Write-Host "✗ Failed to start Redis server" -ForegroundColor Red
        exit 1
    }
    
    if (Test-RedisConnection -Url $RedisUrl) {
        Show-RedisInfo
        
        if ($UpdateConfig) {
            if (Update-ConfigFile -ConfigPath $ConfigPath -RedisUrl $RedisUrl) {
                Write-Host "`nSetup completed successfully!" -ForegroundColor Green
                Write-Host "You can now start SecAuto with Redis support." -ForegroundColor Cyan
            }
            else {
                Write-Host "✗ Failed to update configuration" -ForegroundColor Red
                exit 1
            }
        }
    }
    else {
        Write-Host "✗ Redis connection test failed" -ForegroundColor Red
        exit 1
    }
}

# Show usage if no parameters provided
if (-not ($InstallRedis -or $TestConnection -or $UpdateConfig)) {
    Write-Host "Usage:" -ForegroundColor Yellow
    Write-Host "  .\setup_redis.ps1 -InstallRedis" -ForegroundColor Cyan
    Write-Host "  .\setup_redis.ps1 -TestConnection" -ForegroundColor Cyan
    Write-Host "  .\setup_redis.ps1 -UpdateConfig" -ForegroundColor Cyan
    Write-Host "  .\setup_redis.ps1 -InstallRedis -TestConnection -UpdateConfig" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Parameters:" -ForegroundColor Yellow
    Write-Host "  -InstallRedis    Install Redis on Windows" -ForegroundColor Cyan
    Write-Host "  -TestConnection  Test Redis connection" -ForegroundColor Cyan
    Write-Host "  -UpdateConfig    Update config.yaml to use Redis" -ForegroundColor Cyan
    Write-Host "  -RedisUrl        Redis connection URL (default: redis://localhost:6379/0)" -ForegroundColor Cyan
    Write-Host "  -ConfigPath      Path to config file (default: config.yaml)" -ForegroundColor Cyan
} 
# Log Rotation Implementation

## Overview

The SecAuto SOAR platform now includes configurable log rotation using the `lumberjack` library. This feature ensures that log files don't grow indefinitely and provides automatic management of log file sizes, backups, and cleanup.

## Features

### ✅ Implemented Features

1. **Configurable Log Rotation**
   - Maximum file size (configurable in MB)
   - Maximum number of backup files
   - Maximum age for log files (in days)
   - Compression of rotated logs

2. **Structured JSON Logging**
   - All logs are in JSON format for easy parsing
   - Includes timestamps, log levels, and contextual information
   - Supports multiple logging destinations (stdout, file, both)

3. **Flexible Configuration**
   - Rotation settings configurable via `config.yaml`
   - Environment variable overrides supported
   - Sensible defaults for standalone mode

## Configuration

### config.yaml Settings

```yaml
logging:
  level: INFO
  destination: file   # options: stdout, file, both
  file: logs/secauto.log
  rotation:
    max_size_mb: 10      # Rotate when file exceeds 10 MB
    max_backups: 5       # Keep up to 5 old log files
    max_age_days: 7      # Delete logs older than 7 days
    compress: true       # Compress rotated logs
```

### Configuration Options

| Setting | Default | Description |
|---------|---------|-------------|
| `max_size_mb` | 10 | Maximum file size in MB before rotation |
| `max_backups` | 5 | Maximum number of backup files to keep |
| `max_age_days` | 7 | Maximum age of log files in days |
| `compress` | true | Whether to compress rotated log files |

## Implementation Details

### Dependencies

- **lumberjack**: `gopkg.in/natefinch/lumberjack.v2` for log rotation
- **Standard library**: `net/http`, `encoding/json`, `time`, etc.

### Code Structure

1. **Configuration (`config.go`)**
   ```go
   type RotationConfig struct {
       MaxSizeMB  int  `yaml:"max_size_mb"`
       MaxBackups int  `yaml:"max_backups"`
       MaxAgeDays int  `yaml:"max_age_days"`
       Compress   bool `yaml:"compress"`
   }
   ```

2. **Logger (`main.go`)**
   ```go
   type StructuredLogger struct {
       level      LogLevel
       dest       string
       fileWriter io.Writer
       rotation   *RotationConfig
   }
   ```

3. **Lumberjack Integration**
   ```go
   fileWriter = &lumberjack.Logger{
       Filename:   filePath,
       MaxSize:    rotation.MaxSizeMB,
       MaxBackups: rotation.MaxBackups,
       MaxAge:     rotation.MaxAgeDays,
       Compress:   rotation.Compress,
   }
   ```

## Usage Examples

### Server Mode (with config.yaml)

```bash
# Start server with rotation enabled
./secauto.exe -port 8080 -workers 5
```

### Standalone Mode (with default rotation)

```bash
# Run standalone with default rotation settings
./secauto.exe -s -p playbooks/testplaybook.json
```

### Testing Log Rotation

```powershell
# Run the test script
.\test_log_rotation.ps1
```

## Log File Management

### File Naming Convention

- **Current log**: `secauto.log`
- **Rotated logs**: `secauto.log.1`, `secauto.log.2`, etc.
- **Compressed logs**: `secauto.log.1.gz`, `secauto.log.2.gz`, etc.

### Rotation Behavior

1. **Size-based rotation**: When `secauto.log` reaches `max_size_mb`, it's rotated
2. **Backup management**: Oldest backup is deleted when `max_backups` is exceeded
3. **Age-based cleanup**: Logs older than `max_age_days` are automatically deleted
4. **Compression**: Rotated logs are compressed if `compress: true`

## Monitoring and Maintenance

### Checking Log Files

```powershell
# List all log files
Get-ChildItem logs\secauto.log*

# Check current log size
Get-Item logs\secauto.log | Select-Object Name, Length, LastWriteTime

# Count log entries
(Get-Content logs\secauto.log).Count
```

### Log Analysis

```powershell
# Filter logs by level
Get-Content logs\secauto.log | Where-Object { $_ -match '"level":"ERROR"' }

# Find recent requests
Get-Content logs\secauto.log | Where-Object { $_ -match '"component":"http"' }
```

## Benefits

### ✅ Advantages

1. **Disk Space Management**: Prevents log files from consuming unlimited disk space
2. **Performance**: Smaller log files are faster to read and process
3. **Compliance**: Automatic cleanup helps meet data retention requirements
4. **Monitoring**: Structured JSON logs enable easy parsing and analysis
5. **Flexibility**: Configurable settings for different deployment scenarios

### 🔧 Technical Benefits

1. **Thread-safe**: Lumberjack provides thread-safe log rotation
2. **Non-blocking**: Log rotation doesn't block application execution
3. **Compression**: Reduces disk usage for historical logs
4. **Structured**: JSON format enables easy log analysis and monitoring

## Troubleshooting

### Common Issues

1. **Permission Errors**
   - Ensure the application has write permissions to the logs directory
   - Check that the logs directory exists and is writable

2. **Rotation Not Working**
   - Verify `destination: file` in config.yaml
   - Check that `max_size_mb` is set to a reasonable value
   - Ensure the application has sufficient disk space

3. **Log File Not Created**
   - Check that the logs directory path is correct
   - Verify the application has permission to create files
   - Ensure the server is running and generating logs

### Debug Commands

```powershell
# Check if server is running
Get-Process -Name "secauto" -ErrorAction SilentlyContinue

# Test health endpoint
Invoke-WebRequest -Uri "http://localhost:8080/health" -Method GET

# Monitor log file in real-time
Get-Content logs\secauto.log -Wait -Tail 10
```

## Future Enhancements

### Potential Improvements

1. **Remote Logging**: Send logs to external systems (ELK, Splunk, etc.)
2. **Log Levels**: More granular control over what gets logged
3. **Metrics**: Log rotation statistics and monitoring
4. **Backup**: Automatic backup of log files to external storage
5. **Alerting**: Notifications when log rotation occurs

## Conclusion

The log rotation implementation provides a robust, configurable logging solution that ensures the SecAuto platform can run continuously without disk space issues while maintaining comprehensive audit trails through structured JSON logging. 
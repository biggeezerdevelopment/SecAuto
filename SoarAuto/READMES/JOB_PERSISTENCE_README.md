# Job Persistence & Recovery

## Overview

The SecAuto SOAR platform now includes robust job persistence and recovery capabilities. This feature ensures that jobs survive server restarts, crashes, and provides automatic recovery mechanisms for production environments.

## Features

### ✅ Implemented Features

1. **Persistent Job Storage**
   - JSON-based file storage with atomic operations
   - Automatic data directory creation
   - Thread-safe operations with mutex protection

2. **Job Recovery**
   - Automatic recovery of jobs on server restart
   - Handling of jobs that were running during crashes
   - Webhook notifications for recovered jobs

3. **Background Maintenance**
   - Automatic cleanup of old jobs (configurable retention)
   - Periodic backups of job data
   - Background tasks with configurable intervals

4. **Data Integrity**
   - Atomic write operations using temporary files
   - Automatic backup creation
   - Error handling and logging

## Configuration

### config.yaml Settings

```yaml
# Data Directory for Job Persistence
data_directory: "data"

# Job Persistence Configuration
jobs:
  retention_days: 30     # Keep jobs for 30 days
  cleanup_interval: 24   # Run cleanup every 24 hours
  backup_enabled: true   # Enable automatic backups
  backup_interval: 168   # Create backup every 168 hours (7 days)
```

### Configuration Options

| Setting | Default | Description |
|---------|---------|-------------|
| `data_directory` | "data" | Directory for storing job data |
| `retention_days` | 30 | Days to keep job history |
| `cleanup_interval` | 24 | Hours between cleanup runs |
| `backup_enabled` | true | Enable automatic backups |
| `backup_interval` | 168 | Hours between backups |

## Implementation Details

### File Structure

```
data/
├── jobs.json              # Current job data
├── jobs.json.backup.*     # Automatic backups
└── jobs.json.tmp          # Temporary file for atomic writes
```

### Job Data Format

```json
{
  "job-id-1": {
    "id": "job-id-1",
    "status": "completed",
    "playbook": [...],
    "context": {...},
    "results": [...],
    "error": "",
    "created_at": "2025-07-09T15:30:00Z",
    "started_at": "2025-07-09T15:30:05Z",
    "completed_at": "2025-07-09T15:30:45Z"
  }
}
```

## Usage Examples

### Server Startup with Recovery

```bash
# Start server - jobs will be automatically recovered
./secauto.exe -port 8080 -workers 5
```

### Manual Job Management

```powershell
# Check job data
Get-Content "data\jobs.json"

# Create manual backup
Copy-Item "data\jobs.json" "data\jobs.json.manual.backup"
```

### Testing Recovery

```powershell
# Run the test script
.\test_job_persistence.ps1
```

## Recovery Process

### Automatic Recovery

1. **Server Startup**: Jobs are loaded from `data/jobs.json`
2. **Running Job Detection**: Jobs with status "running" are identified
3. **Status Update**: Running jobs are marked as "failed" with recovery message
4. **Webhook Notification**: Recovery events are sent to configured webhooks
5. **Logging**: Recovery process is logged with structured logging

### Recovery Scenarios

| Scenario | Action | Result |
|----------|--------|--------|
| Normal shutdown | Jobs saved to disk | Jobs available on restart |
| Server crash | Jobs in memory lost | Running jobs marked as failed |
| File corruption | Backup restoration | Jobs recovered from backup |
| Disk failure | No recovery possible | Fresh start with empty store |

## Background Tasks

### Automatic Cleanup

- **Frequency**: Every 24 hours (configurable)
- **Action**: Removes jobs older than retention period
- **Logging**: Structured logs with cleanup statistics

### Automatic Backups

- **Frequency**: Every 7 days (configurable)
- **Format**: `jobs.json.backup.YYYYMMDD-HHMMSS`
- **Compression**: Not implemented (future enhancement)

## API Integration

### Existing Endpoints Enhanced

All existing job-related endpoints now use persistent storage:

- `GET /jobs` - Lists jobs from persistent storage
- `GET /job/{id}` - Retrieves job from persistent storage
- `POST /playbook/async` - Saves job to persistent storage
- `DELETE /job/{id}` - Removes job from persistent storage

### New Features

- **Job History**: All jobs are preserved with full audit trail
- **Statistics**: Accurate job statistics from persistent data
- **Recovery**: Automatic handling of interrupted jobs

## Monitoring and Maintenance

### Health Checks

```powershell
# Check data directory
Get-ChildItem "data" -ErrorAction SilentlyContinue

# Check job file size
Get-Item "data\jobs.json" | Select-Object Name, Length, LastWriteTime

# Monitor job count
(Get-Content "data\jobs.json" | ConvertFrom-Json).Count
```

### Backup Management

```powershell
# List backups
Get-ChildItem "data\jobs.json.backup.*" | Sort-Object LastWriteTime

# Restore from backup
Copy-Item "data\jobs.json.backup.20250709-143000" "data\jobs.json"
```

### Cleanup Verification

```powershell
# Check for old jobs
$jobs = Get-Content "data\jobs.json" | ConvertFrom-Json
$cutoff = (Get-Date).AddDays(-30)
$oldJobs = $jobs.PSObject.Properties | Where-Object {
    [DateTime]::Parse($_.Value.created_at) -lt $cutoff
}
Write-Host "Old jobs: $($oldJobs.Count)"
```

## Benefits

### ✅ Advantages

1. **Data Persistence**: Jobs survive server restarts and crashes
2. **Audit Trail**: Complete job history for compliance
3. **Recovery**: Automatic handling of interrupted operations
4. **Reliability**: Atomic operations prevent data corruption
5. **Maintenance**: Automatic cleanup prevents disk space issues
6. **Monitoring**: Enhanced visibility into job execution

### 🔧 Technical Benefits

1. **Thread Safety**: Mutex-protected operations
2. **Atomic Writes**: Temporary file + rename pattern
3. **Error Handling**: Comprehensive error logging
4. **Background Tasks**: Non-blocking maintenance operations
5. **Configurable**: Flexible retention and backup settings

## Troubleshooting

### Common Issues

1. **Permission Errors**
   - Ensure write permissions to data directory
   - Check disk space availability

2. **File Corruption**
   - Restore from backup file
   - Check for concurrent access issues

3. **Recovery Failures**
   - Verify job file format
   - Check for JSON parsing errors

### Debug Commands

```powershell
# Check server logs
Get-Content "logs\secauto.log" | Select-String "job_store"

# Verify job data integrity
$jobs = Get-Content "data\jobs.json" | ConvertFrom-Json
$jobs.PSObject.Properties | ForEach-Object {
    Write-Host "Job: $($_.Name) - Status: $($_.Value.status)"
}

# Monitor background tasks
Get-Content "logs\secauto.log" | Select-String "cleanup|backup"
```

## Future Enhancements

### Potential Improvements

1. **Database Storage**: SQLite/PostgreSQL for better performance
2. **Compression**: Compress backup files to save space
3. **Encryption**: Encrypt job data for security
4. **Replication**: Multi-node job storage
5. **Metrics**: Job persistence statistics
6. **API**: Backup/restore management endpoints

## Conclusion

The job persistence implementation provides a robust foundation for production SOAR operations, ensuring that critical automation jobs are never lost due to server issues while maintaining comprehensive audit trails and automatic maintenance. 
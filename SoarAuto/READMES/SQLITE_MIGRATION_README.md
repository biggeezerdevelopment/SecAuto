# SQLite Job Persistence Migration

This document describes the migration from file-based job persistence to SQLite database storage in the SecAuto SOAR rules engine.

## Overview

The job persistence system has been converted from storing jobs in a JSON file (`jobs.json`) to using SQLite database (`jobs.db`) for improved performance, reliability, and scalability.

## Changes Made

### 1. Database Schema

The SQLite database uses the following schema:

```sql
CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    playbook TEXT NOT NULL,
    context TEXT NOT NULL,
    results TEXT,
    error TEXT,
    created_at DATETIME NOT NULL,
    started_at DATETIME,
    completed_at DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**Indexes for Performance:**
- `idx_jobs_status` - For filtering by job status
- `idx_jobs_created_at` - For sorting by creation time
- `idx_jobs_updated_at` - For tracking modifications

### 2. Data Storage

- **Complex Data**: Playbook, context, and results are stored as JSON strings
- **Timestamps**: All time fields use SQLite's DATETIME type
- **Nullable Fields**: `started_at`, `completed_at`, `results`, and `error` can be NULL

### 3. Key Features

#### Performance Improvements
- **Indexed Queries**: Fast lookups by job ID and status
- **Efficient Filtering**: SQL-based filtering instead of in-memory iteration
- **Atomic Operations**: SQLite provides ACID compliance

#### Reliability Enhancements
- **ACID Compliance**: Database transactions ensure data integrity
- **Concurrent Access**: Proper locking with read/write mutexes
- **Error Recovery**: Graceful handling of database errors

#### Scalability
- **Large Datasets**: SQLite can handle millions of jobs efficiently
- **Memory Efficiency**: Only loads jobs when needed
- **Backup Support**: Database file can be easily backed up

### 4. API Compatibility

The migration maintains full API compatibility. All existing methods work unchanged:

```go
// Save a job
store.SaveJob(job)

// Load a job by ID
job, exists := store.LoadJob(jobID)

// List jobs with filtering
jobs := store.ListJobs(status, limit)

// Update job status
store.UpdateJobStatus(jobID, status)

// Update job results
store.UpdateJobResults(jobID, results, errorMsg)

// Delete a job
store.DeleteJob(jobID)

// Get statistics
stats := store.GetStats()

// Cleanup old jobs
store.CleanupOldJobs(maxAge)
```

### 5. Database Operations

#### Initialization
```go
store, err := NewJobStore(dataDir)
if err != nil {
    return err
}
defer store.Close()
```

#### Job Storage
- Jobs are stored with `INSERT OR REPLACE` for upsert behavior
- Complex data (playbook, context, results) is JSON-encoded
- Timestamps are automatically managed

#### Querying
- **By ID**: Direct primary key lookup
- **By Status**: Indexed status filtering
- **By Date**: Range queries on created_at
- **Statistics**: SQL aggregation functions

### 6. Migration Benefits

#### Performance
- **Faster Queries**: Indexed lookups vs. file parsing
- **Reduced Memory**: No need to load all jobs into memory
- **Concurrent Access**: Multiple readers, single writer

#### Reliability
- **Data Integrity**: ACID transactions prevent corruption
- **Crash Recovery**: SQLite handles unexpected shutdowns
- **Error Handling**: Graceful degradation on database errors

#### Maintenance
- **Backup**: Simple file copy for database backup
- **Monitoring**: SQL queries for performance monitoring
- **Debugging**: Standard SQL tools for data inspection

### 7. Configuration

The database file is stored in the configured data directory:
```
config.yaml:
data_directory: "./data"
```

Database file location:
```
./data/jobs.db
```

### 8. Backup and Recovery

#### Automatic Backup
- Database backups are created with timestamps
- Backup files: `jobs.db.backup.YYYYMMDD-HHMMSS`
- Backups run every 7 days automatically

#### Manual Backup
```go
err := store.BackupJobs()
```

#### Recovery
- Database automatically recovers from crashes
- Running jobs are marked as failed on restart
- Webhook notifications sent for recovered jobs

### 9. Testing

A test function is provided to verify SQLite functionality:

```go
testSQLiteJobPersistence()
```

The test covers:
- Job creation and storage
- Job loading and retrieval
- Status updates
- Results updates
- Job listing and filtering
- Statistics generation
- Job deletion

### 10. Dependencies

Added SQLite driver dependency:
```go
require (
    github.com/mattn/go-sqlite3 v1.14.22
)
```

### 11. Graceful Shutdown

The application now supports graceful shutdown:
- Signal handling for SIGINT and SIGTERM
- Database connection cleanup
- Background task termination
- Proper resource cleanup

### 12. Monitoring and Logging

Enhanced logging for database operations:
- Connection initialization
- Query performance
- Error conditions
- Backup operations
- Cleanup activities

## Migration Notes

### From File to Database
- Existing `jobs.json` files are not automatically migrated
- New jobs will be stored in SQLite database
- Old file-based jobs remain accessible until manually migrated

### Performance Expectations
- **Initial Load**: Slightly slower due to database initialization
- **Query Performance**: Significantly faster for large datasets
- **Memory Usage**: Reduced memory footprint
- **Concurrent Access**: Improved with proper locking

### Backup Strategy
- Database file can be copied for backup
- Automatic timestamped backups
- Backup files can be restored by replacing database file

## Future Enhancements

### Potential Improvements
1. **Connection Pooling**: For high-concurrency scenarios
2. **Database Migration**: Tool to migrate from JSON files
3. **Performance Monitoring**: Query performance metrics
4. **Compression**: Database compression for large datasets
5. **Replication**: Multi-database support for high availability

### Monitoring
- Database size monitoring
- Query performance tracking
- Connection pool metrics
- Backup success/failure tracking

## Troubleshooting

### Common Issues

1. **Database Locked**
   - Ensure only one instance is running
   - Check for proper cleanup on shutdown

2. **Permission Errors**
   - Verify write permissions to data directory
   - Check file ownership

3. **Performance Issues**
   - Monitor database size
   - Check index usage
   - Review query patterns

4. **Corruption Recovery**
   - Restore from backup file
   - Use SQLite integrity check
   - Rebuild database if necessary

### Debug Commands

```bash
# Check database integrity
sqlite3 data/jobs.db "PRAGMA integrity_check;"

# View database schema
sqlite3 data/jobs.db ".schema"

# Check table sizes
sqlite3 data/jobs.db "SELECT COUNT(*) FROM jobs;"

# Analyze performance
sqlite3 data/jobs.db "ANALYZE;"
```

## Conclusion

The migration to SQLite provides significant improvements in performance, reliability, and scalability while maintaining full API compatibility. The database approach is more suitable for production environments and provides a solid foundation for future enhancements. 
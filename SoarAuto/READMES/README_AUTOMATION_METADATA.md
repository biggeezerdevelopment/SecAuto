# Automation Metadata Management System

This document describes the automation metadata management system that allows you to store and retrieve metadata for automation scripts, including virtual environment paths and return value specifications.

## Overview

The automation metadata system provides:
- **Persistent Storage**: Metadata is saved to disk and loaded at server startup
- **Redis Integration**: Metadata is cached in Redis for fast runtime access
- **Automatic Management**: Metadata is automatically loaded/saved during server lifecycle
- **API Endpoints**: RESTful API for managing automation metadata

## Metadata Structure

Each automation metadata entry follows this JSON structure:

```json
{
  "name": "automation_name",
  "venv": "path/to/virtual/environment",
  "description": "Human-readable description of what the automation does",
  "return": {
    "field1": "type_description",
    "field2": "type_description"
  }
}
```

### Fields

- **name**: The name of the automation script (without extension)
- **venv**: Path to the Python virtual environment for this automation
- **description**: Human-readable description of what the automation does
- **return**: Object describing the expected return values and their types

## API Endpoints

### List All Metadata
```
GET /automation/metadata
```

Returns all automation metadata.

**Response:**
```json
{
  "success": true,
  "message": "Automation metadata retrieved successfully",
  "metadata": [
    {
      "name": "addclient",
      "venv": "Venv",
      "return": {
        "clientname": "string"
      }
    }
  ],
  "timestamp": "2025-01-27T10:00:00Z"
}
```

### Create/Update Metadata
```
POST /automation/metadata
```

Creates new automation metadata.

**Request Body:**
```json
{
  "name": "new_automation",
  "venv": "/path/to/venv",
  "description": "Description of what this automation does",
  "return": {
    "result": "string"
  }
}
```

### Get Specific Metadata
```
GET /automation/metadata/{name}
```

Returns metadata for a specific automation.

### Update Specific Metadata
```
PUT /automation/metadata/{name}
```

Updates metadata for a specific automation.

### Delete Metadata
```
DELETE /automation/metadata/{name}
```

Deletes metadata for a specific automation.

## Uploading Automations with Metadata

When uploading an automation via the `/automation` endpoint, you can include metadata in the form data:

```bash
curl -X POST http://localhost:8000/automation \
  -F "automation=@script.py" \
  -F 'metadata={"name":"script","venv":"Venv","description":"Description of the script","return":{"result":"string"}}'
```

## File Storage

Metadata is stored in `data/automation_metadata.json` and follows this structure:

```json
[
  {
    "name": "addclient",
    "venv": "Venv",
    "description": "Adds a new client to the system and returns client information",
    "return": {
      "clientname": "string"
    }
  },
  {
    "name": "qualysauto",
    "venv": "Venv",
    "description": "Automates Qualys vulnerability scanning operations",
    "return": {
      "scan_results": "object"
    }
  }
]
```

## Redis Storage

Metadata is cached in Redis with the following keys:
- `automation_metadata`: Complete list of all metadata
- `automation_metadata:{name}`: Individual automation metadata

## Server Lifecycle Integration

### Startup
1. Server loads metadata from `data/automation_metadata.json`
2. Metadata is loaded into Redis for fast access
3. If no metadata file exists, an empty metadata set is created

### Shutdown
1. All metadata is automatically saved to disk
2. Redis connections are properly closed

## Usage Examples

### Python Client Example

```python
import requests
import json

# Upload automation with metadata
metadata = {
    "name": "my_script",
    "venv": "/path/to/venv",
    "return": {
        "status": "string",
        "data": "object"
    }
}

files = {'automation': open('script.py', 'rb')}
data = {'metadata': json.dumps(metadata)}

response = requests.post('http://localhost:8000/automation', 
                        files=files, data=data)
print(response.json())
```

### Go Client Example

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
)

type AutomationMetadata struct {
    Name   string                 `json:"name"`
    Venv   string                 `json:"venv"`
    Return map[string]interface{} `json:"return"`
}

func uploadAutomationWithMetadata(scriptPath, serverURL string, metadata AutomationMetadata) error {
    var buf bytes.Buffer
    writer := multipart.NewWriter(&buf)
    
    // Add file
    file, err := writer.CreateFormFile("automation", "script.py")
    if err != nil {
        return err
    }
    
    // Read and write file content
    scriptData, err := io.ReadFile(scriptPath)
    if err != nil {
        return err
    }
    file.Write(scriptData)
    
    // Add metadata
    metadataBytes, _ := json.Marshal(metadata)
    writer.WriteField("metadata", string(metadataBytes))
    writer.Close()
    
    // Send request
    resp, err := http.Post(serverURL+"/automation", writer.FormDataContentType(), &buf)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}
```

## Error Handling

The system gracefully handles various error conditions:
- Missing metadata files (creates empty set)
- Invalid JSON (returns appropriate HTTP errors)
- Redis connection failures (logs errors but continues operation)
- File I/O errors (logs errors and continues)

## Configuration

The metadata system uses the following configuration:
- **Metadata File**: `data/automation_metadata.json`
- **Redis**: Uses the same Redis connection as other system components
- **File Permissions**: 0644 for metadata file, 0755 for directories

## Security Considerations

- Metadata endpoints require valid API key authentication
- Input validation is performed on all metadata
- File paths are validated to prevent directory traversal
- Redis keys are properly namespaced

## Monitoring and Logging

All metadata operations are logged with structured logging:
- Component: `automation_metadata_manager`
- Operation details (add, update, delete, load, save)
- Error conditions and resolutions
- Performance metrics for Redis operations

## Troubleshooting

### Common Issues

1. **Metadata not loading**: Check file permissions and JSON syntax
2. **Redis connection errors**: Verify Redis is running and accessible
3. **File not found errors**: Ensure the data directory exists and is writable

### Debug Commands

Check Redis metadata:
```bash
redis-cli get automation_metadata
redis-cli keys automation_metadata:*
```

Check metadata file:
```bash
cat data/automation_metadata.json | jq .
```

## Future Enhancements

Potential improvements for the metadata system:
- Metadata validation schemas
- Version control for metadata changes
- Bulk metadata operations
- Metadata search and filtering
- Integration with CI/CD pipelines
- Automated metadata extraction from scripts

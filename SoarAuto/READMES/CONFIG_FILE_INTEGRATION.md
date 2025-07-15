# Config File Integration System

## Overview

The SecAuto server now automatically writes a configuration file that Python integrations can read to get the necessary connection details. This solves the problem of environment variables not being available to external processes.

## How It Works

### 1. Server-Side Configuration

When the SecAuto server starts up, it:

1. Reads the `config.yaml` file
2. Extracts the first valid API key from `security.api_keys`
3. Gets the server host and port from `server.host` and `server.port`
4. Sets environment variables for child processes
5. **NEW**: Writes a JSON config file to `data/integration_config.json`

### 2. Config File Format

The config file contains:

```json
{
  "secauto_url": "http://localhost:8080",
  "secauto_api_key": "secauto-api-key-2024-07-14",
  "server_host": "localhost",
  "server_port": 8080,
  "timestamp": "2024-07-14T10:30:00Z"
}
```

### 3. Python Integration Loading

Python integrations now try to load configuration in this order:

1. **Environment variables** (for backward compatibility)
   - `SECAUTO_URL`
   - `SECAUTO_API_KEY`

2. **Config file** (new approach)
   - `data/integration_config.json`
   - `SoarAuto/data/integration_config.json`
   - `../SoarAuto/data/integration_config.json`

3. **Fallback defaults**
   - URL: `http://localhost:8080`
   - API key: `None`

## Benefits

### ✅ Solves Environment Variable Problem

Environment variables set by a process are only available to that process and its child processes. External Python scripts running separately from the server couldn't access them.

### ✅ Automatic Configuration

No manual setup required. The server automatically writes the config file on startup.

### ✅ Multiple Fallback Paths

The integration tries multiple possible file paths to find the config file.

### ✅ Backward Compatibility

Still supports environment variables for existing setups.

## Configuration Requirements

### Server Configuration

The server needs these settings in `config.yaml`:

```yaml
server:
  host: "localhost"  # Changed from "0.0.0.0"
  port: 8080

security:
  api_keys:
    - "secauto-api-key-2024-07-14"  # Must be a real key, not placeholder
    - "another-api-key-if-needed"
```

### Integration Configuration

Python integrations automatically detect and use the config file. No changes needed to integration code.

## Testing

Run the test script to verify everything works:

```bash
python test_config_file.py
```

This will test:
- ✅ Config file exists and is readable
- ✅ VirusTotal integration can load configuration
- ✅ Server connection works with config file

## Troubleshooting

### Config File Not Found

**Problem**: `data/integration_config.json` doesn't exist

**Solutions**:
1. Restart the SecAuto server to generate the config file
2. Check that `config.yaml` has valid API keys (not placeholders)
3. Verify the server has write permissions to the `data/` directory

### API Key Not Set

**Problem**: Config file exists but API key is missing

**Solutions**:
1. Check `config.yaml` has real API keys (not `"your-secauto-api-key-here"`)
2. Restart the server to reload configuration
3. Check server logs for configuration errors

### Integration Still Can't Connect

**Problem**: Integration loads config but can't connect to server

**Solutions**:
1. Verify server is running on the configured host/port
2. Check firewall settings
3. Test server health endpoint manually: `curl http://localhost:8080/health`

## File Locations

The config file is written to these locations (in order of preference):

1. `data/integration_config.json` (relative to server working directory)
2. `SoarAuto/data/integration_config.json` (relative to project root)
3. `../SoarAuto/data/integration_config.json` (relative to integrations directory)

## Security Considerations

- The config file contains sensitive API keys
- File permissions are set to 644 (readable by owner and group)
- The file is written to a `data/` directory that should be protected
- Consider adding the config file to `.gitignore` if using version control

## Migration from Environment Variables

If you were previously using environment variables, the integration will automatically detect and use the config file instead. No changes to your integration code are required.

The integration will log which configuration source it's using:

```
INFO:integrations.virustotal_integration:Using SecAuto config from data/integration_config.json
```

## Example Usage

```python
from integrations.virustotal_integration import VirusTotalIntegration

# The integration automatically loads config from the file
vt = VirusTotalIntegration()

# It will have the correct URL and API key
print(f"URL: {vt.secauto_url}")
print(f"API Key: {vt.secauto_api_key[:10]}...")

# Test the connection
result = vt.scan_url("https://example.com")
print(f"Scan successful: {result.success}")
``` 
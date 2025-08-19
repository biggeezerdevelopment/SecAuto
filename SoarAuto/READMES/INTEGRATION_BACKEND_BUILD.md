# Integration Backend Build System

## Overview

The Integration Backend Build System automatically builds and manages Python dependencies for each integration when its configuration is uploaded. This system uses a layered site-packages approach to isolate dependencies while maintaining a single Python virtual environment.

## Architecture

```
┌─────────────────────────────────────────┐
│         Integration Upload API           │
│              (Go Server)                  │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│      Integration Upload Hook              │
│   (scripts/integration_upload_hook.py)    │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│     Integration Backend Builder           │
│  (scripts/build_integration_backend.py)   │
├─────────────────────────────────────────┤
│ • Parse config for dependencies          │
│ • Create integration site-packages       │
│ • Install packages with pip              │
│ • Register with Python path (.pth)       │
│ • Update sitecustomize.py                │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│        Integration Loader                 │
│    (server/integration_loader.py)         │
├─────────────────────────────────────────┤
│ • Load integration modules               │
│ • Manage sys.path for dependencies       │
│ • Provide function access to automations │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│           Automations                     │
│     (use_integration() function)          │
└─────────────────────────────────────────┘
```

## Integration Configuration Schema

Each integration must provide a configuration file with dependency specifications:

```json
{
  "name": "integration_name",
  "version": "1.0.0",
  "dependencies": {
    "packages": [
      "package==version",
      "another-package>=1.0"
    ],
    "optional": ["pandas>=1.5.0"]
  },
  "backend": {
    "type": "python",
    "entry_point": "integration.py",
    "requires_build": true
  },
  "functions": {
    "function_name": {
      "description": "What this function does",
      "parameters": {}
    }
  }
}
```

## Directory Structure

```
SecAuto/
├── integrations/
│   ├── .site-packages/           # Integration-specific packages
│   │   ├── qualys_integration/   # Qualys dependencies
│   │   │   ├── qualysapi/
│   │   │   ├── lxml/
│   │   │   └── ...
│   │   └── tenable_integration/  # Tenable dependencies
│   │       ├── pytenable/
│   │       └── ...
│   ├── .build_status.json        # Build status tracking
│   └── [integration_configs]     # Integration configurations
├── Venv/
│   └── Lib/site-packages/
│       ├── integration_qualys.pth    # Path registration
│       ├── integration_tenable.pth   # Path registration
│       └── sitecustomize.py          # Dynamic loader
└── scripts/
    ├── build_integration_backend.py  # Builder script
    └── integration_upload_hook.py    # Upload handler
```

## Usage

### 1. Upload Integration Configuration

When an integration configuration is uploaded via the API:

```bash
curl -X POST http://localhost:9090/integrations/upload \
  -H "X-API-Key: your-api-key" \
  -F "config=@qualys_integration_config.json"
```

### 2. Automatic Backend Build

The system automatically:
1. Parses the configuration for dependencies
2. Creates an integration-specific site-packages directory
3. Installs required packages into that directory
4. Registers the directory with the Python path
5. Updates the build status

### 3. Use in Automations

Automations can now use the integration functions:

```python
from server.SoarBaseAPI import use_integration

# Dependencies are automatically loaded
result = use_integration(
    'qualys_integration',
    'scan_hosts',
    hosts=['10.0.0.1'],
    scan_type='vulnerability'
)
```

## Key Components

### Integration Backend Builder
- **Location**: `scripts/build_integration_backend.py`
- **Purpose**: Builds integration-specific environments
- **Features**:
  - Dependency installation to isolated directories
  - Python path registration via .pth files
  - sitecustomize.py updates for dynamic loading
  - Build status tracking

### Integration Loader
- **Location**: `server/integration_loader.py`
- **Purpose**: Runtime loading of integrations
- **Features**:
  - Dynamic module loading
  - Dependency path management
  - Function discovery and invocation
  - Process-isolated environment variables

### SoarBaseAPI Extensions
- **Location**: `server/SoarBaseAPI.py`
- **New Functions**:
  - `use_integration()`: Call integration functions
  - `list_integration_functions()`: Discover available functions
  - `check_integration_available()`: Check if integration is built

## Dependency Isolation

### How It Works

1. **Separate Site-Packages**: Each integration gets its own site-packages directory
2. **Path Registration**: .pth files in main venv point to integration directories
3. **Dynamic Loading**: sitecustomize.py adds paths based on context
4. **Process Isolation**: Environment variables are process-specific, preventing conflicts

### Benefits

- **No Conflicts**: Integrations can use different versions of the same package
- **Clean Uninstall**: Remove integration directory to clean all dependencies
- **Lazy Loading**: Dependencies only loaded when integration is used
- **Single Venv**: No need for multiple virtual environments

## Testing

Run the test script to verify the system:

```bash
cd /Volumes/My\ Shared\ Files/Home/Downloads/SecAuto
chmod +x scripts/test_integration_build.sh
./scripts/test_integration_build.sh
```

## Manual Commands

### Build Integration
```bash
python scripts/build_integration_backend.py build \
  --config integrations/example_integration_config.json
```

### Check Status
```bash
python scripts/build_integration_backend.py status \
  --name qualys_integration
```

### Clean Integration
```bash
python scripts/build_integration_backend.py clean \
  --name qualys_integration
```

## Go Server Integration

The Go server should call the build script when an integration is uploaded:

```go
// In integration upload handler
cmd := exec.Command(
    pythonPath,
    "scripts/integration_upload_hook.py",
    configPath,
    "--async",
)
output, err := cmd.Output()
```

## Troubleshooting

### Dependencies Not Found
- Check `.build_status.json` for build status
- Verify site-packages directory exists
- Check .pth files in main venv

### Import Errors
- Ensure integration is built (`check_integration_available()`)
- Check Python path includes integration site-packages
- Verify sitecustomize.py is updated

### Build Failures
- Check script logs for pip errors
- Verify network access for package downloads
- Ensure sufficient disk space

## Security Considerations

1. **Package Validation**: Verify packages before installation
2. **Sandboxing**: Run builds in restricted environment
3. **Resource Limits**: Set timeouts and memory limits
4. **Audit Logging**: Log all package installations
5. **Cleanup**: Remove unused integration packages

## Future Enhancements

1. **Dependency Caching**: Cache downloaded packages
2. **Version Management**: Support multiple versions of same integration
3. **Rollback Support**: Restore previous build on failure
4. **Container Support**: Build in Docker containers
5. **Package Signing**: Verify package signatures
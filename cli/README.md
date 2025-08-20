# SecAuto CLI

A comprehensive command-line interface for the SecAuto Security Orchestration, Automation & Response (SOAR) platform.

## Features

- **Playbook Management**: Execute, upload, and manage playbooks
- **Job Monitoring**: Track job status and execution history
- **Cache Operations**: Manage cache entries and view statistics
- **Integration Management**: Execute integrations and list automations
- **Cluster Monitoring**: View cluster status and health
- **Configuration Profiles**: Manage multiple server environments
- **Multiple Output Formats**: Table, JSON, and YAML output
- **Health Checking**: Verify server connectivity and status

## Installation

### From Source

```bash
git clone <repository-url>
cd SecAuto/cli
go mod tidy
go build -o secauto-cli .
```

### Binary Installation

Download the appropriate binary for your platform from the releases page.

## Quick Start

1. **Configure the CLI**:
   ```bash
   # Set server URL and API key
   secauto config set server http://localhost:9090
   secauto config set api-key your-api-key-here
   ```

2. **Check server health**:
   ```bash
   secauto health
   ```

3. **List available playbooks**:
   ```bash
   secauto playbook list
   ```

4. **Execute a playbook**:
   ```bash
   secauto playbook execute my-playbook.json
   ```

## Commands

### Global Flags

- `--server`: SecAuto server URL
- `--api-key`: API key for authentication
- `--output`: Output format (table, json, yaml)
- `--no-color`: Disable colored output
- `--verbose`: Enable verbose output
- `--config`: Custom config file path

### Playbook Commands

```bash
# Execute a playbook synchronously
secauto playbook execute playbook.json

# Execute a playbook asynchronously
secauto playbook execute --async playbook.json

# Execute with custom context
secauto playbook execute playbook.json --context '{"key": "value"}'

# Execute by name (uploaded playbook)
secauto playbook execute --name my-playbook

# Upload a playbook
secauto playbook upload playbook.json custom-name

# List available playbooks
secauto playbook list
```

### Job Commands

```bash
# List all jobs
secauto job list

# Filter jobs by status
secauto job list --status completed

# Get specific job details
secauto job get <job-id>

# Watch a job until completion
secauto job watch <job-id>

# Show job statistics
secauto job stats
```

### Cache Commands

```bash
# View cache statistics
secauto cache stats

# Get a cache value
secauto cache get my-key

# Set a cache value
secauto cache set my-key "my-value"

# Set JSON cache value
secauto cache set my-key '{"data": "value"}' --json

# Delete a cache key
secauto cache delete my-key

# Clear all cache entries
secauto cache clear
```

### Integration Commands

```bash
# List available integrations
secauto integration list

# Execute an integration
secauto integration execute virustotal --params '{"url": "https://example.com"}'

# Execute with parameters from file
secauto integration execute qualys --params-file params.json
```

### Automation Commands

```bash
# List automations
secauto automation list

# Upload an automation script
secauto automation upload my_automation.py

# Delete an automation script
secauto automation delete my_automation
```

### Client Management Commands

```bash
# List all clients
secauto client list

# Get client details
secauto client get <client-id>

# Delete a client
secauto client delete <client-id>
```

### API Key Management Commands

```bash
# List all API keys
secauto apikey list

# Create a new API key
secauto apikey create [name] --description "My API key"

# Delete an API key
secauto apikey delete <key-id>

# Show API key usage statistics
secauto apikey stats
```

### Batch Operations

```bash
# Execute multiple playbooks from a directory
secauto batch playbook ./playbooks/ --async --max-concurrent 5

# Upload multiple files from a directory
secauto batch upload ./playbooks/ --type playbook
secauto batch upload ./automations/ --type automation
```

### Cluster Commands

```bash
# View cluster status
secauto cluster status
```

### Configuration Commands

```bash
# Show current configuration
secauto config show

# Set configuration values
secauto config set server http://localhost:9090
secauto config set api-key your-key
secauto config set output json

# Manage profiles
secauto config profile list
secauto config profile add production --server https://prod.example.com --api-key prod-key
secauto config profile use production
secauto config profile remove staging
```

### Health Commands

```bash
# Check server health
secauto health
```

## Configuration

The CLI supports multiple configuration methods:

### Configuration File

Default location: `~/.config/secauto-cli/config.yaml`

```yaml
server: http://localhost:9090
api_key: your-api-key-here
output: table
no_color: false
verbose: false
current: default
profiles:
  default:
    name: default
    server: http://localhost:9090
    api_key: dev-key
    description: Development environment
  production:
    name: production
    server: https://secauto.company.com
    api_key: prod-key
    description: Production environment
```

### Environment Variables

- `SECAUTO_SERVER`: Server URL
- `SECAUTO_API_KEY`: API key
- `SECAUTO_OUTPUT`: Output format
- `SECAUTO_NO_COLOR`: Disable colors (true/false)
- `SECAUTO_VERBOSE`: Enable verbose output (true/false)

### Command Line Flags

All configuration can be overridden with command-line flags.

## Profiles

Profiles allow you to manage multiple SecAuto environments:

```bash
# Add a new profile
secauto config profile add staging \
  --server https://staging.example.com \
  --api-key staging-key \
  --description "Staging environment"

# Switch to the profile
secauto config profile use staging

# List all profiles
secauto config profile list
```

## Output Formats

### Table (Default)
Human-readable tabular output with colors and formatting.

### JSON
Machine-readable JSON output:
```bash
secauto job list --output json
```

### YAML
YAML formatted output:
```bash
secauto playbook list --output yaml
```

## Examples

### Execute Playbook with Context

```bash
# Execute with inline context
secauto playbook execute incident-response.json --context '{
  "incident_id": "INC-001",
  "severity": "high",
  "source_ip": "192.168.1.100"
}'

# Execute with context from file
echo '{"incident_id": "INC-001"}' > context.json
secauto playbook execute incident-response.json --context context.json
```

### Monitor Async Execution

```bash
# Start async execution and watch
secauto playbook execute --async --watch investigation.json

# Or start and check later
secauto playbook execute --async investigation.json
# ... do other work ...
secauto job get <job-id>
```

### Integration Execution

```bash
# VirusTotal URL scan
secauto integration execute virustotal --params '{
  "url": "https://suspicious-site.com"
}'

# Qualys asset scan
secauto integration execute qualys --params '{
  "target": "192.168.1.0/24",
  "scan_type": "discovery"
}'
```

### Batch Operations

```bash
# List multiple job statuses
for job in $(secauto job list --output json | jq -r '.[] | select(.status=="running") | .id'); do
  echo "Job $job status:"
  secauto job get $job
done
```

## Error Handling

The CLI provides detailed error messages and exit codes:

- `0`: Success
- `1`: General error
- `2`: Configuration error
- `3`: Authentication error
- `4`: Server communication error

## Development

### Building

```bash
# Build for current platform
go build -o secauto-cli .

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o secauto-cli-linux .
GOOS=windows GOARCH=amd64 go build -o secauto-cli.exe .
GOOS=darwin GOARCH=amd64 go build -o secauto-cli-mac .
```

### Testing

```bash
# Run tests
go test ./...

# Test against live server
export SECAUTO_SERVER=http://localhost:9090
export SECAUTO_API_KEY=test-key
go test -tags integration ./...
```

## Troubleshooting

### Common Issues

1. **Connection Refused**:
   - Verify server URL is correct
   - Ensure SecAuto server is running
   - Check network connectivity

2. **Authentication Error**:
   - Verify API key is correct
   - Check if API key has required permissions

3. **SSL/TLS Errors**:
   - For self-signed certificates, use HTTP instead of HTTPS for testing
   - Configure proper SSL certificates on the server

### Debug Mode

Enable verbose output for debugging:

```bash
secauto --verbose health
secauto --verbose playbook execute test.json
```

### Configuration Issues

```bash
# Check current configuration
secauto config show

# Reset configuration
rm ~/.config/secauto-cli/config.yaml
secauto config set server http://localhost:9090
secauto config set api-key your-key
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

[Your License Here]

## Support

For issues and questions:
- GitHub Issues: [Link to issues]
- Documentation: [Link to docs]
- Support Email: support@example.com
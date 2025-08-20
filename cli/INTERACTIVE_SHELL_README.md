# SecAuto Interactive CLI

A modern, interactive command-line interface for the SecAuto Security Orchestration, Automation & Response (SOAR) platform built with [ishell](https://github.com/abiosoft/ishell).

## Features

### Interactive Experience
- **Live Shell**: Stay connected with persistent sessions
- **Tab Completion**: Auto-complete commands and arguments
- **Command History**: Navigate previous commands with arrow keys
- **Real-time Feedback**: Instant validation and error messages
- **Colored Output**: Enhanced readability with syntax highlighting
- **Progress Indicators**: Visual feedback for long-running operations

### Core Functionality
- **Configuration Management**: Easy server and API key setup with profiles
- **Playbook Operations**: Execute, upload, validate, and manage playbooks
- **Job Monitoring**: Real-time job tracking and status updates
- **Automation Management**: Upload and manage Python automation scripts
- **Client Administration**: Manage connected clients and their integrations
- **API Key Management**: Create, list, and manage API keys
- **Health Monitoring**: Check server connectivity and status

## Getting Started

### Installation

```bash
# Build the interactive CLI
go build -o secauto-ishell main_ishell.go

# Run the interactive shell
./secauto-ishell
```

### First Time Setup

1. **Start the Interactive Shell**:
   ```bash
   ./secauto-ishell
   ```

2. **Configure Server and API Key**:
   ```
   secauto> config server http://localhost:9090
   secauto> config apikey your-api-key-here
   ```

3. **Verify Connection**:
   ```
   secauto> health
   secauto> status
   ```

## Command Reference

### Global Commands

#### Configuration
```bash
# Show current configuration
config show

# Set server URL
config server <url>

# Set API key
config apikey <key>

# Set output format (table, json, yaml)
config output <format>

# Enable/disable colors
config colors <true|false>
```

#### Profile Management
```bash
# List profiles
config profile list

# Add new profile
config profile add <name>

# Switch to profile
config profile use <name>

# Remove profile
config profile remove <name>
```

#### System Commands
```bash
# Show system status
status

# Check server health
health

# Show version information
version

# Clear screen
clear

# Show help
help

# Exit shell
exit
```

### Playbook Commands

```bash
# List all playbooks
playbook list

# Execute a playbook file
playbook execute my-playbook.json

# Execute with context
playbook execute my-playbook.json --context '{"key": "value"}'

# Execute asynchronously
playbook execute my-playbook.json --async

# Execute and watch progress
playbook execute my-playbook.json --async --watch

# Execute uploaded playbook by name
playbook execute uploaded-playbook-name

# Upload a playbook
playbook upload my-playbook.json [custom-name]

# Validate playbook syntax
playbook validate my-playbook.json
```

### Job Management

```bash
# List all jobs
job list

# Get job details
job get <job-id>

# Show job statistics
job stats
```

### Automation Management

```bash
# List automations
automation list

# Upload automation script
automation upload script.py

# Delete automation
automation delete script-name
```

### Client Management

```bash
# List all clients
client list

# Get client details
client get <client-id>

# Delete client
client delete <client-id>
```

### API Key Management

```bash
# List API keys
apikey list

# Create new API key
apikey create [name]

# Delete API key
apikey delete <key-id>

# Show usage statistics
apikey stats
```

## Interactive Features

### Command Line Arguments
You can also run commands non-interactively:

```bash
# Execute single command
./secauto-ishell "health"

# Execute multiple commands
./secauto-ishell "config show; health; playbook list"
```

### Tab Completion
The shell supports intelligent tab completion:
- Command names
- Subcommands
- File paths for uploads
- Available playbook names

### History and Navigation
- **↑/↓ Arrow Keys**: Navigate command history
- **Ctrl+R**: Reverse search through history
- **Ctrl+L**: Clear screen
- **Ctrl+C**: Interrupt current command
- **Ctrl+D**: Exit shell

### Confirmation Prompts
Destructive operations require confirmation:
```bash
secauto> automation delete my-script
Are you sure you want to delete automation 'my-script'? (y/N): y
✓ Automation 'my-script' deleted successfully
```

## Advanced Usage

### Configuration Profiles
Manage multiple SecAuto environments:

```bash
# Add development profile
secauto> config profile add dev
Server URL: http://localhost:9090
API Key: dev-api-key
Description (optional): Development environment

# Add production profile
secauto> config profile add prod
Server URL: https://secauto.company.com
API Key: prod-api-key
Description (optional): Production environment

# Switch between environments
secauto> config profile use prod
✓ Switched to profile: prod

secauto> config profile use dev
✓ Switched to profile: dev
```

### Batch Operations
Execute multiple operations efficiently:

```bash
# Upload multiple automations
secauto> automation upload script1.py
secauto> automation upload script2.py
secauto> automation upload script3.py

# Execute multiple playbooks
secauto> playbook execute playbook1.json --async
secauto> playbook execute playbook2.json --async
secauto> job list
```

### Context Management
Pass context to playbooks:

```bash
# Inline JSON context
secauto> playbook execute incident-response.json --context '{"severity": "high", "incident_id": "INC-001"}'

# Context from file
secauto> playbook execute incident-response.json --context context.json
```

### Output Formatting
Switch between output formats on the fly:

```bash
secauto> config output json
secauto> job list
# Shows JSON output

secauto> config output table
secauto> job list
# Shows table output
```

## Error Handling

The interactive shell provides detailed error messages and suggestions:

```bash
secauto> playbook execute
Error: Missing required arguments
Usage: playbook execute <file|name> [--async] [--context <json>]

secauto> health
✗ server health check failed: connection refused

secauto> config show
=== Current Configuration ===
Server: (not set)
API Key: (not set)
# Use 'config server <url>' to set server URL
```

## Troubleshooting

### Common Issues

1. **Connection Problems**:
   ```bash
   secauto> status
   # Check if server and API key are configured
   secauto> health
   # Verify connectivity
   ```

2. **Configuration Issues**:
   ```bash
   secauto> config show
   # Review current settings
   secauto> config server http://localhost:9090
   secauto> config apikey your-api-key
   ```

3. **Profile Problems**:
   ```bash
   secauto> config profile list
   # Check available profiles
   secauto> config profile use default
   # Switch to default profile
   ```

### Debug Mode
Enable verbose output for troubleshooting:

```bash
secauto> config verbose true
# All subsequent commands will show detailed output
```

## Comparison with Traditional CLI

| Feature | Traditional CLI | Interactive CLI |
|---------|----------------|-----------------|
| **Session Persistence** | No | Yes |
| **Tab Completion** | Limited | Full support |
| **Command History** | Shell-dependent | Built-in |
| **Real-time Feedback** | No | Yes |
| **Context Awareness** | No | Yes |
| **Configuration Management** | File-based | Interactive |
| **Error Handling** | Exit codes | Rich messages |
| **Progress Indicators** | Basic | Advanced |

## Development

### Adding New Commands

1. **Create Command File**:
   ```go
   // shell/mycommand.go
   package shell

   func RegisterMyCommands(sh *ishell.Shell, ctx *Context) {
       myCmd := &ishell.Cmd{
           Name: "mycommand",
           Help: "Description of my command",
       }
       
       myCmd.AddCmd(&ishell.Cmd{
           Name: "subcommand",
           Help: "Subcommand description",
           Func: func(c *ishell.Context) {
               // Command implementation
           },
       })
       
       sh.AddCmd(myCmd)
   }
   ```

2. **Register in Main**:
   ```go
   // main_ishell.go
   shell.RegisterMyCommands(sh, ctx)
   ```

### Best Practices

1. **Always check connectivity**: Use `ctx.RequireConnection(c)`
2. **Validate arguments**: Use `ctx.RequireArgs(c, count, usage)`
3. **Provide feedback**: Use progress spinners for long operations
4. **Handle errors gracefully**: Use `ctx.PrintError(c, err)`
5. **Confirm destructive operations**: Use interactive prompts

## Future Enhancements

- **Scripting Support**: Execute command scripts from files
- **Plugin System**: Load custom commands dynamically
- **Advanced Autocomplete**: Context-aware suggestions
- **Session Recording**: Save and replay command sessions
- **Multi-server Management**: Connect to multiple servers simultaneously
- **Real-time Notifications**: Live updates for job status changes

## Contributing

The interactive shell is built on top of the existing CLI infrastructure, making it easy to add new commands or extend functionality. See the main project README for contribution guidelines.
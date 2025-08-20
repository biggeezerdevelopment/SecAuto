package shell

import (
	"fmt"

	"github.com/abiosoft/ishell/v2"
	"secauto-cli/pkg/client"
	"secauto-cli/pkg/config"
	"secauto-cli/pkg/multiserver"
	"secauto-cli/pkg/output"
)

// Context holds the shared state for all shell commands
type Context struct {
	Config      *config.Config
	Shell       *ishell.Shell
	Printer     *output.Printer
	Client      *client.Client
	MultiServer *multiserver.Manager
}

// EnsureClient ensures that the API client is initialized
func (ctx *Context) EnsureClient() error {
	if ctx.Client == nil {
		if ctx.Config.Server == "" {
			return fmt.Errorf("server not configured. Use 'config server <url>' to set server URL")
		}
		if ctx.Config.APIKey == "" {
			return fmt.Errorf("API key not configured. Use 'config apikey <key>' to set API key")
		}
		ctx.Client = client.NewClient(ctx.Config.Server, ctx.Config.APIKey)
	}
	return nil
}

// UpdateConfig updates the configuration and refreshes dependent components
func (ctx *Context) UpdateConfig() {
	ctx.Printer = output.NewPrinter(ctx.Config.Output, ctx.Config.NoColor)
	if ctx.Config.Server != "" && ctx.Config.APIKey != "" {
		ctx.Client = client.NewClient(ctx.Config.Server, ctx.Config.APIKey)
	} else {
		ctx.Client = nil
	}
}

// PrintError prints an error message with consistent formatting
func (ctx *Context) PrintError(c *ishell.Context, err error) {
	ctx.Printer.PrintError(err.Error())
}

// PrintSuccess prints a success message with consistent formatting
func (ctx *Context) PrintSuccess(c *ishell.Context, message string) {
	ctx.Printer.PrintSuccess(message)
}

// PrintInfo prints an info message with consistent formatting
func (ctx *Context) PrintInfo(c *ishell.Context, message string) {
	ctx.Printer.PrintInfo(message)
}

// RequireArgs checks if the required number of arguments are provided
func (ctx *Context) RequireArgs(c *ishell.Context, required int, usage string) bool {
	if len(c.Args) < required {
		c.Printf("Error: Missing required arguments\n")
		c.Printf("Usage: %s\n", usage)
		return false
	}
	return true
}

// RequireConnection ensures server and API key are configured and connection works
func (ctx *Context) RequireConnection(c *ishell.Context) bool {
	if err := ctx.EnsureClient(); err != nil {
		ctx.PrintError(c, err)
		return false
	}
	
	if err := ctx.Client.HealthCheck(); err != nil {
		ctx.PrintError(c, fmt.Errorf("server health check failed: %v", err))
		return false
	}
	
	return true
}
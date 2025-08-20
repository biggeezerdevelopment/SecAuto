package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abiosoft/ishell/v2"
	"github.com/fatih/color"
	"secauto-cli/pkg/client"
	"secauto-cli/pkg/config"
	"secauto-cli/pkg/multiserver"
	"secauto-cli/pkg/output"
	"secauto-cli/shell"
)

const (
	shellPrompt = "secauto"
	version     = "2.0.0-interactive"
)

func main() {
	// Create new shell
	sh := ishell.New()

	// Configure shell
	sh.Println("SecAuto Interactive CLI")
	sh.Println("Type 'help' for available commands or 'exit' to quit")
	sh.SetPrompt(fmt.Sprintf("%s> ", color.CyanString(shellPrompt)))

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		sh.Printf("Warning: Could not load config: %v\n", err)
		cfg = &config.Config{
			Output:  "table",
			NoColor: false,
		}
	}

	// Initialize multi-server manager
	dbPath := filepath.Join(os.Getenv("HOME"), ".secauto", "servers.db")
	multiServer, err := multiserver.NewManager(dbPath)
	if err != nil {
		sh.Printf("Warning: Could not initialize multi-server manager: %v\n", err)
		// Continue without multi-server support
	}

	// Create context for commands
	ctx := &shell.Context{
		Config:      cfg,
		Shell:       sh,
		Printer:     output.NewPrinter(cfg.Output, cfg.NoColor),
		Client:      nil, // Will be initialized when server/api-key are set
		MultiServer: multiServer,
	}

	// Register all commands
	shell.RegisterConfigCommands(sh, ctx)
	shell.RegisterHealthCommands(sh, ctx)
	shell.RegisterPlaybookCommands(sh, ctx)
	shell.RegisterJobCommands(sh, ctx)
	shell.RegisterCacheCommands(sh, ctx)
	shell.RegisterAutomationCommands(sh, ctx)
	shell.RegisterIntegrationCommands(sh, ctx)
	shell.RegisterClientCommands(sh, ctx)
	shell.RegisterAPIKeyCommands(sh, ctx)
	shell.RegisterBatchCommands(sh, ctx)
	shell.RegisterClusterCommands(sh, ctx)
	shell.RegisterScheduleCommands(sh, ctx)
	
	// Register multi-server commands if available
	if multiServer != nil {
		shell.RegisterServerCommands(sh, ctx)
		shell.RegisterMultiServerPlaybookCommands(sh, ctx)
	}

	// Add global commands
	registerGlobalCommands(sh, ctx)

	// Check for command line arguments to run non-interactively
	if len(os.Args) > 1 {
		commandLine := strings.Join(os.Args[1:], " ")
		sh.Process(commandLine)
		return
	}

	// Start interactive shell
	sh.Run()
}

func registerGlobalCommands(sh *ishell.Shell, ctx *shell.Context) {
	// Version command
	sh.AddCmd(&ishell.Cmd{
		Name: "version",
		Help: "Show version information",
		Func: func(c *ishell.Context) {
			c.Printf("SecAuto CLI Version: %s\n", version)
			c.Printf("Interactive Shell: %s\n", color.GreenString("enabled"))
			if ctx.Config.Server != "" {
				c.Printf("Server: %s\n", ctx.Config.Server)
			}
			if ctx.Config.APIKey != "" {
				c.Printf("API Key: %s\n", maskAPIKey(ctx.Config.APIKey))
			}
		},
	})

	// Clear command
	sh.AddCmd(&ishell.Cmd{
		Name: "clear",
		Help: "Clear the screen",
		Func: func(c *ishell.Context) {
			c.ClearScreen()
		},
	})

	// Status command - shows current configuration and connection status
	sh.AddCmd(&ishell.Cmd{
		Name: "status",
		Help: "Show current configuration and connection status",
		Func: func(c *ishell.Context) {
			c.Printf("=== SecAuto CLI Status ===\n")
			c.Printf("Version: %s\n", version)
			c.Printf("Output format: %s\n", ctx.Config.Output)
			c.Printf("Colors: %s\n", boolToStatus(!ctx.Config.NoColor))
			
			if ctx.Config.Server == "" {
				c.Printf("Server: %s\n", color.RedString("not configured"))
				c.Printf("Use 'config server <url>' to set server URL\n")
			} else {
				c.Printf("Server: %s\n", ctx.Config.Server)
				
				if ctx.Config.APIKey == "" {
					c.Printf("API Key: %s\n", color.RedString("not configured"))
					c.Printf("Use 'config apikey <key>' to set API key\n")
				} else {
					c.Printf("API Key: %s\n", color.GreenString("configured"))
					
					// Test connection
					if ctx.Client == nil {
						ctx.Client = client.NewClient(ctx.Config.Server, ctx.Config.APIKey)
					}
					
					if err := ctx.Client.HealthCheck(); err != nil {
						c.Printf("Connection: %s (%v)\n", color.RedString("failed"), err)
					} else {
						c.Printf("Connection: %s\n", color.GreenString("healthy"))
					}
				}
			}
		},
	})

	// Help command with better formatting
	sh.AddCmd(&ishell.Cmd{
		Name: "help",
		Help: "Show available commands and usage",
		Func: func(c *ishell.Context) {
			c.Printf("=== SecAuto CLI Commands ===\n\n")
			
			categories := map[string][]string{
				"Configuration": {"config", "status", "version"},
				"Server": {"health"},
				"Playbooks": {"playbook"},
				"Jobs": {"job"},
				"Cache": {"cache"},
				"Automations": {"automation"},
				"Integrations": {"integration"},
				"Clients": {"client"},
				"API Keys": {"apikey"},
				"Batch Operations": {"batch"},
				"Cluster": {"cluster"},
				"Schedules": {"schedule"},
				"General": {"help", "clear", "exit"},
			}
			
			for category, commands := range categories {
				c.Printf("%s:\n", color.CyanString(category))
				for _, cmd := range commands {
					if shellCmd := findCommand(sh, cmd); shellCmd != nil {
						c.Printf("  %-12s %s\n", cmd, shellCmd.Help)
					}
				}
				c.Printf("\n")
			}
			
			c.Printf("Use '<command> help' for detailed command usage\n")
			c.Printf("Use 'status' to check connection and configuration\n")
		},
	})
}

func findCommand(sh *ishell.Shell, name string) *ishell.Cmd {
	for _, cmd := range sh.Cmds() {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

func boolToStatus(b bool) string {
	if b {
		return color.GreenString("enabled")
	}
	return color.RedString("disabled")
}
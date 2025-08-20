package shell

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abiosoft/ishell/v2"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"secauto-cli/pkg/client"
)

// RegisterServerCommands registers server management commands
func RegisterServerCommands(sh *ishell.Shell, ctx *Context) {
	serverCmd := &ishell.Cmd{
		Name: "server",
		Help: "Manage multiple SecAuto servers",
	}

	// server add
	serverCmd.AddCmd(&ishell.Cmd{
		Name: "add",
		Help: "Add a new server (add <name> <url> <api-key> [description])",
		Func: func(c *ishell.Context) {
			if len(c.Args) < 3 {
				c.Printf("Usage: server add <name> <url> <api-key> [description]\n")
				c.Printf("Example: server add prod https://prod.secauto.com:9090 prod-api-key-123 'Production server'\n")
				return
			}

			name := c.Args[0]
			url := c.Args[1]
			apiKey := c.Args[2]
			description := ""
			if len(c.Args) > 3 {
				description = strings.Join(c.Args[3:], " ")
			}

			// Ask if this should be the default server
			c.Printf("Set as default server? (y/N): ")
			isDefaultStr := c.ReadLine()
			isDefault := isDefaultStr == "y" || isDefaultStr == "yes"

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Adding server and testing connection..."
			}

			err := ctx.MultiServer.AddServer(name, url, apiKey, description, isDefault)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to add server: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Server '%s' added successfully", name))
		},
	})

	// server list
	serverCmd.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "List all configured servers",
		Func: func(c *ishell.Context) {
			servers, err := ctx.MultiServer.ListServers()
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to list servers: %v", err))
				return
			}

			if len(servers) == 0 {
				c.Printf("No servers configured. Use 'server add' to add a server.\n")
				return
			}

			// Convert to format suitable for table output
			var data []map[string]interface{}
			for _, server := range servers {
				status := color.GreenString("Active")
				if !server.IsActive {
					status = color.RedString("Inactive")
				}

				defaultStr := ""
				if server.IsDefault {
					defaultStr = color.CyanString("★")
				}

				data = append(data, map[string]interface{}{
					"Default": defaultStr,
					"Name":    server.Name,
					"URL":     server.URL,
					"Status":  status,
					"Description": server.Description,
				})
			}

			if err := ctx.Printer.Print(data); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// server remove
	serverCmd.AddCmd(&ishell.Cmd{
		Name: "remove",
		Help: "Remove a server (remove <name>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireArgs(c, 1, "server remove <name>") {
				return
			}

			name := c.Args[0]

			// Confirm deletion
			c.Printf("Are you sure you want to remove server '%s'? (y/N): ", name)
			confirmation := c.ReadLine()
			if confirmation != "y" && confirmation != "yes" {
				c.Printf("Removal cancelled\n")
				return
			}

			err := ctx.MultiServer.RemoveServer(name)
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to remove server: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Server '%s' removed successfully", name))
		},
	})

	// server default
	serverCmd.AddCmd(&ishell.Cmd{
		Name: "default",
		Help: "Set default server (default <name>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireArgs(c, 1, "server default <name>") {
				return
			}

			name := c.Args[0]

			err := ctx.MultiServer.SetDefaultServer(name)
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to set default server: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Server '%s' set as default", name))
		},
	})

	// server enable
	serverCmd.AddCmd(&ishell.Cmd{
		Name: "enable",
		Help: "Enable a server (enable <name>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireArgs(c, 1, "server enable <name>") {
				return
			}

			name := c.Args[0]

			err := ctx.MultiServer.ToggleServer(name, true)
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to enable server: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Server '%s' enabled", name))
		},
	})

	// server disable
	serverCmd.AddCmd(&ishell.Cmd{
		Name: "disable",
		Help: "Disable a server (disable <name>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireArgs(c, 1, "server disable <name>") {
				return
			}

			name := c.Args[0]

			err := ctx.MultiServer.ToggleServer(name, false)
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to disable server: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Server '%s' disabled", name))
		},
	})

	// server test
	serverCmd.AddCmd(&ishell.Cmd{
		Name: "test",
		Help: "Test connectivity to all servers",
		Func: func(c *ishell.Context) {
			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Testing server connectivity..."
			}

			results, err := ctx.MultiServer.TestAllServers()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to test servers: %v", err))
				return
			}

			// Display results
			c.Printf("\n=== Server Connectivity Test Results ===\n")
			for _, result := range results {
				status := color.GreenString("✓ Connected")
				if !result.Success {
					status = color.RedString("✗ Failed: " + result.Error)
				}
				c.Printf("%s: %s (Duration: %s)\n", result.ServerName, status, result.Duration)
			}
		},
	})

	// server sync
	serverCmd.AddCmd(&ishell.Cmd{
		Name: "sync",
		Help: "Sync files to all servers (sync <directory> <playbook|automation>)",
		Func: func(c *ishell.Context) {
			if len(c.Args) < 2 {
				c.Printf("Usage: server sync <directory> <playbook|automation>\n")
				c.Printf("Example: server sync ./playbooks playbook\n")
				c.Printf("Example: server sync ./automations automation\n")
				return
			}

			directory := c.Args[0]
			fileType := c.Args[1]

			if fileType != "playbook" && fileType != "automation" {
				ctx.PrintError(c, fmt.Errorf("invalid file type: %s (must be 'playbook' or 'automation')", fileType))
				return
			}

			// Check if directory exists
			if _, err := os.Stat(directory); os.IsNotExist(err) {
				ctx.PrintError(c, fmt.Errorf("directory not found: %s", directory))
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = fmt.Sprintf(" Syncing %ss from %s to all servers...", fileType, directory)
			}

			err := ctx.MultiServer.SyncDirectory(directory, fileType)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, err)
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Successfully synced %ss to all servers", fileType))
		},
	})

	// server history
	serverCmd.AddCmd(&ishell.Cmd{
		Name: "history",
		Help: "Show execution history (history [server-name] [limit])",
		Func: func(c *ishell.Context) {
			serverName := ""
			limit := 20

			if len(c.Args) > 0 {
				serverName = c.Args[0]
			}
			if len(c.Args) > 1 {
				fmt.Sscanf(c.Args[1], "%d", &limit)
			}

			history, err := ctx.MultiServer.GetExecutionHistory(serverName, limit)
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get execution history: %v", err))
				return
			}

			if len(history) == 0 {
				c.Printf("No execution history found\n")
				return
			}

			// Convert to format suitable for table output
			var data []map[string]interface{}
			for _, exec := range history {
				status := color.GreenString(exec.Status)
				if exec.Status == "failed" {
					status = color.RedString(exec.Status)
				}

				data = append(data, map[string]interface{}{
					"Time":   exec.ExecutedAt.Format("2006-01-02 15:04:05"),
					"Server": exec.ServerName,
					"Type":   exec.Type,
					"Name":   exec.Name,
					"Status": status,
				})
			}

			if err := ctx.Printer.Print(data); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// server stats
	serverCmd.AddCmd(&ishell.Cmd{
		Name: "stats",
		Help: "Show execution statistics",
		Func: func(c *ishell.Context) {
			stats, err := ctx.MultiServer.GetExecutionStats()
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get statistics: %v", err))
				return
			}

			c.Printf("\n=== Execution Statistics ===\n")
			if err := ctx.Printer.Print(stats); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	sh.AddCmd(serverCmd)
}

// RegisterMultiServerPlaybookCommands extends playbook commands for multi-server
func RegisterMultiServerPlaybookCommands(sh *ishell.Shell, ctx *Context) {
	// Find the existing playbook command
	for _, cmd := range sh.Cmds() {
		if cmd.Name == "playbook" {
			// Add multi-server commands
			cmd.AddCmd(&ishell.Cmd{
				Name: "upload-all",
				Help: "Upload playbook to all servers (upload-all <file>)",
				Func: func(c *ishell.Context) {
					if !ctx.RequireArgs(c, 1, "playbook upload-all <file>") {
						return
					}

					playbookFile := c.Args[0]

					// Read and parse playbook
					content, err := os.ReadFile(playbookFile)
					if err != nil {
						ctx.PrintError(c, fmt.Errorf("failed to read playbook: %v", err))
						return
					}

					var playbook interface{}
					if err := json.Unmarshal(content, &playbook); err != nil {
						ctx.PrintError(c, fmt.Errorf("invalid playbook JSON: %v", err))
						return
					}

					name := strings.TrimSuffix(filepath.Base(playbookFile), ".json")

					s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
					if !ctx.Config.NoColor {
						s.Start()
						s.Suffix = " Uploading playbook to all servers..."
					}

					results, err := ctx.MultiServer.UploadPlaybookToAll(name, playbook)

					if !ctx.Config.NoColor {
						s.Stop()
					}

					if err != nil {
						ctx.PrintError(c, fmt.Errorf("failed to upload playbook: %v", err))
						return
					}

					// Display results
					c.Printf("\n=== Upload Results ===\n")
					for _, result := range results {
						if result.Success {
							c.Printf("%s: %s\n", result.ServerName, color.GreenString("✓ Success"))
						} else {
							c.Printf("%s: %s\n", result.ServerName, color.RedString("✗ "+result.Error))
						}
					}
				},
			})

			cmd.AddCmd(&ishell.Cmd{
				Name: "execute-all",
				Help: "Execute playbook on all servers (execute-all <file|name>)",
				Func: func(c *ishell.Context) {
					if !ctx.RequireArgs(c, 1, "playbook execute-all <file|name>") {
						return
					}

					input := c.Args[0]
					var req client.PlaybookRequest

					// Check if it's a file
					if strings.HasSuffix(input, ".json") {
						content, err := os.ReadFile(input)
						if err != nil {
							ctx.PrintError(c, fmt.Errorf("failed to read playbook: %v", err))
							return
						}

						var playbook interface{}
						if err := json.Unmarshal(content, &playbook); err != nil {
							ctx.PrintError(c, fmt.Errorf("invalid playbook JSON: %v", err))
							return
						}
						req.Playbook = playbook
					} else {
						req.PlaybookName = input
					}

					s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
					if !ctx.Config.NoColor {
						s.Start()
						s.Suffix = " Executing playbook on all servers..."
					}

					results, err := ctx.MultiServer.ExecutePlaybookOnAll(&req)

					if !ctx.Config.NoColor {
						s.Stop()
					}

					if err != nil {
						ctx.PrintError(c, fmt.Errorf("failed to execute playbook: %v", err))
						return
					}

					// Display results
					c.Printf("\n=== Execution Results ===\n")
					for _, result := range results {
						if result.Success {
							c.Printf("%s: %s (Duration: %s)\n", 
								result.ServerName, 
								color.GreenString("✓ Success"), 
								result.Duration)
						} else {
							c.Printf("%s: %s\n", 
								result.ServerName, 
								color.RedString("✗ "+result.Error))
						}
					}
				},
			})
			break
		}
	}
}
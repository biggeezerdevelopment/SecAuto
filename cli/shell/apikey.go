package shell

import (
	"fmt"
	"strings"
	"time"

	"github.com/abiosoft/ishell/v2"
	"github.com/briandowns/spinner"
)

// RegisterAPIKeyCommands registers API key-related commands
func RegisterAPIKeyCommands(sh *ishell.Shell, ctx *Context) {
	apikeyCmd := &ishell.Cmd{
		Name: "apikey",
		Help: "Manage API keys",
	}

	// apikey list
	apikeyCmd.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "List all API keys",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching API keys..."
			}

			apiKeys, err := ctx.Client.ListAPIKeys()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to list API keys: %v", err))
				return
			}

			if len(apiKeys) == 0 {
				c.Printf("No API keys found\n")
				return
			}

			if err := ctx.Printer.Print(apiKeys); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// apikey create
	apikeyCmd.AddCmd(&ishell.Cmd{
		Name: "create",
		Help: "Create a new API key (create [name])",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			var name string
			if len(c.Args) > 0 {
				name = c.Args[0]
			} else {
				// Prompt for name
				c.Print("API Key name (optional): ")
				name = c.ReadLine()
			}

			// Prompt for description
			c.Print("Description (optional): ")
			description := c.ReadLine()

			// Prompt for permissions
			c.Print("Permissions (comma-separated, optional): ")
			permissionsStr := c.ReadLine()
			var permissions []string
			if permissionsStr != "" {
				permissions = strings.Split(permissionsStr, ",")
				for i, perm := range permissions {
					permissions[i] = strings.TrimSpace(perm)
				}
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Creating API key..."
			}

			apiKeyInfo, err := ctx.Client.CreateAPIKey(name, description, permissions)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to create API key: %v", err))
				return
			}

			ctx.PrintSuccess(c, "API key created successfully")
			if err := ctx.Printer.Print(apiKeyInfo); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// apikey delete
	apikeyCmd.AddCmd(&ishell.Cmd{
		Name: "delete",
		Help: "Delete an API key (delete <key-id>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "apikey delete <key-id>") {
				return
			}

			keyID := c.Args[0]

			// Confirm deletion
			c.Printf("Are you sure you want to delete API key '%s'? (y/N): ", keyID)
			confirmation := c.ReadLine()
			if confirmation != "y" && confirmation != "yes" && confirmation != "Y" && confirmation != "YES" {
				c.Printf("Deletion cancelled\n")
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Deleting API key..."
			}

			err := ctx.Client.DeleteAPIKey(keyID)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("delete failed: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("API key '%s' deleted successfully", keyID))
		},
	})

	// apikey stats
	apikeyCmd.AddCmd(&ishell.Cmd{
		Name: "stats",
		Help: "Show API key usage statistics",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching API key statistics..."
			}

			stats, err := ctx.Client.GetAPIKeyStats()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get API key stats: %v", err))
				return
			}

			if err := ctx.Printer.Print(stats); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	sh.AddCmd(apikeyCmd)
}
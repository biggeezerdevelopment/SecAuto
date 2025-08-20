package shell

import (
	"fmt"
	"strings"

	"github.com/abiosoft/ishell/v2"
	"github.com/fatih/color"
	"secauto-cli/pkg/config"
)

// RegisterConfigCommands registers configuration-related commands
func RegisterConfigCommands(sh *ishell.Shell, ctx *Context) {
	configCmd := &ishell.Cmd{
		Name: "config",
		Help: "Manage CLI configuration",
	}

	// config show
	configCmd.AddCmd(&ishell.Cmd{
		Name: "show",
		Help: "Show current configuration",
		Func: func(c *ishell.Context) {
			c.Printf("=== Current Configuration ===\n")
			c.Printf("Server: %s\n", valueOrEmpty(ctx.Config.Server))
			c.Printf("API Key: %s\n", maskAPIKey(ctx.Config.APIKey))
			c.Printf("Output: %s\n", ctx.Config.Output)
			c.Printf("No Color: %t\n", ctx.Config.NoColor)
			c.Printf("Verbose: %t\n", ctx.Config.Verbose)
			
			if ctx.Config.Current != "" {
				c.Printf("Current Profile: %s\n", ctx.Config.Current)
			}
			
			if len(ctx.Config.Profiles) > 0 {
				c.Printf("\nAvailable Profiles:\n")
				for name, profile := range ctx.Config.Profiles {
					indicator := ""
					if name == ctx.Config.Current {
						indicator = color.GreenString(" (current)")
					}
					c.Printf("  %s: %s%s\n", name, profile.Server, indicator)
				}
			}
		},
	})

	// config server
	configCmd.AddCmd(&ishell.Cmd{
		Name: "server",
		Help: "Set server URL",
		Func: func(c *ishell.Context) {
			if !ctx.RequireArgs(c, 1, "config server <url>") {
				return
			}
			
			ctx.Config.Server = c.Args[0]
			ctx.UpdateConfig()
			ctx.PrintSuccess(c, fmt.Sprintf("Server set to: %s", ctx.Config.Server))
			
			// Save configuration
			if err := ctx.Config.Save(); err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to save config: %v", err))
			}
		},
	})

	// config apikey
	configCmd.AddCmd(&ishell.Cmd{
		Name: "apikey",
		Help: "Set API key",
		Func: func(c *ishell.Context) {
			if !ctx.RequireArgs(c, 1, "config apikey <key>") {
				return
			}
			
			ctx.Config.APIKey = c.Args[0]
			ctx.UpdateConfig()
			ctx.PrintSuccess(c, "API key configured")
			
			// Save configuration
			if err := ctx.Config.Save(); err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to save config: %v", err))
			}
		},
	})

	// config output
	configCmd.AddCmd(&ishell.Cmd{
		Name: "output",
		Help: "Set output format (table, json, yaml)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireArgs(c, 1, "config output <format>") {
				return
			}
			
			format := strings.ToLower(c.Args[0])
			if format != "table" && format != "json" && format != "yaml" {
				ctx.PrintError(c, fmt.Errorf("invalid output format. Use: table, json, or yaml"))
				return
			}
			
			ctx.Config.Output = format
			ctx.UpdateConfig()
			ctx.PrintSuccess(c, fmt.Sprintf("Output format set to: %s", format))
			
			// Save configuration
			if err := ctx.Config.Save(); err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to save config: %v", err))
			}
		},
	})

	// config colors
	configCmd.AddCmd(&ishell.Cmd{
		Name: "colors",
		Help: "Enable or disable colored output (true/false)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireArgs(c, 1, "config colors <true|false>") {
				return
			}
			
			value := strings.ToLower(c.Args[0])
			if value == "true" || value == "on" || value == "yes" {
				ctx.Config.NoColor = false
				ctx.PrintSuccess(c, "Colors enabled")
			} else if value == "false" || value == "off" || value == "no" {
				ctx.Config.NoColor = true
				ctx.PrintSuccess(c, "Colors disabled")
			} else {
				ctx.PrintError(c, fmt.Errorf("invalid value. Use: true, false, on, off, yes, or no"))
				return
			}
			
			ctx.UpdateConfig()
			
			// Save configuration
			if err := ctx.Config.Save(); err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to save config: %v", err))
			}
		},
	})

	// config profile commands
	profileCmd := &ishell.Cmd{
		Name: "profile",
		Help: "Manage configuration profiles",
	}

	// config profile list
	profileCmd.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "List all configuration profiles",
		Func: func(c *ishell.Context) {
			if len(ctx.Config.Profiles) == 0 {
				c.Printf("No profiles configured\n")
				return
			}
			
			c.Printf("=== Configuration Profiles ===\n")
			for name, profile := range ctx.Config.Profiles {
				indicator := ""
				if name == ctx.Config.Current {
					indicator = color.GreenString(" (current)")
				}
				c.Printf("  %s%s\n", color.CyanString(name), indicator)
				c.Printf("    Server: %s\n", profile.Server)
				c.Printf("    Description: %s\n", valueOrEmpty(profile.Description))
				c.Printf("\n")
			}
		},
	})

	// config profile use
	profileCmd.AddCmd(&ishell.Cmd{
		Name: "use",
		Help: "Switch to a configuration profile",
		Func: func(c *ishell.Context) {
			if !ctx.RequireArgs(c, 1, "config profile use <name>") {
				return
			}
			
			profileName := c.Args[0]
			profile, exists := ctx.Config.Profiles[profileName]
			if !exists {
				ctx.PrintError(c, fmt.Errorf("profile '%s' not found", profileName))
				return
			}
			
			// Switch to profile
			ctx.Config.Current = profileName
			ctx.Config.Server = profile.Server
			ctx.Config.APIKey = profile.APIKey
			ctx.UpdateConfig()
			
			ctx.PrintSuccess(c, fmt.Sprintf("Switched to profile: %s", profileName))
			
			// Save configuration
			if err := ctx.Config.Save(); err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to save config: %v", err))
			}
		},
	})

	// config profile add
	profileCmd.AddCmd(&ishell.Cmd{
		Name: "add",
		Help: "Add a new configuration profile",
		Func: func(c *ishell.Context) {
			if !ctx.RequireArgs(c, 1, "config profile add <name>") {
				return
			}
			
			profileName := c.Args[0]
			
			// Check if profile already exists
			if _, exists := ctx.Config.Profiles[profileName]; exists {
				ctx.PrintError(c, fmt.Errorf("profile '%s' already exists", profileName))
				return
			}
			
			// Get server URL
			c.Print("Server URL: ")
			server := c.ReadLine()
			if server == "" {
				ctx.PrintError(c, fmt.Errorf("server URL is required"))
				return
			}
			
			// Get API key
			c.Print("API Key: ")
			apiKey := c.ReadLine()
			if apiKey == "" {
				ctx.PrintError(c, fmt.Errorf("API key is required"))
				return
			}
			
			// Get optional description
			c.Print("Description (optional): ")
			description := c.ReadLine()
			
			// Initialize profiles map if nil
			if ctx.Config.Profiles == nil {
				ctx.Config.Profiles = make(map[string]*config.Profile)
			}
			
			// Add profile
			ctx.Config.Profiles[profileName] = &config.Profile{
				Name:        profileName,
				Server:      server,
				APIKey:      apiKey,
				Description: description,
			}
			
			ctx.PrintSuccess(c, fmt.Sprintf("Profile '%s' added successfully", profileName))
			
			// Save configuration
			if err := ctx.Config.Save(); err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to save config: %v", err))
			}
		},
	})

	// config profile remove
	profileCmd.AddCmd(&ishell.Cmd{
		Name: "remove",
		Help: "Remove a configuration profile",
		Func: func(c *ishell.Context) {
			if !ctx.RequireArgs(c, 1, "config profile remove <name>") {
				return
			}
			
			profileName := c.Args[0]
			
			if _, exists := ctx.Config.Profiles[profileName]; !exists {
				ctx.PrintError(c, fmt.Errorf("profile '%s' not found", profileName))
				return
			}
			
			// Check if trying to remove current profile
			if profileName == ctx.Config.Current {
				ctx.PrintError(c, fmt.Errorf("cannot remove current profile. Switch to another profile first"))
				return
			}
			
			delete(ctx.Config.Profiles, profileName)
			ctx.PrintSuccess(c, fmt.Sprintf("Profile '%s' removed", profileName))
			
			// Save configuration
			if err := ctx.Config.Save(); err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to save config: %v", err))
			}
		},
	})

	configCmd.AddCmd(profileCmd)
	sh.AddCmd(configCmd)
}

func valueOrEmpty(s string) string {
	if s == "" {
		return color.RedString("(not set)")
	}
	return s
}

func maskAPIKey(key string) string {
	if key == "" {
		return color.RedString("(not set)")
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
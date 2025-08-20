package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"secauto-cli/pkg/config"
	"secauto-cli/pkg/output"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long:  `Manage CLI configuration including profiles, server settings, and authentication.`,
}

// configShowCmd represents the config show command
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current CLI configuration including active profile and settings.`,
	RunE:  configShow,
}

// configSetCmd represents the config set command
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value. Available keys:
  server     - SecAuto server URL
  api-key    - API key for authentication
  output     - Output format (table, json, yaml)
  no-color   - Disable colored output (true/false)
  verbose    - Enable verbose output (true/false)`,
	Args: cobra.ExactArgs(2),
	RunE: configSet,
}

// configProfileCmd represents the config profile command
var configProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage configuration profiles",
	Long:  `Create, list, and manage configuration profiles for different SecAuto environments.`,
}

// configProfileListCmd represents the config profile list command
var configProfileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Long:  `List all available configuration profiles.`,
	RunE:  configProfileList,
}

// configProfileAddCmd represents the config profile add command
var configProfileAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new profile",
	Long:  `Add a new configuration profile.`,
	Args:  cobra.ExactArgs(1),
	RunE:  configProfileAdd,
}

// configProfileRemoveCmd represents the config profile remove command
var configProfileRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a profile",
	Long:  `Remove a configuration profile.`,
	Args:  cobra.ExactArgs(1),
	RunE:  configProfileRemove,
}

// configProfileUseCmd represents the config profile use command
var configProfileUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch to a profile",
	Long:  `Switch to use a specific configuration profile.`,
	Args:  cobra.ExactArgs(1),
	RunE:  configProfileUse,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configProfileCmd)
	
	configProfileCmd.AddCommand(configProfileListCmd)
	configProfileCmd.AddCommand(configProfileAddCmd)
	configProfileCmd.AddCommand(configProfileRemoveCmd)
	configProfileCmd.AddCommand(configProfileUseCmd)

	// Profile add flags
	configProfileAddCmd.Flags().String("server", "", "Server URL for the profile")
	configProfileAddCmd.Flags().String("api-key", "", "API key for the profile")
	configProfileAddCmd.Flags().String("description", "", "Description for the profile")
}

func configShow(cmd *cobra.Command, args []string) error {
	cfg := GetGlobalConfig()

	fmt.Printf("Current Configuration:\n")
	fmt.Printf("  Server: %s\n", cfg.Server)
	fmt.Printf("  API Key: %s\n", maskAPIKey(cfg.APIKey))
	fmt.Printf("  Output Format: %s\n", cfg.Output)
	fmt.Printf("  No Color: %t\n", cfg.NoColor)
	fmt.Printf("  Verbose: %t\n", cfg.Verbose)
	fmt.Printf("  Current Profile: %s\n", cfg.Current)

	if len(cfg.Profiles) > 0 {
		fmt.Printf("\nAvailable Profiles:\n")
		for name, profile := range cfg.Profiles {
			marker := ""
			if name == cfg.Current {
				marker = " (current)"
			}
			fmt.Printf("  %s%s\n", name, marker)
			if profile.Description != "" {
				fmt.Printf("    Description: %s\n", profile.Description)
			}
			fmt.Printf("    Server: %s\n", profile.Server)
			fmt.Printf("    API Key: %s\n", maskAPIKey(profile.APIKey))
		}
	}

	return nil
}

func configSet(cmd *cobra.Command, args []string) error {
	cfg := GetGlobalConfig()
	printer := output.NewPrinter(cfg.Output, cfg.NoColor)

	key := args[0]
	value := args[1]

	switch key {
	case "server":
		cfg.Server = value
	case "api-key":
		cfg.APIKey = value
	case "output":
		if value != "table" && value != "json" && value != "yaml" {
			return fmt.Errorf("invalid output format: %s (must be table, json, or yaml)", value)
		}
		cfg.Output = value
	case "no-color":
		if value == "true" {
			cfg.NoColor = true
		} else if value == "false" {
			cfg.NoColor = false
		} else {
			return fmt.Errorf("invalid boolean value: %s (must be true or false)", value)
		}
	case "verbose":
		if value == "true" {
			cfg.Verbose = true
		} else if value == "false" {
			cfg.Verbose = false
		} else {
			return fmt.Errorf("invalid boolean value: %s (must be true or false)", value)
		}
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	printer.PrintSuccess(fmt.Sprintf("Configuration '%s' set to '%s'", key, value))
	return nil
}

func configProfileList(cmd *cobra.Command, args []string) error {
	cfg := GetGlobalConfig()
	printer := output.NewPrinter(cfg.Output, cfg.NoColor)

	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles configured")
		return nil
	}

	// Convert to format suitable for table output
	var data []map[string]interface{}
	for name, profile := range cfg.Profiles {
		current := ""
		if name == cfg.Current {
			current = "✓"
		}
		
		data = append(data, map[string]interface{}{
			"Current":     current,
			"Name":        name,
			"Server":      profile.Server,
			"API Key":     maskAPIKey(profile.APIKey),
			"Description": profile.Description,
		})
	}

	return printer.Print(data)
}

func configProfileAdd(cmd *cobra.Command, args []string) error {
	cfg := GetGlobalConfig()
	printer := output.NewPrinter(cfg.Output, cfg.NoColor)

	name := args[0]
	server, _ := cmd.Flags().GetString("server")
	apiKey, _ := cmd.Flags().GetString("api-key")
	description, _ := cmd.Flags().GetString("description")

	// Prompt for missing values
	if server == "" {
		fmt.Print("Server URL: ")
		fmt.Scanln(&server)
	}
	
	if apiKey == "" {
		fmt.Print("API Key: ")
		fmt.Scanln(&apiKey)
	}

	if description == "" {
		fmt.Print("Description (optional): ")
		fmt.Scanln(&description)
	}

	profile := config.Profile{
		Name:        name,
		Server:      server,
		APIKey:      apiKey,
		Description: description,
	}

	cfg.AddProfile(name, &profile)
	
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	printer.PrintSuccess(fmt.Sprintf("Profile '%s' added successfully", name))
	return nil
}

func configProfileRemove(cmd *cobra.Command, args []string) error {
	cfg := GetGlobalConfig()
	printer := output.NewPrinter(cfg.Output, cfg.NoColor)

	name := args[0]

	if err := cfg.RemoveProfile(name); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	printer.PrintSuccess(fmt.Sprintf("Profile '%s' removed successfully", name))
	return nil
}

func configProfileUse(cmd *cobra.Command, args []string) error {
	cfg := GetGlobalConfig()
	printer := output.NewPrinter(cfg.Output, cfg.NoColor)

	name := args[0]

	if err := cfg.SetCurrentProfile(name); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	printer.PrintSuccess(fmt.Sprintf("Switched to profile '%s'", name))
	return nil
}

// maskAPIKey masks an API key for display
func maskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "(not set)"
	}
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}
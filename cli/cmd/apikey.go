package cmd

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"secauto-cli/pkg/client"
	"secauto-cli/pkg/output"
)

// apikeyCmd represents the apikey command
var apikeyCmd = &cobra.Command{
	Use:   "apikey",
	Short: "Manage API keys",
	Long:  `Create, list, and delete API keys for SecAuto platform access.`,
}

// apikeyListCmd represents the apikey list command
var apikeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all API keys",
	Long:  `List all API keys configured in the SecAuto platform.`,
	RunE:  listAPIKeys,
}

// apikeyCreateCmd represents the apikey create command
var apikeyCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new API key",
	Long:  `Create a new API key with optional name/description.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  createAPIKey,
}

// apikeyDeleteCmd represents the apikey delete command
var apikeyDeleteCmd = &cobra.Command{
	Use:   "delete <key-id>",
	Short: "Delete an API key",
	Long:  `Delete an API key from the SecAuto platform.`,
	Args:  cobra.ExactArgs(1),
	RunE:  deleteAPIKey,
}

// apikeyStatsCmd represents the apikey stats command
var apikeyStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show API key usage statistics",
	Long:  `Show usage statistics for API keys.`,
	RunE:  getAPIKeyStats,
}

func init() {
	rootCmd.AddCommand(apikeyCmd)
	apikeyCmd.AddCommand(apikeyListCmd)
	apikeyCmd.AddCommand(apikeyCreateCmd)
	apikeyCmd.AddCommand(apikeyDeleteCmd)
	apikeyCmd.AddCommand(apikeyStatsCmd)

	// Create command flags
	apikeyCreateCmd.Flags().String("description", "", "Description for the API key")
	apikeyCreateCmd.Flags().StringSlice("permissions", []string{}, "Permissions for the API key")
}

func listAPIKeys(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	// Check server health
	if err := apiClient.HealthCheck(); err != nil {
		return fmt.Errorf("server health check failed: %v", err)
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Fetching API keys..."
	}

	apiKeys, err := apiClient.ListAPIKeys()

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to list API keys: %v", err)
	}

	if len(apiKeys) == 0 {
		fmt.Println("No API keys found")
		return nil
	}

	return printer.Print(apiKeys)
}

func createAPIKey(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	var name string
	if len(args) > 0 {
		name = args[0]
	}

	description, _ := cmd.Flags().GetString("description")
	permissions, _ := cmd.Flags().GetStringSlice("permissions")

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Creating API key..."
	}

	apiKeyInfo, err := apiClient.CreateAPIKey(name, description, permissions)

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to create API key: %v", err)
	}

	printer.PrintSuccess("API key created successfully")
	return printer.Print(apiKeyInfo)
}

func deleteAPIKey(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	keyID := args[0]

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Deleting API key..."
	}

	err := apiClient.DeleteAPIKey(keyID)

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("delete failed: %v", err)
	}

	printer.PrintSuccess(fmt.Sprintf("API key '%s' deleted successfully", keyID))
	return nil
}

func getAPIKeyStats(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Fetching API key statistics..."
	}

	stats, err := apiClient.GetAPIKeyStats()

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to get API key stats: %v", err)
	}

	return printer.Print(stats)
}
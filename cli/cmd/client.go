package cmd

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"secauto-cli/pkg/client"
	"secauto-cli/pkg/output"
)

// clientCmd represents the client command
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Manage SecAuto clients",
	Long:  `List and manage SecAuto clients connected to the platform.`,
}

// clientListCmd represents the client list command
var clientListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all clients",
	Long:  `List all clients connected to the SecAuto platform.`,
	RunE:  listClients,
}

// clientGetCmd represents the client get command
var clientGetCmd = &cobra.Command{
	Use:   "get <client-id>",
	Short: "Get client details",
	Long:  `Get detailed information about a specific client.`,
	Args:  cobra.ExactArgs(1),
	RunE:  getClient,
}

// clientDeleteCmd represents the client delete command
var clientDeleteCmd = &cobra.Command{
	Use:   "delete <client-id>",
	Short: "Delete a client",
	Long:  `Delete a client from the SecAuto platform.`,
	Args:  cobra.ExactArgs(1),
	RunE:  deleteClient,
}

func init() {
	rootCmd.AddCommand(clientCmd)
	clientCmd.AddCommand(clientListCmd)
	clientCmd.AddCommand(clientGetCmd)
	clientCmd.AddCommand(clientDeleteCmd)
}

func listClients(cmd *cobra.Command, args []string) error {
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
		s.Suffix = " Fetching clients..."
	}

	clients, err := apiClient.ListClients()

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to list clients: %v", err)
	}

	if len(clients) == 0 {
		fmt.Println("No clients found")
		return nil
	}

	return printer.Print(clients)
}

func getClient(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	clientID := args[0]

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Fetching client details..."
	}

	clientInfo, err := apiClient.GetClient(clientID)

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to get client '%s': %v", clientID, err)
	}

	return printer.Print(clientInfo)
}

func deleteClient(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	clientID := args[0]

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Deleting client..."
	}

	err := apiClient.DeleteClient(clientID)

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("delete failed: %v", err)
	}

	printer.PrintSuccess(fmt.Sprintf("Client '%s' deleted successfully", clientID))
	return nil
}
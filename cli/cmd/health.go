package cmd

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"secauto-cli/pkg/client"
	"secauto-cli/pkg/output"
)

// healthCmd represents the health command
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check server health",
	Long:  `Check the health status of the SecAuto server.`,
	RunE:  healthCheck,
}

func init() {
	rootCmd.AddCommand(healthCmd)
}

func healthCheck(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Checking server health..."
	}

	start := time.Now()
	err := apiClient.HealthCheck()
	duration := time.Since(start)

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		printer.PrintError(fmt.Sprintf("Health check failed: %v", err))
		return fmt.Errorf("server is unhealthy")
	}

	printer.PrintSuccess(fmt.Sprintf("Server is healthy (response time: %s)", output.FormatDuration(duration)))
	
	fmt.Printf("\nServer Details:\n")
	fmt.Printf("  URL: %s\n", config.Server)
	fmt.Printf("  Response Time: %s\n", output.FormatDuration(duration))
	fmt.Printf("  Status: %s\n", printer.FormatStatus("healthy"))

	return nil
}
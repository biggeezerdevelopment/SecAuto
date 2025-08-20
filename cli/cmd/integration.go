package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"secauto-cli/pkg/client"
	"secauto-cli/pkg/output"
)

// integrationCmd represents the integration command
var integrationCmd = &cobra.Command{
	Use:   "integration",
	Short: "Manage integrations",
	Long:  `List and execute integrations with external services.`,
}

// integrationListCmd represents the integration list command
var integrationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available integrations",
	Long:  `List all available integrations that can be executed.`,
	RunE:  listIntegrations,
}

// integrationExecuteCmd represents the integration execute command
var integrationExecuteCmd = &cobra.Command{
	Use:   "execute <integration-name>",
	Short: "Execute an integration",
	Long: `Execute a specific integration with optional parameters.
	
Examples:
  secauto integration execute virustotal --params '{"url": "https://example.com"}'
  secauto integration execute qualys --params-file params.json`,
	Args: cobra.ExactArgs(1),
	RunE: executeIntegration,
}

// automationCmd represents the automation command
var automationCmd = &cobra.Command{
	Use:   "automation",
	Short: "Manage automations",
	Long:  `List available Python automation scripts.`,
}

// automationListCmd represents the automation list command
var automationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available automations",
	Long:  `List all available Python automation scripts.`,
	RunE:  listAutomations,
}

// automationUploadCmd represents the automation upload command
var automationUploadCmd = &cobra.Command{
	Use:   "upload <file>",
	Short: "Upload an automation script",
	Long: `Upload a Python automation script to the SecAuto server.
	
Examples:
  secauto automation upload my_automation.py`,
	Args: cobra.ExactArgs(1),
	RunE: uploadAutomation,
}

// automationDeleteCmd represents the automation delete command
var automationDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an automation script",
	Long: `Delete an automation script from the SecAuto server.
	
Examples:
  secauto automation delete my_automation`,
	Args: cobra.ExactArgs(1),
	RunE: deleteAutomation,
}

func init() {
	rootCmd.AddCommand(integrationCmd)
	rootCmd.AddCommand(automationCmd)
	
	integrationCmd.AddCommand(integrationListCmd)
	integrationCmd.AddCommand(integrationExecuteCmd)
	
	automationCmd.AddCommand(automationListCmd)
	automationCmd.AddCommand(automationUploadCmd)
	automationCmd.AddCommand(automationDeleteCmd)

	// Integration execute flags
	integrationExecuteCmd.Flags().String("params", "", "Parameters as JSON string")
	integrationExecuteCmd.Flags().String("params-file", "", "Parameters from JSON file")
}

func listIntegrations(cmd *cobra.Command, args []string) error {
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
		s.Suffix = " Fetching integrations..."
	}

	integrations, err := apiClient.ListIntegrations()
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to list integrations: %v", err)
	}

	if len(integrations) == 0 {
		fmt.Println("No integrations found")
		return nil
	}

	// Convert to format suitable for table output
	var data []map[string]interface{}
	for i, integration := range integrations {
		data = append(data, map[string]interface{}{
			"#":    i + 1,
			"Name": integration,
		})
	}

	return printer.Print(data)
}

func executeIntegration(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	integrationName := args[0]

	// Parse parameters
	params := make(map[string]interface{})
	
	paramsStr, _ := cmd.Flags().GetString("params")
	paramsFile, _ := cmd.Flags().GetString("params-file")

	if paramsStr != "" && paramsFile != "" {
		return fmt.Errorf("cannot specify both --params and --params-file")
	}

	if paramsStr != "" {
		if err := json.Unmarshal([]byte(paramsStr), &params); err != nil {
			return fmt.Errorf("failed to parse parameters JSON: %v", err)
		}
	} else if paramsFile != "" {
		file, err := os.Open(paramsFile)
		if err != nil {
			return fmt.Errorf("failed to open parameters file: %v", err)
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("failed to read parameters file: %v", err)
		}

		if err := json.Unmarshal(data, &params); err != nil {
			return fmt.Errorf("failed to parse parameters file JSON: %v", err)
		}
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Executing integration..."
	}

	result, err := apiClient.ExecuteIntegration(integrationName, params)
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to execute integration '%s': %v", integrationName, err)
	}

	printer.PrintSuccess(fmt.Sprintf("Integration '%s' executed successfully", integrationName))
	
	fmt.Println("\nResults:")
	return printer.Print(result)
}

func listAutomations(cmd *cobra.Command, args []string) error {
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
		s.Suffix = " Fetching automations..."
	}

	automations, err := apiClient.ListAutomations()
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to list automations: %v", err)
	}

	if len(automations) == 0 {
		fmt.Println("No automations found")
		return nil
	}

	// Convert to format suitable for table output
	var data []map[string]interface{}
	for i, automation := range automations {
		data = append(data, map[string]interface{}{
			"#":    i + 1,
			"Name": automation,
		})
	}

	return printer.Print(data)
}

func uploadAutomation(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	automationFile := args[0]

	// Read file content
	file, err := os.Open(automationFile)
	if err != nil {
		return fmt.Errorf("failed to open automation file: %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read automation file: %v", err)
	}

	// Get the base filename
	filename := automationFile
	if idx := strings.LastIndex(automationFile, "/"); idx != -1 {
		filename = automationFile[idx+1:]
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Uploading automation..."
	}

	err = apiClient.UploadAutomation(filename, content)
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("upload failed: %v", err)
	}

	printer.PrintSuccess(fmt.Sprintf("Automation '%s' uploaded successfully", automationFile))
	return nil
}

func deleteAutomation(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	automationName := args[0]

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Deleting automation..."
	}

	err := apiClient.DeleteAutomation(automationName)
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("delete failed: %v", err)
	}

	printer.PrintSuccess(fmt.Sprintf("Automation '%s' deleted successfully", automationName))
	return nil
}
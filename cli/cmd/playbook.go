package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"secauto-cli/pkg/client"
	"secauto-cli/pkg/output"
)

// playbookCmd represents the playbook command
var playbookCmd = &cobra.Command{
	Use:   "playbook",
	Short: "Manage and execute playbooks",
	Long:  `Execute, upload, and manage SecAuto playbooks.`,
}

// executeCmd represents the execute command
var executeCmd = &cobra.Command{
	Use:   "execute [playbook-name-or-file]",
	Short: "Execute a playbook",
	Long: `Execute a playbook either by name (from uploaded playbooks) or from a file.
	
Examples:
  secauto playbook execute my-playbook.json
  secauto playbook execute --name uploaded-playbook
  secauto playbook execute --async my-playbook.json`,
	Args: cobra.MaximumNArgs(1),
	RunE: executePlaybook,
}

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload <file> [name]",
	Short: "Upload a playbook",
	Long: `Upload a playbook file to the SecAuto server.
	
Examples:
  secauto playbook upload my-playbook.json
  secauto playbook upload my-playbook.json custom-name`,
	Args: cobra.RangeArgs(1, 2),
	RunE: uploadPlaybook,
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available playbooks",
	Long:  `List all playbooks available on the SecAuto server.`,
	RunE:  listPlaybooks,
}

func init() {
	rootCmd.AddCommand(playbookCmd)
	playbookCmd.AddCommand(executeCmd)
	playbookCmd.AddCommand(uploadCmd)
	playbookCmd.AddCommand(listCmd)

	// Execute command flags
	executeCmd.Flags().String("name", "", "Name of uploaded playbook to execute")
	executeCmd.Flags().Bool("async", false, "Execute playbook asynchronously")
	executeCmd.Flags().String("context", "", "Context JSON string or file path")
	executeCmd.Flags().Bool("watch", false, "Watch job status until completion (async mode only)")
	executeCmd.Flags().Duration("timeout", 5*time.Minute, "Timeout for synchronous execution")
}

func executePlaybook(cmd *cobra.Command, args []string) error {
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

	// Parse flags
	playbookName, _ := cmd.Flags().GetString("name")
	async, _ := cmd.Flags().GetBool("async")
	contextFlag, _ := cmd.Flags().GetString("context")
	watch, _ := cmd.Flags().GetBool("watch")

	// Determine playbook source
	var playbook interface{}
	var err error

	if playbookName != "" {
		// Use named playbook
		playbook = playbookName
	} else if len(args) > 0 {
		// Load from file
		playbookFile := args[0]
		playbook, err = loadPlaybookFromFile(playbookFile)
		if err != nil {
			return fmt.Errorf("failed to load playbook: %v", err)
		}
	} else {
		return fmt.Errorf("must specify either --name flag or playbook file")
	}

	// Parse context
	context := make(map[string]interface{})
	if contextFlag != "" {
		context, err = parseContext(contextFlag)
		if err != nil {
			return fmt.Errorf("failed to parse context: %v", err)
		}
	}

	// Prepare request
	req := &client.PlaybookRequest{
		Context: context,
	}

	if playbookName != "" {
		req.PlaybookName = playbookName
	} else {
		req.Playbook = playbook
	}

	// Execute playbook
	if async {
		return executePlaybookAsync(apiClient, printer, req, watch)
	} else {
		return executePlaybookSync(apiClient, printer, req)
	}
}

func executePlaybookSync(apiClient *client.Client, printer *output.Printer, req *client.PlaybookRequest) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Executing playbook..."
	}

	resp, err := apiClient.ExecutePlaybook(req)
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("execution failed: %v", err)
	}

	if !resp.Success {
		printer.PrintError(fmt.Sprintf("Playbook execution failed: %s", resp.Error))
		return fmt.Errorf("playbook execution failed")
	}

	printer.PrintSuccess("Playbook executed successfully")
	
	if resp.Results != nil {
		fmt.Println("\nResults:")
		return printer.Print(resp.Results)
	}

	return nil
}

func executePlaybookAsync(apiClient *client.Client, printer *output.Printer, req *client.PlaybookRequest, watch bool) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Starting playbook execution..."
	}

	resp, err := apiClient.ExecutePlaybookAsync(req)
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("execution failed: %v", err)
	}

	if !resp.Success {
		printer.PrintError(fmt.Sprintf("Playbook execution failed: %s", resp.Error))
		return fmt.Errorf("playbook execution failed")
	}

	printer.PrintSuccess(fmt.Sprintf("Playbook execution started. Job ID: %s", resp.JobID))

	if watch && resp.JobID != "" {
		return watchJob(apiClient, printer, resp.JobID)
	}

	fmt.Printf("\nUse 'secauto job get %s' to check status\n", resp.JobID)
	return nil
}

func watchJob(apiClient *client.Client, printer *output.Printer, jobID string) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Watching job..."
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			job, err := apiClient.GetJob(jobID)
			if err != nil {
				if !printer.NoColor {
					s.Stop()
				}
				return fmt.Errorf("failed to get job status: %v", err)
			}

			switch job.Status {
			case "completed":
				if !printer.NoColor {
					s.Stop()
				}
				printer.PrintSuccess("Job completed successfully")
				if job.Results != nil {
					fmt.Println("\nResults:")
					return printer.Print(job.Results)
				}
				return nil
			case "failed":
				if !printer.NoColor {
					s.Stop()
				}
				printer.PrintError(fmt.Sprintf("Job failed: %s", job.Error))
				return fmt.Errorf("job failed")
			}
		}
	}
}

func uploadPlaybook(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	playbookFile := args[0]
	var name string
	if len(args) > 1 {
		name = args[1]
	} else {
		// Use filename without extension as name
		name = strings.TrimSuffix(filepath.Base(playbookFile), filepath.Ext(playbookFile))
	}

	// Load playbook from file
	playbook, err := loadPlaybookFromFile(playbookFile)
	if err != nil {
		return fmt.Errorf("failed to load playbook: %v", err)
	}

	// Upload playbook
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Uploading playbook..."
	}

	err = apiClient.UploadPlaybook(name, playbook)
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("upload failed: %v", err)
	}

	printer.PrintSuccess(fmt.Sprintf("Playbook '%s' uploaded successfully", name))
	return nil
}

func listPlaybooks(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	playbooks, err := apiClient.ListPlaybooks()
	if err != nil {
		return fmt.Errorf("failed to list playbooks: %v", err)
	}

	if len(playbooks) == 0 {
		fmt.Println("No playbooks found")
		return nil
	}

	// Convert to format suitable for table output
	var data []map[string]interface{}
	for i, playbook := range playbooks {
		data = append(data, map[string]interface{}{
			"#":    i + 1,
			"Name": playbook,
		})
	}

	return printer.Print(data)
}

func loadPlaybookFromFile(filename string) (interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var playbook interface{}
	if err := json.Unmarshal(data, &playbook); err != nil {
		return nil, err
	}

	return playbook, nil
}

func parseContext(contextFlag string) (map[string]interface{}, error) {
	context := make(map[string]interface{})

	// Check if it's a file path
	if _, err := os.Stat(contextFlag); err == nil {
		// It's a file
		file, err := os.Open(contextFlag)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(data, &context); err != nil {
			return nil, err
		}
	} else {
		// It's a JSON string
		if err := json.Unmarshal([]byte(contextFlag), &context); err != nil {
			return nil, err
		}
	}

	return context, nil
}
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"secauto-cli/pkg/client"
	"secauto-cli/pkg/output"
)

// batchCmd represents the batch command
var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Execute batch operations",
	Long:  `Execute multiple operations in batch mode for efficient processing.`,
}

// batchPlaybookCmd represents the batch playbook command
var batchPlaybookCmd = &cobra.Command{
	Use:   "playbook <directory>",
	Short: "Execute multiple playbooks from a directory",
	Long: `Execute all playbooks found in a directory.
	
Examples:
  secauto batch playbook ./playbooks/
  secauto batch playbook ./playbooks/ --async --max-concurrent 5`,
	Args: cobra.ExactArgs(1),
	RunE: batchExecutePlaybooks,
}

// batchUploadCmd represents the batch upload command
var batchUploadCmd = &cobra.Command{
	Use:   "upload <directory>",
	Short: "Upload multiple files from a directory",
	Long: `Upload multiple playbooks or automations from a directory.
	
Examples:
  secauto batch upload ./playbooks/ --type playbook
  secauto batch upload ./automations/ --type automation`,
	Args: cobra.ExactArgs(1),
	RunE: batchUpload,
}

func init() {
	rootCmd.AddCommand(batchCmd)
	batchCmd.AddCommand(batchPlaybookCmd)
	batchCmd.AddCommand(batchUploadCmd)

	// Batch playbook flags
	batchPlaybookCmd.Flags().Bool("async", false, "Execute playbooks asynchronously")
	batchPlaybookCmd.Flags().Int("max-concurrent", 3, "Maximum concurrent executions")
	batchPlaybookCmd.Flags().String("context", "", "Context JSON string or file path to use for all playbooks")
	batchPlaybookCmd.Flags().Bool("continue-on-error", false, "Continue processing even if some playbooks fail")

	// Batch upload flags
	batchUploadCmd.Flags().String("type", "playbook", "Type of files to upload (playbook, automation)")
	batchUploadCmd.Flags().String("pattern", "*.json", "File pattern to match (for playbooks: *.json, for automations: *.py)")
	batchUploadCmd.Flags().Bool("continue-on-error", false, "Continue processing even if some uploads fail")
}

func batchExecutePlaybooks(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	directory := args[0]

	// Check server health
	if err := apiClient.HealthCheck(); err != nil {
		return fmt.Errorf("server health check failed: %v", err)
	}

	// Parse flags
	async, _ := cmd.Flags().GetBool("async")
	maxConcurrent, _ := cmd.Flags().GetInt("max-concurrent")
	contextFlag, _ := cmd.Flags().GetString("context")
	continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

	// Parse context if provided
	var context map[string]interface{}
	if contextFlag != "" {
		var err error
		context, err = parseContext(contextFlag)
		if err != nil {
			return fmt.Errorf("failed to parse context: %v", err)
		}
	}

	// Find all JSON files in directory
	files, err := filepath.Glob(filepath.Join(directory, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to find playbook files: %v", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no playbook files found in directory: %s", directory)
	}

	printer.PrintInfo(fmt.Sprintf("Found %d playbook files", len(files)))

	if async {
		return executeBatchAsync(apiClient, printer, files, context, maxConcurrent, continueOnError)
	} else {
		return executeBatchSync(apiClient, printer, files, context, continueOnError)
	}
}

func executeBatchSync(apiClient *client.Client, printer *output.Printer, files []string, context map[string]interface{}, continueOnError bool) error {
	var results []map[string]interface{}
	var errors []string

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	
	for i, file := range files {
		filename := filepath.Base(file)
		
		if !printer.NoColor {
			s.Start()
			s.Suffix = fmt.Sprintf(" Executing %s (%d/%d)...", filename, i+1, len(files))
		}

		// Load playbook
		playbook, err := loadPlaybookFromFile(file)
		if err != nil {
			if !printer.NoColor {
				s.Stop()
			}
			errorMsg := fmt.Sprintf("Failed to load %s: %v", filename, err)
			printer.PrintError(errorMsg)
			errors = append(errors, errorMsg)
			
			if !continueOnError {
				return fmt.Errorf("batch execution stopped due to error")
			}
			continue
		}

		// Execute playbook
		req := &client.PlaybookRequest{
			Playbook: playbook,
			Context:  context,
		}

		resp, err := apiClient.ExecutePlaybook(req)
		
		if !printer.NoColor {
			s.Stop()
		}

		result := map[string]interface{}{
			"file":      filename,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		if err != nil {
			errorMsg := fmt.Sprintf("Failed to execute %s: %v", filename, err)
			printer.PrintError(errorMsg)
			result["status"] = "error"
			result["error"] = err.Error()
			errors = append(errors, errorMsg)
			
			if !continueOnError {
				return fmt.Errorf("batch execution stopped due to error")
			}
		} else if !resp.Success {
			errorMsg := fmt.Sprintf("Execution failed for %s: %s", filename, resp.Error)
			printer.PrintError(errorMsg)
			result["status"] = "failed"
			result["error"] = resp.Error
			errors = append(errors, errorMsg)
			
			if !continueOnError {
				return fmt.Errorf("batch execution stopped due to error")
			}
		} else {
			printer.PrintSuccess(fmt.Sprintf("Successfully executed %s", filename))
			result["status"] = "success"
			result["results"] = resp.Results
		}

		results = append(results, result)
	}

	// Print summary
	fmt.Printf("\nBatch Execution Summary:\n")
	fmt.Printf("Total files: %d\n", len(files))
	fmt.Printf("Successful: %d\n", len(files)-len(errors))
	fmt.Printf("Failed: %d\n", len(errors))

	if len(errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	return printer.Print(results)
}

func executeBatchAsync(apiClient *client.Client, printer *output.Printer, files []string, context map[string]interface{}, maxConcurrent int, continueOnError bool) error {
	var results []map[string]interface{}
	var mutex sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrent)

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Starting batch async execution..."
	}

	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			filename := filepath.Base(file)
			
			// Load playbook
			playbook, err := loadPlaybookFromFile(file)
			if err != nil {
				mutex.Lock()
				results = append(results, map[string]interface{}{
					"file":      filename,
					"status":    "error",
					"error":     err.Error(),
					"timestamp": time.Now().Format(time.RFC3339),
				})
				mutex.Unlock()
				return
			}

			// Execute playbook async
			req := &client.PlaybookRequest{
				Playbook: playbook,
				Context:  context,
			}

			resp, err := apiClient.ExecutePlaybookAsync(req)
			
			result := map[string]interface{}{
				"file":      filename,
				"timestamp": time.Now().Format(time.RFC3339),
			}

			if err != nil {
				result["status"] = "error"
				result["error"] = err.Error()
			} else if !resp.Success {
				result["status"] = "failed"
				result["error"] = resp.Error
			} else {
				result["status"] = "started"
				result["job_id"] = resp.JobID
			}

			mutex.Lock()
			results = append(results, result)
			mutex.Unlock()
		}(file)
	}

	wg.Wait()
	
	if !printer.NoColor {
		s.Stop()
	}

	printer.PrintSuccess(fmt.Sprintf("Started async execution for %d playbooks", len(files)))

	// Count successful starts
	successCount := 0
	for _, result := range results {
		if result["status"] == "started" {
			successCount++
		}
	}

	fmt.Printf("\nBatch Async Execution Summary:\n")
	fmt.Printf("Total files: %d\n", len(files))
	fmt.Printf("Successfully started: %d\n", successCount)
	fmt.Printf("Failed to start: %d\n", len(files)-successCount)

	return printer.Print(results)
}

func batchUpload(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	directory := args[0]
	uploadType, _ := cmd.Flags().GetString("type")
	pattern, _ := cmd.Flags().GetString("pattern")
	continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

	// Validate upload type
	if uploadType != "playbook" && uploadType != "automation" {
		return fmt.Errorf("invalid upload type: %s (must be 'playbook' or 'automation')", uploadType)
	}

	// Set default pattern based on type
	if pattern == "*.json" && uploadType == "automation" {
		pattern = "*.py"
	}

	// Find files
	files, err := filepath.Glob(filepath.Join(directory, pattern))
	if err != nil {
		return fmt.Errorf("failed to find files: %v", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found matching pattern: %s in directory: %s", pattern, directory)
	}

	printer.PrintInfo(fmt.Sprintf("Found %d %s files", len(files), uploadType))

	var results []map[string]interface{}
	var errors []string

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)

	for i, file := range files {
		filename := filepath.Base(file)
		name := strings.TrimSuffix(filename, filepath.Ext(filename))
		
		if !printer.NoColor {
			s.Start()
			s.Suffix = fmt.Sprintf(" Uploading %s (%d/%d)...", filename, i+1, len(files))
		}

		result := map[string]interface{}{
			"file":      filename,
			"name":      name,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		var err error
		if uploadType == "playbook" {
			// Load and upload playbook
			playbook, loadErr := loadPlaybookFromFile(file)
			if loadErr != nil {
				err = loadErr
			} else {
				err = apiClient.UploadPlaybook(name, playbook)
			}
		} else {
			// Read and upload automation
			content, readErr := os.ReadFile(file)
			if readErr != nil {
				err = readErr
			} else {
				err = apiClient.UploadAutomation(filename, content)
			}
		}

		if !printer.NoColor {
			s.Stop()
		}

		if err != nil {
			errorMsg := fmt.Sprintf("Failed to upload %s: %v", filename, err)
			printer.PrintError(errorMsg)
			result["status"] = "error"
			result["error"] = err.Error()
			errors = append(errors, errorMsg)
			
			if !continueOnError {
				return fmt.Errorf("batch upload stopped due to error")
			}
		} else {
			printer.PrintSuccess(fmt.Sprintf("Successfully uploaded %s", filename))
			result["status"] = "success"
		}

		results = append(results, result)
	}

	// Print summary
	fmt.Printf("\nBatch Upload Summary:\n")
	fmt.Printf("Total files: %d\n", len(files))
	fmt.Printf("Successful: %d\n", len(files)-len(errors))
	fmt.Printf("Failed: %d\n", len(errors))

	if len(errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	return printer.Print(results)
}
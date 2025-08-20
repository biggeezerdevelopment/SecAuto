package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"secauto-cli/pkg/client"
	"secauto-cli/pkg/output"
)

// scheduleCmd represents the schedule command
var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage job schedules",
	Long: `Create, list, update, delete, and execute scheduled jobs.
	
Schedules allow you to run playbooks automatically at specified times using cron expressions.`,
}

// scheduleListCmd represents the schedule list command
var scheduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all schedules",
	Long:  `List all configured schedules with their status and next run time.`,
	RunE:  listSchedules,
}

// scheduleGetCmd represents the schedule get command
var scheduleGetCmd = &cobra.Command{
	Use:   "get <schedule-id>",
	Short: "Get schedule details",
	Long:  `Get detailed information about a specific schedule.`,
	Args:  cobra.ExactArgs(1),
	RunE:  getSchedule,
}

// scheduleCreateCmd represents the schedule create command
var scheduleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new schedule",
	Long: `Create a new scheduled job with cron expression and playbook.
	
Example:
  secauto schedule create --name "Daily Scan" --cron "0 0 * * *" --playbook scan.json --enabled`,
	RunE: createSchedule,
}

// scheduleUpdateCmd represents the schedule update command
var scheduleUpdateCmd = &cobra.Command{
	Use:   "update <schedule-id>",
	Short: "Update an existing schedule",
	Long:  `Update an existing schedule's configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE:  updateSchedule,
}

// scheduleDeleteCmd represents the schedule delete command
var scheduleDeleteCmd = &cobra.Command{
	Use:   "delete <schedule-id>",
	Short: "Delete a schedule",
	Long:  `Delete an existing schedule.`,
	Args:  cobra.ExactArgs(1),
	RunE:  deleteSchedule,
}

// scheduleExecuteCmd represents the schedule execute command
var scheduleExecuteCmd = &cobra.Command{
	Use:   "execute <schedule-id>",
	Short: "Execute a schedule manually",
	Long:  `Manually trigger a scheduled job to run immediately.`,
	Args:  cobra.ExactArgs(1),
	RunE:  executeSchedule,
}

// scheduleStatsCmd represents the schedule stats command
var scheduleStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Get schedule statistics",
	Long:  `Display statistics about all schedules.`,
	RunE:  scheduleStats,
}

func init() {
	rootCmd.AddCommand(scheduleCmd)
	scheduleCmd.AddCommand(scheduleListCmd)
	scheduleCmd.AddCommand(scheduleGetCmd)
	scheduleCmd.AddCommand(scheduleCreateCmd)
	scheduleCmd.AddCommand(scheduleUpdateCmd)
	scheduleCmd.AddCommand(scheduleDeleteCmd)
	scheduleCmd.AddCommand(scheduleExecuteCmd)
	scheduleCmd.AddCommand(scheduleStatsCmd)

	// Schedule list flags
	scheduleListCmd.Flags().String("status", "", "Filter schedules by status (enabled, disabled)")

	// Schedule create flags
	scheduleCreateCmd.Flags().String("name", "", "Schedule name (required)")
	scheduleCreateCmd.Flags().String("description", "", "Schedule description")
	scheduleCreateCmd.Flags().String("cron", "", "Cron expression (required)")
	scheduleCreateCmd.Flags().String("playbook", "", "Playbook file path or name")
	scheduleCreateCmd.Flags().String("playbook-json", "", "Inline playbook JSON")
	scheduleCreateCmd.Flags().String("context", "{}", "Context JSON for the playbook")
	scheduleCreateCmd.Flags().Bool("enabled", false, "Enable schedule immediately")
	scheduleCreateCmd.MarkFlagRequired("name")
	scheduleCreateCmd.MarkFlagRequired("cron")

	// Schedule update flags
	scheduleUpdateCmd.Flags().String("name", "", "Update schedule name")
	scheduleUpdateCmd.Flags().String("description", "", "Update schedule description")
	scheduleUpdateCmd.Flags().String("cron", "", "Update cron expression")
	scheduleUpdateCmd.Flags().String("playbook", "", "Update playbook file path or name")
	scheduleUpdateCmd.Flags().String("playbook-json", "", "Update inline playbook JSON")
	scheduleUpdateCmd.Flags().String("context", "", "Update context JSON")
	scheduleUpdateCmd.Flags().Bool("enable", false, "Enable the schedule")
	scheduleUpdateCmd.Flags().Bool("disable", false, "Disable the schedule")

	// Schedule delete flags
	scheduleDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

func listSchedules(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	// Get filter flag
	statusFilter, _ := cmd.Flags().GetString("status")

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Fetching schedules..."
	}

	schedules, err := apiClient.ListSchedules(statusFilter)

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to list schedules: %v", err)
	}

	if len(schedules) == 0 {
		if statusFilter != "" {
			fmt.Printf("No schedules found with status: %s\n", statusFilter)
		} else {
			fmt.Println("No schedules found")
		}
		return nil
	}

	// Convert to interface{} slice for the printer
	var schedulesInterface []interface{}
	for _, schedule := range schedules {
		scheduleData := map[string]interface{}{
			"id":          schedule.ID,
			"name":        schedule.Name,
			"description": schedule.Description,
			"cron":        schedule.CronExpr,
			"enabled":     schedule.Enabled,
			"next_run":    schedule.NextRun,
			"last_run":    schedule.LastRun,
			"created_at":  schedule.CreatedAt,
		}
		schedulesInterface = append(schedulesInterface, scheduleData)
	}

	return printer.ScheduleTable(schedulesInterface)
}

func getSchedule(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	scheduleID := args[0]

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Fetching schedule details..."
	}

	schedule, err := apiClient.GetSchedule(scheduleID)

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to get schedule: %v", err)
	}

	// Format output based on output type
	switch config.Output {
	case "json":
		data, _ := json.MarshalIndent(schedule, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		printer.PrintYAML(schedule)
	default:
		printer.PrintScheduleDetails(schedule)
	}

	return nil
}

func createSchedule(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	// Get flags
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	cronExpr, _ := cmd.Flags().GetString("cron")
	playbookFile, _ := cmd.Flags().GetString("playbook")
	playbookJSON, _ := cmd.Flags().GetString("playbook-json")
	contextStr, _ := cmd.Flags().GetString("context")
	enabled, _ := cmd.Flags().GetBool("enabled")

	// Parse playbook
	var playbook interface{}
	if playbookFile != "" {
		// Read playbook from file
		data, err := os.ReadFile(playbookFile)
		if err != nil {
			return fmt.Errorf("failed to read playbook file: %v", err)
		}
		if err := json.Unmarshal(data, &playbook); err != nil {
			// If not JSON, use as playbook name
			playbook = playbookFile
		}
	} else if playbookJSON != "" {
		// Parse inline JSON
		if err := json.Unmarshal([]byte(playbookJSON), &playbook); err != nil {
			return fmt.Errorf("invalid playbook JSON: %v", err)
		}
	} else {
		return fmt.Errorf("either --playbook or --playbook-json is required")
	}

	// Parse context
	var context map[string]interface{}
	if err := json.Unmarshal([]byte(contextStr), &context); err != nil {
		return fmt.Errorf("invalid context JSON: %v", err)
	}

	// Create schedule request
	schedule := &client.Schedule{
		Name:        name,
		Description: description,
		CronExpr:    cronExpr,
		Playbook:    playbook,
		Context:     context,
		Enabled:     enabled,
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Creating schedule..."
	}

	createdSchedule, err := apiClient.CreateSchedule(schedule)

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to create schedule: %v", err)
	}

	printer.Success(fmt.Sprintf("Schedule created successfully: %s", createdSchedule.ID))

	// Display created schedule
	switch config.Output {
	case "json":
		data, _ := json.MarshalIndent(createdSchedule, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		printer.PrintYAML(createdSchedule)
	default:
		printer.PrintScheduleDetails(createdSchedule)
	}

	return nil
}

func updateSchedule(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	scheduleID := args[0]

	// Build update request
	updates := &client.Schedule{}
	hasUpdates := false

	if name, _ := cmd.Flags().GetString("name"); name != "" {
		updates.Name = name
		hasUpdates = true
	}

	if description, _ := cmd.Flags().GetString("description"); description != "" {
		updates.Description = description
		hasUpdates = true
	}

	if cronExpr, _ := cmd.Flags().GetString("cron"); cronExpr != "" {
		updates.CronExpr = cronExpr
		hasUpdates = true
	}

	// Handle playbook update
	if playbookFile, _ := cmd.Flags().GetString("playbook"); playbookFile != "" {
		data, err := os.ReadFile(playbookFile)
		if err != nil {
			return fmt.Errorf("failed to read playbook file: %v", err)
		}
		var playbook interface{}
		if err := json.Unmarshal(data, &playbook); err != nil {
			// If not JSON, use as playbook name
			playbook = playbookFile
		}
		updates.Playbook = playbook
		hasUpdates = true
	} else if playbookJSON, _ := cmd.Flags().GetString("playbook-json"); playbookJSON != "" {
		var playbook interface{}
		if err := json.Unmarshal([]byte(playbookJSON), &playbook); err != nil {
			return fmt.Errorf("invalid playbook JSON: %v", err)
		}
		updates.Playbook = playbook
		hasUpdates = true
	}

	// Handle context update
	if contextStr, _ := cmd.Flags().GetString("context"); contextStr != "" {
		var context map[string]interface{}
		if err := json.Unmarshal([]byte(contextStr), &context); err != nil {
			return fmt.Errorf("invalid context JSON: %v", err)
		}
		updates.Context = context
		hasUpdates = true
	}

	// Handle enable/disable
	if enable, _ := cmd.Flags().GetBool("enable"); enable {
		updates.Enabled = true
		hasUpdates = true
	} else if disable, _ := cmd.Flags().GetBool("disable"); disable {
		updates.Enabled = false
		hasUpdates = true
	}

	if !hasUpdates {
		return fmt.Errorf("no updates specified")
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Updating schedule..."
	}

	err := apiClient.UpdateSchedule(scheduleID, updates)

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to update schedule: %v", err)
	}

	printer.Success(fmt.Sprintf("Schedule updated successfully: %s", scheduleID))

	return nil
}

func deleteSchedule(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	scheduleID := args[0]
	force, _ := cmd.Flags().GetBool("force")

	// Confirmation prompt
	if !force {
		fmt.Printf("Are you sure you want to delete schedule %s? (y/N): ", scheduleID)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Deleting schedule..."
	}

	err := apiClient.DeleteSchedule(scheduleID)

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to delete schedule: %v", err)
	}

	printer.Success(fmt.Sprintf("Schedule deleted successfully: %s", scheduleID))
	return nil
}

func executeSchedule(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	scheduleID := args[0]

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Executing schedule..."
	}

	result, err := apiClient.ExecuteSchedule(scheduleID)

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to execute schedule: %v", err)
	}

	printer.Success(fmt.Sprintf("Schedule executed successfully: %s", scheduleID))

	// Display execution result
	switch config.Output {
	case "json":
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		printer.PrintYAML(result)
	default:
		fmt.Printf("\nExecution Results:\n")
		fmt.Printf("Schedule ID: %s\n", scheduleID)
		if result != nil {
			resultJSON, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(resultJSON))
		}
	}

	return nil
}

func scheduleStats(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Fetching schedule statistics..."
	}

	stats, err := apiClient.GetScheduleStats()

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to get schedule statistics: %v", err)
	}

	// Display statistics
	switch config.Output {
	case "json":
		data, _ := json.MarshalIndent(stats, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		printer.PrintYAML(stats)
	default:
		printer.PrintScheduleStats(stats)
	}

	return nil
}
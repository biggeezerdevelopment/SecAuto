package cmd

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"secauto-cli/pkg/client"
	"secauto-cli/pkg/output"
)

// jobCmd represents the job command
var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "Manage and monitor jobs",
	Long:  `List, get status, and manage SecAuto jobs.`,
}

// jobListCmd represents the job list command
var jobListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all jobs",
	Long:  `List all jobs with their current status.`,
	RunE:  listJobs,
}

// jobGetCmd represents the job get command
var jobGetCmd = &cobra.Command{
	Use:   "get <job-id>",
	Short: "Get job details",
	Long:  `Get detailed information about a specific job.`,
	Args:  cobra.ExactArgs(1),
	RunE:  getJob,
}

// jobWatchCmd represents the job watch command
var jobWatchCmd = &cobra.Command{
	Use:   "watch <job-id>",
	Short: "Watch job until completion",
	Long:  `Watch a job and display status updates until it completes or fails.`,
	Args:  cobra.ExactArgs(1),
	RunE:  watchJobCmd,
}

// jobStatsCmd represents the job stats command
var jobStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show job statistics",
	Long:  `Show statistics for job execution including counts by status.`,
	RunE:  getJobStats,
}

func init() {
	rootCmd.AddCommand(jobCmd)
	jobCmd.AddCommand(jobListCmd)
	jobCmd.AddCommand(jobGetCmd)
	jobCmd.AddCommand(jobWatchCmd)
	jobCmd.AddCommand(jobStatsCmd)

	// Job list flags
	jobListCmd.Flags().String("status", "", "Filter jobs by status (pending, running, completed, failed)")
	jobListCmd.Flags().Int("limit", 0, "Limit number of jobs to display")
}

func listJobs(cmd *cobra.Command, args []string) error {
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
		s.Suffix = " Fetching jobs..."
	}

	jobs, err := apiClient.ListJobs()
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to list jobs: %v", err)
	}

	// Get filter flags
	statusFilter, _ := cmd.Flags().GetString("status")
	limit, _ := cmd.Flags().GetInt("limit")

	// Filter jobs
	var filteredJobs []*client.Job
	for _, job := range jobs {
		if statusFilter != "" && job.Status != statusFilter {
			continue
		}
		filteredJobs = append(filteredJobs, job)
		if limit > 0 && len(filteredJobs) >= limit {
			break
		}
	}

	if len(filteredJobs) == 0 {
		if statusFilter != "" {
			fmt.Printf("No jobs found with status: %s\n", statusFilter)
		} else {
			fmt.Println("No jobs found")
		}
		return nil
	}

	// Convert to interface{} slice for the job table function
	var jobsInterface []interface{}
	for _, job := range filteredJobs {
		jobData := map[string]interface{}{
			"id":           job.ID,
			"status":       job.Status,
			"created_at":   formatTimePtr(job.CreatedAt),
			"started_at":   formatTimePtr(job.StartedAt),
			"completed_at": formatTimePtr(job.CompletedAt),
		}
		jobsInterface = append(jobsInterface, jobData)
	}

	return printer.JobTable(jobsInterface)
}

func getJob(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	jobID := args[0]

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Fetching job details..."
	}

	job, err := apiClient.GetJob(jobID)
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to get job: %v", err)
	}

	// Create detailed job information
	jobInfo := map[string]interface{}{
		"ID":          job.ID,
		"Status":      printer.FormatStatus(job.Status),
		"Priority":    job.Priority,
		"Created":     output.FormatTime(job.CreatedAt),
		"Started":     output.FormatTime(job.StartedAt),
		"Completed":   output.FormatTime(job.CompletedAt),
	}

	// Calculate duration if applicable
	if job.StartedAt != nil && job.CompletedAt != nil {
		duration := job.CompletedAt.Sub(*job.StartedAt)
		jobInfo["Duration"] = output.FormatDuration(duration)
	} else if job.StartedAt != nil {
		duration := time.Since(*job.StartedAt)
		jobInfo["Running For"] = output.FormatDuration(duration)
	}

	if job.Error != "" {
		jobInfo["Error"] = job.Error
	}

	fmt.Printf("Job Details:\n")
	for key, value := range jobInfo {
		fmt.Printf("  %s: %v\n", key, value)
	}

	if job.Context != nil && len(job.Context) > 0 {
		fmt.Printf("\nContext:\n")
		printer.Print(job.Context)
	}

	if job.Results != nil {
		fmt.Printf("\nResults:\n")
		printer.Print(job.Results)
	}

	if job.Metadata != nil && len(job.Metadata) > 0 {
		fmt.Printf("\nMetadata:\n")
		printer.Print(job.Metadata)
	}

	return nil
}

func watchJobCmd(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	jobID := args[0]

	printer.PrintInfo(fmt.Sprintf("Watching job: %s", jobID))
	printer.PrintInfo("Press Ctrl+C to stop watching")

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Watching job..."
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastStatus string

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

			// Update spinner suffix with current status
			if lastStatus != job.Status {
				if !printer.NoColor {
					s.Stop()
				}
				
				printer.PrintInfo(fmt.Sprintf("Status changed: %s -> %s", 
					printer.FormatStatus(lastStatus), 
					printer.FormatStatus(job.Status)))
				
				lastStatus = job.Status
				
				if !printer.NoColor {
					s.Start()
					s.Suffix = fmt.Sprintf(" Job status: %s", job.Status)
				}
			}

			switch job.Status {
			case "completed":
				if !printer.NoColor {
					s.Stop()
				}
				printer.PrintSuccess("Job completed successfully")
				
				if job.StartedAt != nil && job.CompletedAt != nil {
					duration := job.CompletedAt.Sub(*job.StartedAt)
					fmt.Printf("Execution time: %s\n", output.FormatDuration(duration))
				}
				
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

func getJobStats(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Fetching job statistics..."
	}

	stats, err := apiClient.GetJobStats()

	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to get job stats: %v", err)
	}

	return printer.Print(stats)
}

// Helper function to format time pointer for display
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
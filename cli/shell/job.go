package shell

import (
	"fmt"
	"time"

	"github.com/abiosoft/ishell/v2"
	"github.com/briandowns/spinner"
)

// RegisterJobCommands registers job-related commands
func RegisterJobCommands(sh *ishell.Shell, ctx *Context) {
	jobCmd := &ishell.Cmd{
		Name: "job",
		Help: "Manage and monitor jobs",
	}

	// job list
	jobCmd.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "List all jobs",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching jobs..."
			}

			jobs, err := ctx.Client.ListJobs()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to list jobs: %v", err))
				return
			}

			if len(jobs) == 0 {
				c.Printf("No jobs found\n")
				return
			}

			// Convert jobs to interface{} slice for table display
			var jobsInterface []interface{}
			for _, job := range jobs {
				jobData := map[string]interface{}{
					"id":           job.ID[:8] + "...",
					"status":       job.Status,
					"created_at":   formatTimePtr(job.CreatedAt),
				}
				jobsInterface = append(jobsInterface, jobData)
			}

			if err := ctx.Printer.JobTable(jobsInterface); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// job get
	jobCmd.AddCmd(&ishell.Cmd{
		Name: "get",
		Help: "Get job details",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "job get <job-id>") {
				return
			}

			jobID := c.Args[0]

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching job details..."
			}

			job, err := ctx.Client.GetJob(jobID)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get job: %v", err))
				return
			}

			// Display job details
			c.Printf("=== Job Details ===\n")
			c.Printf("ID: %s\n", job.ID)
			c.Printf("Status: %s\n", ctx.Printer.FormatStatus(job.Status))
			c.Printf("Priority: %d\n", job.Priority)

			if job.CreatedAt != nil {
				c.Printf("Created: %s\n", job.CreatedAt.Format(time.RFC3339))
			}
			if job.StartedAt != nil {
				c.Printf("Started: %s\n", job.StartedAt.Format(time.RFC3339))
			}
			if job.CompletedAt != nil {
				c.Printf("Completed: %s\n", job.CompletedAt.Format(time.RFC3339))
			}

			if job.Error != "" {
				c.Printf("Error: %s\n", job.Error)
			}

			if job.Results != nil {
				c.Printf("\nResults:\n")
				if err := ctx.Printer.Print(job.Results); err != nil {
					ctx.PrintError(c, err)
				}
			}
		},
	})

	// job stats
	jobCmd.AddCmd(&ishell.Cmd{
		Name: "stats",
		Help: "Show job statistics",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching job statistics..."
			}

			stats, err := ctx.Client.GetJobStats()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get job stats: %v", err))
				return
			}

			if err := ctx.Printer.Print(stats); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	sh.AddCmd(jobCmd)
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("15:04:05")
}
package shell

import (
	"fmt"
	"time"

	"github.com/abiosoft/ishell/v2"
	"github.com/briandowns/spinner"
)

// RegisterClusterCommands registers cluster-related commands
func RegisterClusterCommands(sh *ishell.Shell, ctx *Context) {
	clusterCmd := &ishell.Cmd{
		Name: "cluster",
		Help: "Manage cluster operations",
	}

	// cluster status
	clusterCmd.AddCmd(&ishell.Cmd{
		Name: "status",
		Help: "Show cluster status",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching cluster status..."
			}

			status, err := ctx.Client.GetClusterStatus()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get cluster status: %v", err))
				return
			}

			// Format the output
			data := map[string]interface{}{
				"Node ID":      status.NodeID,
				"Total Nodes":  status.TotalNodes,
				"Active Nodes": status.ActiveNodes,
				"Status":       status.Status,
			}

			if err := ctx.Printer.Print(data); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// cluster jobs
	clusterCmd.AddCmd(&ishell.Cmd{
		Name: "jobs",
		Help: "List cluster jobs",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching cluster jobs..."
			}

			jobs, err := ctx.Client.GetClusterJobs()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get cluster jobs: %v", err))
				return
			}

			if len(jobs) == 0 {
				c.Printf("No cluster jobs found\n")
				return
			}

			if err := ctx.Printer.Print(jobs); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// cluster job
	clusterCmd.AddCmd(&ishell.Cmd{
		Name: "job",
		Help: "Get cluster job details (job <job-id>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "cluster job <job-id>") {
				return
			}

			jobID := c.Args[0]

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching cluster job details..."
			}

			job, err := ctx.Client.GetClusterJob(jobID)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get cluster job '%s': %v", jobID, err))
				return
			}

			if err := ctx.Printer.Print(job); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	sh.AddCmd(clusterCmd)
}
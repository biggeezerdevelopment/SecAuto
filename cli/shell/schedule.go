package shell

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/abiosoft/ishell/v2"
	"github.com/briandowns/spinner"
)

// RegisterScheduleCommands registers schedule-related commands
func RegisterScheduleCommands(sh *ishell.Shell, ctx *Context) {
	scheduleCmd := &ishell.Cmd{
		Name: "schedule",
		Help: "Manage job schedules",
	}

	// schedule list
	scheduleCmd.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "List all schedules (list [status])",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			status := ""
			if len(c.Args) > 0 {
				status = c.Args[0]
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching schedules..."
			}

			schedules, err := ctx.Client.ListSchedules(status)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to list schedules: %v", err))
				return
			}

			if len(schedules) == 0 {
				c.Printf("No schedules found\n")
				return
			}

			if err := ctx.Printer.Print(schedules); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// schedule get
	scheduleCmd.AddCmd(&ishell.Cmd{
		Name: "get",
		Help: "Get schedule details (get <schedule-id>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "schedule get <schedule-id>") {
				return
			}

			scheduleID := c.Args[0]

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching schedule details..."
			}

			schedule, err := ctx.Client.GetSchedule(scheduleID)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get schedule '%s': %v", scheduleID, err))
				return
			}

			if err := ctx.Printer.Print(schedule); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// schedule create
	scheduleCmd.AddCmd(&ishell.Cmd{
		Name: "create",
		Help: "Create a new schedule (create <json-file>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			c.Printf("Enter schedule name: ")
			name := c.ReadLine()

			c.Printf("Enter cron expression (e.g., '0 0 * * *'): ")
			cronExpr := c.ReadLine()

			c.Printf("Enter playbook name or leave empty to specify inline: ")
			playbookName := c.ReadLine()

			var playbook interface{}
			if playbookName == "" {
				c.Printf("Enter playbook JSON (or type 'cancel'): ")
				playbookJSON := c.ReadLine()
				if playbookJSON == "cancel" {
					c.Printf("Schedule creation cancelled\n")
					return
				}
				if err := json.Unmarshal([]byte(playbookJSON), &playbook); err != nil {
					ctx.PrintError(c, fmt.Errorf("invalid playbook JSON: %v", err))
					return
				}
			}

			c.Printf("Enter description (optional): ")
			description := c.ReadLine()

			schedule := map[string]interface{}{
				"name":            name,
				"cron_expression": cronExpr,
				"description":     description,
				"enabled":         true,
			}

			if playbookName != "" {
				schedule["playbook_name"] = playbookName
			} else {
				schedule["playbook"] = playbook
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Creating schedule..."
			}

			result, err := ctx.Client.CreateSchedule(schedule)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to create schedule: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Schedule created with ID: %s", result.ID))
		},
	})

	// schedule update
	scheduleCmd.AddCmd(&ishell.Cmd{
		Name: "update",
		Help: "Update a schedule (update <schedule-id>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "schedule update <schedule-id>") {
				return
			}

			scheduleID := c.Args[0]

			// Get current schedule
			current, err := ctx.Client.GetSchedule(scheduleID)
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get current schedule: %v", err))
				return
			}

			c.Printf("Current cron expression: %s\n", current.CronExpr)
			c.Printf("Enter new cron expression (or press Enter to keep): ")
			cronExpr := c.ReadLine()
			if cronExpr == "" {
				cronExpr = current.CronExpr
			}

			c.Printf("Current enabled status: %v\n", current.Enabled)
			c.Printf("Enable schedule? (y/n, press Enter to keep): ")
			enabledStr := c.ReadLine()
			enabled := current.Enabled
			if enabledStr == "y" || enabledStr == "yes" {
				enabled = true
			} else if enabledStr == "n" || enabledStr == "no" {
				enabled = false
			}

			updates := map[string]interface{}{
				"cron_expression": cronExpr,
				"enabled":         enabled,
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Updating schedule..."
			}

			err = ctx.Client.UpdateSchedule(scheduleID, updates)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to update schedule: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Schedule '%s' updated successfully", scheduleID))
		},
	})

	// schedule delete
	scheduleCmd.AddCmd(&ishell.Cmd{
		Name: "delete",
		Help: "Delete a schedule (delete <schedule-id>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "schedule delete <schedule-id>") {
				return
			}

			scheduleID := c.Args[0]

			// Confirm deletion
			c.Printf("Are you sure you want to delete schedule '%s'? (y/N): ", scheduleID)
			confirmation := c.ReadLine()
			if confirmation != "y" && confirmation != "yes" && confirmation != "Y" && confirmation != "YES" {
				c.Printf("Deletion cancelled\n")
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Deleting schedule..."
			}

			err := ctx.Client.DeleteSchedule(scheduleID)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to delete schedule: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Schedule '%s' deleted successfully", scheduleID))
		},
	})

	// schedule execute
	scheduleCmd.AddCmd(&ishell.Cmd{
		Name: "execute",
		Help: "Execute a schedule immediately (execute <schedule-id>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "schedule execute <schedule-id>") {
				return
			}

			scheduleID := c.Args[0]

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Executing schedule..."
			}

			result, err := ctx.Client.ExecuteSchedule(scheduleID)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to execute schedule '%s': %v", scheduleID, err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Schedule executed successfully. Job ID: %s", result.JobID))
		},
	})

	// schedule stats
	scheduleCmd.AddCmd(&ishell.Cmd{
		Name: "stats",
		Help: "Show schedule statistics",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching schedule statistics..."
			}

			stats, err := ctx.Client.GetScheduleStats()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to get schedule stats: %v", err))
				return
			}

			if err := ctx.Printer.Print(stats); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	sh.AddCmd(scheduleCmd)
}
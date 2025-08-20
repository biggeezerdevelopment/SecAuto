package shell

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/abiosoft/ishell/v2"
	"github.com/briandowns/spinner"
)

// RegisterAutomationCommands registers automation-related commands
func RegisterAutomationCommands(sh *ishell.Shell, ctx *Context) {
	automationCmd := &ishell.Cmd{
		Name: "automation",
		Help: "Manage automations",
	}

	// automation list
	automationCmd.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "List all automations",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching automations..."
			}

			automations, err := ctx.Client.ListAutomations()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to list automations: %v", err))
				return
			}

			if len(automations) == 0 {
				c.Printf("No automations found\n")
				return
			}

			// Convert to format suitable for table output
			var data []map[string]interface{}
			for i, automation := range automations {
				data = append(data, map[string]interface{}{
					"#":    i + 1,
					"Name": automation,
				})
			}

			if err := ctx.Printer.Print(data); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// automation upload
	automationCmd.AddCmd(&ishell.Cmd{
		Name: "upload",
		Help: "Upload an automation script (upload <file>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "automation upload <file>") {
				return
			}

			automationFile := c.Args[0]

			// Read file content
			file, err := os.Open(automationFile)
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to open automation file: %v", err))
				return
			}
			defer file.Close()

			content, err := io.ReadAll(file)
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to read automation file: %v", err))
				return
			}

			// Get the base filename
			filename := automationFile
			if idx := strings.LastIndex(automationFile, "/"); idx != -1 {
				filename = automationFile[idx+1:]
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Uploading automation..."
			}

			err = ctx.Client.UploadAutomation(filename, content)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("upload failed: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Automation '%s' uploaded successfully", automationFile))
		},
	})

	// automation delete
	automationCmd.AddCmd(&ishell.Cmd{
		Name: "delete",
		Help: "Delete an automation script (delete <name>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "automation delete <name>") {
				return
			}

			automationName := c.Args[0]

			// Confirm deletion
			c.Printf("Are you sure you want to delete automation '%s'? (y/N): ", automationName)
			confirmation := c.ReadLine()
			if confirmation != "y" && confirmation != "yes" && confirmation != "Y" && confirmation != "YES" {
				c.Printf("Deletion cancelled\n")
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Deleting automation..."
			}

			err := ctx.Client.DeleteAutomation(automationName)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("delete failed: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Automation '%s' deleted successfully", automationName))
		},
	})

	sh.AddCmd(automationCmd)
}
package shell

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abiosoft/ishell/v2"
	"github.com/briandowns/spinner"
	"secauto-cli/pkg/client"
)

// RegisterPlaybookCommands registers playbook-related commands
func RegisterPlaybookCommands(sh *ishell.Shell, ctx *Context) {
	playbookCmd := &ishell.Cmd{
		Name: "playbook",
		Help: "Manage and execute playbooks",
	}

	// playbook list
	playbookCmd.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "List all available playbooks",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Fetching playbooks..."
			}

			playbooks, err := ctx.Client.ListPlaybooks()

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to list playbooks: %v", err))
				return
			}

			if len(playbooks) == 0 {
				c.Printf("No playbooks found\n")
				return
			}

			// Convert to format suitable for table output
			var data []map[string]interface{}
			for i, playbook := range playbooks {
				data = append(data, map[string]interface{}{
					"#":    i + 1,
					"Name": playbook,
				})
			}

			if err := ctx.Printer.Print(data); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// playbook execute
	playbookCmd.AddCmd(&ishell.Cmd{
		Name: "execute",
		Help: "Execute a playbook (execute <file|name> [--async] [--context <json>])",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "playbook execute <file|name> [--async] [--context <json>]") {
				return
			}

			playbookArg := c.Args[0]
			async := contains(c.Args, "--async")
			watch := contains(c.Args, "--watch")

			// Parse context if provided
			var context map[string]interface{}
			if contextIndex := findFlag(c.Args, "--context"); contextIndex != -1 && contextIndex+1 < len(c.Args) {
				contextStr := c.Args[contextIndex+1]
				var err error
				context, err = parseContext(contextStr)
				if err != nil {
					ctx.PrintError(c, fmt.Errorf("failed to parse context: %v", err))
					return
				}
			}

			// Determine if it's a file or playbook name
			var playbook interface{}

			if _, err := os.Stat(playbookArg); err == nil {
				// It's a file
				playbook, err = loadPlaybookFromFile(playbookArg)
				if err != nil {
					ctx.PrintError(c, fmt.Errorf("failed to load playbook: %v", err))
					return
				}
			} else {
				// It's a playbook name
				playbook = playbookArg
			}

			// Prepare request
			req := &client.PlaybookRequest{
				Context: context,
			}

			if _, isFile := playbook.(map[string]interface{}); isFile || isSlice(playbook) {
				req.Playbook = playbook
			} else {
				req.PlaybookName = playbookArg
			}

			// Execute playbook
			if async {
				executePlaybookAsync(c, ctx, req, watch)
			} else {
				executePlaybookSync(c, ctx, req)
			}
		},
	})

	// playbook upload
	playbookCmd.AddCmd(&ishell.Cmd{
		Name: "upload",
		Help: "Upload a playbook (upload <file> [name])",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "playbook upload <file> [name]") {
				return
			}

			playbookFile := c.Args[0]
			var name string
			if len(c.Args) > 1 {
				name = c.Args[1]
			} else {
				// Use filename without extension as name
				name = strings.TrimSuffix(filepath.Base(playbookFile), filepath.Ext(playbookFile))
			}

			// Load playbook from file
			playbook, err := loadPlaybookFromFile(playbookFile)
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to load playbook: %v", err))
				return
			}

			// Upload playbook
			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			if !ctx.Config.NoColor {
				s.Start()
				s.Suffix = " Uploading playbook..."
			}

			err = ctx.Client.UploadPlaybook(name, playbook)

			if !ctx.Config.NoColor {
				s.Stop()
			}

			if err != nil {
				ctx.PrintError(c, fmt.Errorf("upload failed: %v", err))
				return
			}

			ctx.PrintSuccess(c, fmt.Sprintf("Playbook '%s' uploaded successfully", name))
		},
	})

	// playbook validate
	playbookCmd.AddCmd(&ishell.Cmd{
		Name: "validate",
		Help: "Validate a playbook file",
		Func: func(c *ishell.Context) {
			if !ctx.RequireArgs(c, 1, "playbook validate <file>") {
				return
			}

			playbookFile := c.Args[0]

			// Load and validate playbook
			playbook, err := loadPlaybookFromFile(playbookFile)
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("validation failed: %v", err))
				return
			}

			// Basic validation
			if playbook == nil {
				ctx.PrintError(c, fmt.Errorf("playbook is empty"))
				return
			}

			// Check if it's a valid array or object
			switch v := playbook.(type) {
			case []interface{}:
				ctx.PrintSuccess(c, fmt.Sprintf("Playbook is valid (array with %d steps)", len(v)))
			case map[string]interface{}:
				ctx.PrintSuccess(c, "Playbook is valid (object format)")
			default:
				ctx.PrintError(c, fmt.Errorf("invalid playbook format"))
				return
			}

			// Pretty print playbook structure
			if ctx.Config.Output == "json" {
				if err := ctx.Printer.Print(playbook); err != nil {
					ctx.PrintError(c, err)
				}
			}
		},
	})

	sh.AddCmd(playbookCmd)
}

func executePlaybookSync(c *ishell.Context, ctx *Context, req *client.PlaybookRequest) {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !ctx.Config.NoColor {
		s.Start()
		s.Suffix = " Executing playbook..."
	}

	resp, err := ctx.Client.ExecutePlaybook(req)

	if !ctx.Config.NoColor {
		s.Stop()
	}

	if err != nil {
		ctx.PrintError(c, fmt.Errorf("execution failed: %v", err))
		return
	}

	if !resp.Success {
		ctx.PrintError(c, fmt.Errorf("playbook execution failed: %s", resp.Error))
		return
	}

	ctx.PrintSuccess(c, "Playbook executed successfully")

	if resp.Results != nil {
		c.Printf("\nResults:\n")
		if err := ctx.Printer.Print(resp.Results); err != nil {
			ctx.PrintError(c, err)
		}
	}
}

func executePlaybookAsync(c *ishell.Context, ctx *Context, req *client.PlaybookRequest, watch bool) {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !ctx.Config.NoColor {
		s.Start()
		s.Suffix = " Starting playbook execution..."
	}

	resp, err := ctx.Client.ExecutePlaybookAsync(req)

	if !ctx.Config.NoColor {
		s.Stop()
	}

	if err != nil {
		ctx.PrintError(c, fmt.Errorf("execution failed: %v", err))
		return
	}

	if !resp.Success {
		ctx.PrintError(c, fmt.Errorf("playbook execution failed: %s", resp.Error))
		return
	}

	ctx.PrintSuccess(c, fmt.Sprintf("Playbook execution started. Job ID: %s", resp.JobID))

	if watch && resp.JobID != "" {
		watchJob(c, ctx, resp.JobID)
	} else {
		c.Printf("\nUse 'job get %s' to check status\n", resp.JobID)
	}
}

func watchJob(c *ishell.Context, ctx *Context, jobID string) {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !ctx.Config.NoColor {
		s.Start()
		s.Suffix = " Watching job..."
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			job, err := ctx.Client.GetJob(jobID)
			if err != nil {
				if !ctx.Config.NoColor {
					s.Stop()
				}
				ctx.PrintError(c, fmt.Errorf("failed to get job status: %v", err))
				return
			}

			switch job.Status {
			case "completed":
				if !ctx.Config.NoColor {
					s.Stop()
				}
				ctx.PrintSuccess(c, "Job completed successfully")
				if job.Results != nil {
					c.Printf("\nResults:\n")
					if err := ctx.Printer.Print(job.Results); err != nil {
						ctx.PrintError(c, err)
					}
				}
				return
			case "failed":
				if !ctx.Config.NoColor {
					s.Stop()
				}
				ctx.PrintError(c, fmt.Errorf("job failed: %s", job.Error))
				return
			}
		}
	}
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

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func findFlag(args []string, flag string) int {
	for i, arg := range args {
		if arg == flag {
			return i
		}
	}
	return -1
}

func isSlice(v interface{}) bool {
	_, ok := v.([]interface{})
	return ok
}
package shell

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abiosoft/ishell/v2"
	"github.com/briandowns/spinner"
	"secauto-cli/pkg/client"
)

// RegisterBatchCommands registers batch operation commands
func RegisterBatchCommands(sh *ishell.Shell, ctx *Context) {
	batchCmd := &ishell.Cmd{
		Name: "batch",
		Help: "Execute batch operations",
	}

	// batch playbook
	batchCmd.AddCmd(&ishell.Cmd{
		Name: "playbook",
		Help: "Execute multiple playbooks (playbook <file1> [file2] ...)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "batch playbook <file1> [file2] ...") {
				return
			}

			files := c.Args
			results := make([]map[string]interface{}, 0)

			for _, file := range files {
				c.Printf("Executing playbook: %s\n", file)

				// Read playbook file
				content, err := os.ReadFile(file)
				if err != nil {
					results = append(results, map[string]interface{}{
						"File":    file,
						"Status":  "Failed",
						"Error":   fmt.Sprintf("Failed to read file: %v", err),
					})
					continue
				}

				// Parse playbook
				var playbook interface{}
				if err := json.Unmarshal(content, &playbook); err != nil {
					results = append(results, map[string]interface{}{
						"File":    file,
						"Status":  "Failed",
						"Error":   fmt.Sprintf("Invalid JSON: %v", err),
					})
					continue
				}

				// Execute playbook
				s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
				if !ctx.Config.NoColor {
					s.Start()
					s.Suffix = fmt.Sprintf(" Executing %s...", filepath.Base(file))
				}

				req := &client.PlaybookRequest{
					Playbook: playbook,
				}

				result, err := ctx.Client.ExecutePlaybook(req)

				if !ctx.Config.NoColor {
					s.Stop()
				}

				if err != nil {
					results = append(results, map[string]interface{}{
						"File":    file,
						"Status":  "Failed",
						"Error":   err.Error(),
					})
				} else {
					results = append(results, map[string]interface{}{
						"File":    file,
						"Status":  "Success",
						"JobID":   result.JobID,
					})
				}
			}

			// Display results
			c.Printf("\n=== Batch Execution Results ===\n")
			if err := ctx.Printer.Print(results); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// batch upload
	batchCmd.AddCmd(&ishell.Cmd{
		Name: "upload",
		Help: "Upload multiple files (upload <directory> [--playbooks|--automations])",
		Func: func(c *ishell.Context) {
			if !ctx.RequireConnection(c) {
				return
			}

			if !ctx.RequireArgs(c, 1, "batch upload <directory> [--playbooks|--automations]") {
				return
			}

			directory := c.Args[0]
			uploadType := "all" // default

			if len(c.Args) > 1 {
				switch c.Args[1] {
				case "--playbooks":
					uploadType = "playbooks"
				case "--automations":
					uploadType = "automations"
				}
			}

			// Check if directory exists
			info, err := os.Stat(directory)
			if err != nil || !info.IsDir() {
				ctx.PrintError(c, fmt.Errorf("directory '%s' does not exist or is not a directory", directory))
				return
			}

			// Read directory
			entries, err := os.ReadDir(directory)
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to read directory: %v", err))
				return
			}

			results := make([]map[string]interface{}, 0)

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				filename := entry.Name()
				filepath := filepath.Join(directory, filename)

				// Determine file type
				isPlaybook := strings.HasSuffix(filename, ".json")
				isAutomation := strings.HasSuffix(filename, ".py")

				// Skip based on upload type
				if uploadType == "playbooks" && !isPlaybook {
					continue
				}
				if uploadType == "automations" && !isAutomation {
					continue
				}
				if !isPlaybook && !isAutomation {
					continue
				}

				c.Printf("Uploading: %s\n", filename)

				// Read file
				content, err := os.ReadFile(filepath)
				if err != nil {
					results = append(results, map[string]interface{}{
						"File":   filename,
						"Type":   getFileType(filename),
						"Status": "Failed",
						"Error":  fmt.Sprintf("Failed to read file: %v", err),
					})
					continue
				}

				s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
				if !ctx.Config.NoColor {
					s.Start()
					s.Suffix = fmt.Sprintf(" Uploading %s...", filename)
				}

				var uploadErr error
				if isPlaybook {
					// Parse and upload playbook
					var playbook interface{}
					if err := json.Unmarshal(content, &playbook); err != nil {
						uploadErr = fmt.Errorf("invalid JSON: %v", err)
					} else {
						name := strings.TrimSuffix(filename, ".json")
						uploadErr = ctx.Client.UploadPlaybook(name, playbook)
					}
				} else if isAutomation {
					// Upload automation
					uploadErr = ctx.Client.UploadAutomation(filename, content)
				}

				if !ctx.Config.NoColor {
					s.Stop()
				}

				if uploadErr != nil {
					results = append(results, map[string]interface{}{
						"File":   filename,
						"Type":   getFileType(filename),
						"Status": "Failed",
						"Error":  uploadErr.Error(),
					})
				} else {
					results = append(results, map[string]interface{}{
						"File":   filename,
						"Type":   getFileType(filename),
						"Status": "Success",
					})
				}
			}

			// Display results
			c.Printf("\n=== Batch Upload Results ===\n")
			if err := ctx.Printer.Print(results); err != nil {
				ctx.PrintError(c, err)
			}
		},
	})

	// batch execute
	batchCmd.AddCmd(&ishell.Cmd{
		Name: "execute",
		Help: "Execute commands from a file (execute <commands-file>)",
		Func: func(c *ishell.Context) {
			if !ctx.RequireArgs(c, 1, "batch execute <commands-file>") {
				return
			}

			file := c.Args[0]

			// Read commands file
			content, err := os.ReadFile(file)
			if err != nil {
				ctx.PrintError(c, fmt.Errorf("failed to read commands file: %v", err))
				return
			}

			// Split into lines
			lines := strings.Split(string(content), "\n")

			c.Printf("Executing %d commands from %s\n\n", len(lines), file)

			for i, line := range lines {
				line = strings.TrimSpace(line)
				
				// Skip empty lines and comments
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				c.Printf("[%d] Executing: %s\n", i+1, line)
				
				// Process the command through the shell
				ctx.Shell.Process(line)
				
				// Small delay between commands
				time.Sleep(100 * time.Millisecond)
			}

			c.Printf("\nBatch execution completed\n")
		},
	})

	sh.AddCmd(batchCmd)
}

func getFileType(filename string) string {
	if strings.HasSuffix(filename, ".json") {
		return "Playbook"
	}
	if strings.HasSuffix(filename, ".py") {
		return "Automation"
	}
	return "Unknown"
}
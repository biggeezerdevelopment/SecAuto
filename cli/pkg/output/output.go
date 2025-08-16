package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v2"
)

// Format represents output format types
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

// Printer handles output formatting
type Printer struct {
	Format  Format
	NoColor bool
	Writer  *os.File
}

// NewPrinter creates a new output printer
func NewPrinter(format string, noColor bool) *Printer {
	return &Printer{
		Format:  Format(format),
		NoColor: noColor,
		Writer:  os.Stdout,
	}
}

// Print outputs data in the specified format
func (p *Printer) Print(data interface{}) error {
	switch p.Format {
	case FormatJSON:
		return p.printJSON(data)
	case FormatYAML:
		return p.printYAML(data)
	case FormatTable:
		return p.printTable(data)
	default:
		return fmt.Errorf("unsupported output format: %s", p.Format)
	}
}

// printJSON outputs data as JSON
func (p *Printer) printJSON(data interface{}) error {
	encoder := json.NewEncoder(p.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// printYAML outputs data as YAML
func (p *Printer) printYAML(data interface{}) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	_, err = p.Writer.Write(yamlData)
	return err
}

// printTable outputs data as a table
func (p *Printer) printTable(data interface{}) error {
	switch v := data.(type) {
	case []map[string]interface{}:
		return p.printGenericTable(v)
	default:
		// For single objects, convert to JSON for readable output
		return p.printJSON(data)
	}
}

// printGenericTable prints a generic table from slice of maps
func (p *Printer) printGenericTable(data []map[string]interface{}) error {
	if len(data) == 0 {
		fmt.Println("No data to display")
		return nil
	}

	// Get all unique keys for headers
	headers := make(map[string]bool)
	for _, row := range data {
		for key := range row {
			headers[key] = true
		}
	}

	// Convert to sorted slice
	var headerSlice []string
	for header := range headers {
		headerSlice = append(headerSlice, header)
	}

	table := tablewriter.NewWriter(p.Writer)
	table.SetHeader(headerSlice)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	if !p.NoColor {
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		)
	}

	for _, row := range data {
		var rowData []string
		for _, header := range headerSlice {
			if val, exists := row[header]; exists {
				rowData = append(rowData, fmt.Sprintf("%v", val))
			} else {
				rowData = append(rowData, "")
			}
		}
		table.Append(rowData)
	}

	table.Render()
	return nil
}

// PrintSuccess prints a success message
func (p *Printer) PrintSuccess(message string) {
	if p.NoColor {
		fmt.Fprintf(p.Writer, "✓ %s\n", message)
	} else {
		color.New(color.FgGreen).Fprintf(p.Writer, "✓ %s\n", message)
	}
}

// PrintError prints an error message
func (p *Printer) PrintError(message string) {
	if p.NoColor {
		fmt.Fprintf(p.Writer, "✗ %s\n", message)
	} else {
		color.New(color.FgRed).Fprintf(p.Writer, "✗ %s\n", message)
	}
}

// PrintWarning prints a warning message
func (p *Printer) PrintWarning(message string) {
	if p.NoColor {
		fmt.Fprintf(p.Writer, "⚠ %s\n", message)
	} else {
		color.New(color.FgYellow).Fprintf(p.Writer, "⚠ %s\n", message)
	}
}

// PrintInfo prints an info message
func (p *Printer) PrintInfo(message string) {
	if p.NoColor {
		fmt.Fprintf(p.Writer, "ℹ %s\n", message)
	} else {
		color.New(color.FgBlue).Fprintf(p.Writer, "ℹ %s\n", message)
	}
}

// FormatDuration formats a time duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}

// FormatTime formats a time pointer in a human-readable way
func FormatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

// FormatStatus formats a status with color
func (p *Printer) FormatStatus(status string) string {
	if p.NoColor {
		return status
	}

	switch strings.ToLower(status) {
	case "completed", "success", "active", "healthy":
		return color.GreenString(status)
	case "failed", "error", "inactive", "unhealthy":
		return color.RedString(status)
	case "running", "in_progress", "pending":
		return color.YellowString(status)
	default:
		return status
	}
}

// JobTable creates a formatted table for jobs
func (p *Printer) JobTable(jobs []interface{}) error {
	if len(jobs) == 0 {
		fmt.Println("No jobs found")
		return nil
	}

	table := tablewriter.NewWriter(p.Writer)
	table.SetHeader([]string{"ID", "Status", "Created", "Started", "Completed", "Duration"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	if !p.NoColor {
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		)
	}

	for _, job := range jobs {
		if jobMap, ok := job.(map[string]interface{}); ok {
			id := fmt.Sprintf("%.8s", getString(jobMap, "id"))
			status := p.FormatStatus(getString(jobMap, "status"))
			
			var created, started, completed, duration string
			
			if createdStr := getString(jobMap, "created_at"); createdStr != "" {
				if createdTime, err := time.Parse(time.RFC3339, createdStr); err == nil {
					created = createdTime.Format("15:04:05")
				}
			}
			
			if startedStr := getString(jobMap, "started_at"); startedStr != "" {
				if startedTime, err := time.Parse(time.RFC3339, startedStr); err == nil {
					started = startedTime.Format("15:04:05")
				}
			}
			
			if completedStr := getString(jobMap, "completed_at"); completedStr != "" {
				if completedTime, err := time.Parse(time.RFC3339, completedStr); err == nil {
					completed = completedTime.Format("15:04:05")
					
					// Calculate duration if we have both start and completion times
					if startedStr := getString(jobMap, "started_at"); startedStr != "" {
						if startedTime, err := time.Parse(time.RFC3339, startedStr); err == nil {
							duration = FormatDuration(completedTime.Sub(startedTime))
						}
					}
				}
			}

			table.Append([]string{id, status, created, started, completed, duration})
		}
	}

	table.Render()
	return nil
}

// getString safely gets a string value from a map
func getString(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists && val != nil {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// ScheduleTable creates a formatted table for schedules
func (p *Printer) ScheduleTable(schedules []interface{}) error {
	if len(schedules) == 0 {
		fmt.Println("No schedules found")
		return nil
	}

	table := tablewriter.NewWriter(p.Writer)
	table.SetHeader([]string{"ID", "Name", "Cron", "Enabled", "Next Run", "Last Run"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	if !p.NoColor {
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		)
	}

	for _, schedule := range schedules {
		if scheduleMap, ok := schedule.(map[string]interface{}); ok {
			id := fmt.Sprintf("%.8s", getString(scheduleMap, "id"))
			name := getString(scheduleMap, "name")
			cronExpr := getString(scheduleMap, "cron")
			
			enabled := getString(scheduleMap, "enabled")
			if enabled == "true" {
				enabled = p.FormatStatus("enabled")
			} else {
				enabled = p.FormatStatus("disabled")
			}
			
			nextRun := getString(scheduleMap, "next_run")
			if nextRun != "" && nextRun != "-" {
				if nextTime, err := time.Parse(time.RFC3339, nextRun); err == nil {
					nextRun = nextTime.Format("2006-01-02 15:04:05")
				}
			}
			
			lastRun := getString(scheduleMap, "last_run")
			if lastRun != "" && lastRun != "-" {
				if lastTime, err := time.Parse(time.RFC3339, lastRun); err == nil {
					lastRun = lastTime.Format("2006-01-02 15:04:05")
				}
			} else {
				lastRun = "-"
			}

			table.Append([]string{id, name, cronExpr, enabled, nextRun, lastRun})
		}
	}

	table.Render()
	return nil
}

// PrintScheduleDetails prints detailed information about a schedule
func (p *Printer) PrintScheduleDetails(schedule interface{}) {
	fmt.Fprintf(p.Writer, "\n%s\n", p.FormatHeader("Schedule Details"))
	
	if scheduleMap, ok := schedule.(map[string]interface{}); ok {
		p.PrintKeyValue("ID", getString(scheduleMap, "id"))
		p.PrintKeyValue("Name", getString(scheduleMap, "name"))
		p.PrintKeyValue("Description", getString(scheduleMap, "description"))
		p.PrintKeyValue("Cron Expression", getString(scheduleMap, "cron_expression"))
		
		enabled := getString(scheduleMap, "enabled")
		if enabled == "true" {
			p.PrintKeyValue("Status", p.FormatStatus("enabled"))
		} else {
			p.PrintKeyValue("Status", p.FormatStatus("disabled"))
		}
		
		p.PrintKeyValue("Created At", getString(scheduleMap, "created_at"))
		p.PrintKeyValue("Updated At", getString(scheduleMap, "updated_at"))
		p.PrintKeyValue("Next Run", getString(scheduleMap, "next_run"))
		p.PrintKeyValue("Last Run", getString(scheduleMap, "last_run"))
		
		// Print playbook info
		if playbook := scheduleMap["playbook"]; playbook != nil {
			fmt.Fprintf(p.Writer, "\n%s\n", p.FormatHeader("Playbook"))
			playbookJSON, _ := json.MarshalIndent(playbook, "", "  ")
			fmt.Fprintln(p.Writer, string(playbookJSON))
		}
		
		// Print context if not empty
		if context := scheduleMap["context"]; context != nil {
			if contextMap, ok := context.(map[string]interface{}); ok && len(contextMap) > 0 {
				fmt.Fprintf(p.Writer, "\n%s\n", p.FormatHeader("Context"))
				contextJSON, _ := json.MarshalIndent(context, "", "  ")
				fmt.Fprintln(p.Writer, string(contextJSON))
			}
		}
	}
}

// PrintScheduleStats prints schedule statistics
func (p *Printer) PrintScheduleStats(stats map[string]interface{}) {
	fmt.Fprintf(p.Writer, "\n%s\n", p.FormatHeader("Schedule Statistics"))
	
	if total, ok := stats["total"].(float64); ok {
		p.PrintKeyValue("Total Schedules", fmt.Sprintf("%.0f", total))
	}
	
	if enabled, ok := stats["enabled"].(float64); ok {
		p.PrintKeyValue("Enabled", fmt.Sprintf("%.0f", enabled))
	}
	
	if disabled, ok := stats["disabled"].(float64); ok {
		p.PrintKeyValue("Disabled", fmt.Sprintf("%.0f", disabled))
	}
	
	if running, ok := stats["running"].(float64); ok {
		p.PrintKeyValue("Currently Running", fmt.Sprintf("%.0f", running))
	}
	
	if failed, ok := stats["failed_last_run"].(float64); ok {
		p.PrintKeyValue("Failed Last Run", fmt.Sprintf("%.0f", failed))
	}
	
	if nextScheduled, ok := stats["next_scheduled"].(string); ok && nextScheduled != "" {
		p.PrintKeyValue("Next Scheduled", nextScheduled)
	}
}

// FormatHeader formats a header string
func (p *Printer) FormatHeader(header string) string {
	if p.NoColor {
		return fmt.Sprintf("=== %s ===", header)
	}
	return color.New(color.Bold, color.FgCyan).Sprintf("=== %s ===", header)
}

// PrintKeyValue prints a key-value pair
func (p *Printer) PrintKeyValue(key, value string) {
	if p.NoColor {
		fmt.Fprintf(p.Writer, "%s: %s\n", key, value)
	} else {
		fmt.Fprintf(p.Writer, "%s: %s\n", 
			color.New(color.Bold).Sprint(key),
			value)
	}
}

// PrintYAML outputs data as YAML
func (p *Printer) PrintYAML(data interface{}) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	_, err = p.Writer.Write(yamlData)
	return err
}

// Success prints a success message (simplified name)
func (p *Printer) Success(message string) {
	p.PrintSuccess(message)
}
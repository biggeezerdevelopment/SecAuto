package cmd

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"secauto-cli/pkg/client"
	"secauto-cli/pkg/output"
)

// clusterCmd represents the cluster command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage cluster operations",
	Long:  `View and manage SecAuto cluster status and operations.`,
}

// clusterStatusCmd represents the cluster status command
var clusterStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cluster status",
	Long:  `Display current cluster status including node information and health.`,
	RunE:  clusterStatus,
}

func init() {
	rootCmd.AddCommand(clusterCmd)
	clusterCmd.AddCommand(clusterStatusCmd)
}

func clusterStatus(cmd *cobra.Command, args []string) error {
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
		s.Suffix = " Fetching cluster status..."
	}

	status, err := apiClient.GetClusterStatus()
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to get cluster status: %v", err)
	}

	// Display cluster status in a readable format
	fmt.Printf("Cluster Status:\n")
	fmt.Printf("  Node ID: %s\n", status.NodeID)
	fmt.Printf("  Status: %s\n", printer.FormatStatus(status.Status))
	fmt.Printf("  Total Nodes: %d\n", status.TotalNodes)
	fmt.Printf("  Active Nodes: %d\n", status.ActiveNodes)
	
	// Health indicator
	healthPercentage := float64(status.ActiveNodes) / float64(status.TotalNodes) * 100
	fmt.Printf("  Health: %.1f%%\n", healthPercentage)

	if status.ActiveNodes < status.TotalNodes {
		unavailableNodes := status.TotalNodes - status.ActiveNodes
		printer.PrintWarning(fmt.Sprintf("%d node(s) unavailable", unavailableNodes))
	} else {
		printer.PrintSuccess("All nodes are active")
	}

	return nil
}
package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"secauto-cli/pkg/client"
	"secauto-cli/pkg/output"
)

// cacheCmd represents the cache command
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage cache operations",
	Long:  `Perform cache operations like get, set, delete, and view statistics.`,
}

// cacheStatsCmd represents the cache stats command
var cacheStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show cache statistics",
	Long:  `Display cache statistics including hit rate, memory usage, and key count.`,
	RunE:  cacheStats,
}

// cacheGetCmd represents the cache get command
var cacheGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a value from cache",
	Long:  `Retrieve a value from the cache by its key.`,
	Args:  cobra.ExactArgs(1),
	RunE:  cacheGet,
}

// cacheSetCmd represents the cache set command
var cacheSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a value in cache",
	Long:  `Store a value in the cache with the specified key.`,
	Args:  cobra.ExactArgs(2),
	RunE:  cacheSet,
}

// cacheDeleteCmd represents the cache delete command
var cacheDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a key from cache",
	Long:  `Remove a key and its value from the cache.`,
	Args:  cobra.ExactArgs(1),
	RunE:  cacheDelete,
}

// cacheClearCmd represents the cache clear command
var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all cache entries",
	Long:  `Remove all entries from the cache. This operation cannot be undone.`,
	RunE:  cacheClear,
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheStatsCmd)
	cacheCmd.AddCommand(cacheGetCmd)
	cacheCmd.AddCommand(cacheSetCmd)
	cacheCmd.AddCommand(cacheDeleteCmd)
	cacheCmd.AddCommand(cacheClearCmd)

	// Cache set flags
	cacheSetCmd.Flags().Bool("json", false, "Treat value as JSON")
}

func cacheStats(cmd *cobra.Command, args []string) error {
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
		s.Suffix = " Fetching cache statistics..."
	}

	stats, err := apiClient.GetCacheStats()
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to get cache stats: %v", err)
	}

	// Display stats in a readable format
	fmt.Printf("Cache Statistics:\n")
	fmt.Printf("  Total Keys: %d\n", stats.TotalKeys)
	fmt.Printf("  Total Memory: %d bytes\n", stats.TotalMemory)
	fmt.Printf("  Hit Rate: %.2f%%\n", stats.HitRate*100)
	fmt.Printf("  Miss Rate: %.2f%%\n", stats.MissRate*100)

	return nil
}

func cacheGet(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	key := args[0]

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Fetching cache value..."
	}

	value, err := apiClient.GetCacheKey(key)
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to get cache key '%s': %v", key, err)
	}

	if value == nil {
		fmt.Printf("Key '%s' not found in cache\n", key)
		return nil
	}

	fmt.Printf("Key: %s\n", key)
	fmt.Printf("Value:\n")
	return printer.Print(value)
}

func cacheSet(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	key := args[0]
	valueStr := args[1]

	// Parse value
	var value interface{}
	isJSON, _ := cmd.Flags().GetBool("json")
	
	if isJSON {
		if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
			return fmt.Errorf("failed to parse JSON value: %v", err)
		}
	} else {
		value = valueStr
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Setting cache value..."
	}

	err := apiClient.SetCacheKey(key, value)
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to set cache key '%s': %v", key, err)
	}

	printer.PrintSuccess(fmt.Sprintf("Cache key '%s' set successfully", key))
	return nil
}

func cacheDelete(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	key := args[0]

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Deleting cache key..."
	}

	err := apiClient.DeleteCacheKey(key)
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to delete cache key '%s': %v", key, err)
	}

	printer.PrintSuccess(fmt.Sprintf("Cache key '%s' deleted successfully", key))
	return nil
}

func cacheClear(cmd *cobra.Command, args []string) error {
	config := GetGlobalConfig()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	printer := output.NewPrinter(config.Output, config.NoColor)
	apiClient := client.NewClient(config.Server, config.APIKey)

	// Confirm operation
	fmt.Print("Are you sure you want to clear all cache entries? (y/N): ")
	var response string
	fmt.Scanln(&response)
	
	if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
		fmt.Println("Operation cancelled")
		return nil
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if !printer.NoColor {
		s.Start()
		s.Suffix = " Clearing cache..."
	}

	err := apiClient.ClearCache()
	
	if !printer.NoColor {
		s.Stop()
	}

	if err != nil {
		return fmt.Errorf("failed to clear cache: %v", err)
	}

	printer.PrintSuccess("Cache cleared successfully")
	return nil
}
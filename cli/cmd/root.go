package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"secauto-cli/pkg/config"
)

var cfgFile string
var globalConfig *config.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "secauto",
	Short: "SecAuto CLI - Command line interface for SecAuto SOAR platform",
	Long: `SecAuto CLI provides a command line interface for interacting with the SecAuto
Security Orchestration, Automation & Response (SOAR) platform.

Features:
  - Execute playbooks synchronously and asynchronously
  - Manage jobs and monitor execution status
  - Handle cache operations and statistics
  - Manage integrations and automations
  - Monitor cluster status
  - Upload and list playbooks`,
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.secauto-cli.yaml)")
	rootCmd.PersistentFlags().String("server", "", "SecAuto server URL (e.g., http://localhost:9090)")
	rootCmd.PersistentFlags().String("api-key", "", "API key for authentication")
	rootCmd.PersistentFlags().String("output", "table", "Output format (table, json, yaml)")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")

	// Bind flags to viper
	viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))
	viper.BindPFlag("api-key", rootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".secauto-cli" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".secauto-cli")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("verbose") {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}

	// Initialize global config
	var err error
	globalConfig, err = config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
}

// getConfigDir returns the config directory path
func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "secauto-cli"), nil
}

// GetGlobalConfig returns the global configuration
func GetGlobalConfig() *config.Config {
	return globalConfig
}
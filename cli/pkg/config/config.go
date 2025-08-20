package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the CLI configuration
type Config struct {
	Server     string            `mapstructure:"server"`
	APIKey     string            `mapstructure:"api_key"`
	Output     string            `mapstructure:"output"`
	NoColor    bool              `mapstructure:"no_color"`
	Verbose    bool              `mapstructure:"verbose"`
	Profiles   map[string]*Profile `mapstructure:"profiles"`
	Current    string            `mapstructure:"current"`
}

// Profile represents a server profile
type Profile struct {
	Name        string `mapstructure:"name"`
	Server      string `mapstructure:"server"`
	APIKey      string `mapstructure:"api_key"`
	Description string `mapstructure:"description"`
}

// LoadConfig loads configuration from file and environment
func LoadConfig() (*Config, error) {
	config := &Config{
		Server:  "http://localhost:9090",
		Output:  "table",
		NoColor: false,
		Verbose: false,
		Profiles: make(map[string]*Profile),
		Current: "default",
	}

	// Override with viper values
	if viper.IsSet("server") {
		config.Server = viper.GetString("server")
	}
	if viper.IsSet("api-key") {
		config.APIKey = viper.GetString("api-key")
	}
	if viper.IsSet("output") {
		config.Output = viper.GetString("output")
	}
	if viper.IsSet("no-color") {
		config.NoColor = viper.GetBool("no-color")
	}
	if viper.IsSet("verbose") {
		config.Verbose = viper.GetBool("verbose")
	}

	// Load profiles from config file
	if viper.IsSet("profiles") {
		profilesData := viper.GetStringMap("profiles")
		config.Profiles = make(map[string]*Profile)
		for name, data := range profilesData {
			if profileMap, ok := data.(map[string]interface{}); ok {
				profile := &Profile{}
				if server, exists := profileMap["server"]; exists {
					if serverStr, ok := server.(string); ok {
						profile.Server = serverStr
					}
				}
				if apiKey, exists := profileMap["api_key"]; exists {
					if apiKeyStr, ok := apiKey.(string); ok {
						profile.APIKey = apiKeyStr
					}
				}
				if desc, exists := profileMap["description"]; exists {
					if descStr, ok := desc.(string); ok {
						profile.Description = descStr
					}
				}
				profile.Name = name
				config.Profiles[name] = profile
			}
		}
	}
	if viper.IsSet("current") {
		config.Current = viper.GetString("current")
	}

	// Apply current profile if set
	if profile, exists := config.Profiles[config.Current]; exists {
		if config.Server == "http://localhost:9090" {
			config.Server = profile.Server
		}
		if config.APIKey == "" {
			config.APIKey = profile.APIKey
		}
	}

	return config, nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configFile := filepath.Join(configDir, "config.yaml")
	
	viper.Set("server", c.Server)
	viper.Set("api_key", c.APIKey)
	viper.Set("output", c.Output)
	viper.Set("no_color", c.NoColor)
	viper.Set("verbose", c.Verbose)
	viper.Set("profiles", c.Profiles)
	viper.Set("current", c.Current)

	return viper.WriteConfigAs(configFile)
}

// GetCurrentProfile returns the current active profile
func (c *Config) GetCurrentProfile() (*Profile, bool) {
	profile, exists := c.Profiles[c.Current]
	return profile, exists
}

// AddProfile adds a new profile
func (c *Config) AddProfile(name string, profile *Profile) {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*Profile)
	}
	profile.Name = name
	c.Profiles[name] = profile
}

// RemoveProfile removes a profile
func (c *Config) RemoveProfile(name string) error {
	if name == "default" {
		return fmt.Errorf("cannot remove default profile")
	}
	if _, exists := c.Profiles[name]; !exists {
		return fmt.Errorf("profile '%s' does not exist", name)
	}
	delete(c.Profiles, name)
	if c.Current == name {
		c.Current = "default"
	}
	return nil
}

// SetCurrentProfile sets the current active profile
func (c *Config) SetCurrentProfile(name string) error {
	if _, exists := c.Profiles[name]; !exists && name != "default" {
		return fmt.Errorf("profile '%s' does not exist", name)
	}
	c.Current = name
	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server == "" {
		return fmt.Errorf("server URL is required")
	}
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// getConfigDir returns the config directory path
func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "secauto-cli"), nil
}
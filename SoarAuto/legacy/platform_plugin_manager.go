package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// PlatformPluginManager manages plugins across different platforms
type PlatformPluginManager struct {
	platforms map[string]*PluginManager
	config    map[string]PlatformConfig
	mutex     sync.RWMutex
	logger    *StructuredLogger
}

// NewPlatformPluginManager creates a new platform-aware plugin manager
func NewPlatformPluginManager(config *Config) (*PlatformPluginManager, error) {
	ppm := &PlatformPluginManager{
		platforms: make(map[string]*PluginManager),
		config:    make(map[string]PlatformConfig),
		logger:    logger,
	}

	// Initialize platform configurations
	if err := ppm.initializePlatforms(config); err != nil {
		return nil, fmt.Errorf("failed to initialize platforms: %v", err)
	}

	ppm.logger.Info("Platform plugin manager initialized", map[string]interface{}{
		"component": "platform_plugin_manager",
		"platforms": len(ppm.platforms),
	})

	return ppm, nil
}

// initializePlatforms sets up platform-specific plugin managers
func (ppm *PlatformPluginManager) initializePlatforms(config *Config) error {
	for platformName, platformConfig := range config.Plugins.Platforms {
		if !platformConfig.Enabled {
			ppm.logger.Info("Platform disabled", map[string]interface{}{
				"component": "platform_plugin_manager",
				"platform":  platformName,
			})
			continue
		}

		// Create platform directory if it doesn't exist
		if err := os.MkdirAll(platformConfig.Directory, 0755); err != nil {
			return fmt.Errorf("failed to create platform directory %s: %v", platformName, err)
		}

		// Convert platform config to plugin manager config
		pmConfig := map[string]interface{}{
			"enabled":              platformConfig.Enabled,
			"hot_reload":           config.Plugins.HotReload,
			"reload_interval":      config.Plugins.ReloadInterval,
			"supported_types":      config.Plugins.SupportedTypes,
			"max_plugins":          config.Plugins.MaxPlugins,
			"plugin_timeout":       platformConfig.Timeout,
			"sandbox_mode":         platformConfig.SandboxMode,
			"allow_executables":    true,
			"allow_python":         true,
			"allow_go_plugins":     true,
			"plugin_validation":    config.Plugins.PluginValidation,
			"plugin_logging":       config.Plugins.PluginLogging,
			"platform":             platformName,
			"interpreter":          platformConfig.Interpreter,
			"build_mode":           platformConfig.BuildMode,
			"max_memory":           platformConfig.MaxMemory,
			"allow_network_access": platformConfig.AllowNetworkAccess,
			"allow_file_access":    platformConfig.AllowFileAccess,
			"venv_path":            config.GetVenvPath(),
		}

		// Create plugin manager for this platform
		pm, err := NewPluginManager(platformConfig.Directory, pmConfig)
		if err != nil {
			ppm.logger.Error("Failed to create plugin manager for platform", map[string]interface{}{
				"component": "platform_plugin_manager",
				"platform":  platformName,
				"error":     err.Error(),
			})
			continue
		}

		ppm.platforms[platformName] = pm
		ppm.config[platformName] = platformConfig

		ppm.logger.Info("Platform plugin manager created", map[string]interface{}{
			"component": "platform_plugin_manager",
			"platform":  platformName,
			"directory": platformConfig.Directory,
		})
	}

	return nil
}

// GetPlugin retrieves a plugin by name across all platforms
func (ppm *PlatformPluginManager) GetPlugin(name string) (interface{}, bool) {
	ppm.mutex.RLock()
	defer ppm.mutex.RUnlock()

	for platformName, pm := range ppm.platforms {
		if plugin, exists := pm.GetPlugin(name); exists {
			ppm.logger.Debug("Plugin found", map[string]interface{}{
				"component": "platform_plugin_manager",
				"plugin":    name,
				"platform":  platformName,
			})
			return plugin, true
		}
	}

	return nil, false
}

// GetPluginByName retrieves a plugin by name with platform information
func (ppm *PlatformPluginManager) GetPluginByName(pluginName string) (interface{}, bool) {
	return ppm.GetPlugin(pluginName)
}

// GetPluginsByType retrieves all plugins of a specific type across all platforms
func (ppm *PlatformPluginManager) GetPluginsByType(pluginType PluginType) []interface{} {
	ppm.mutex.RLock()
	defer ppm.mutex.RUnlock()

	var plugins []interface{}
	for platformName, pm := range ppm.platforms {
		platformPlugins := pm.GetPluginsByType(pluginType)
		plugins = append(plugins, platformPlugins...)

		ppm.logger.Debug("Retrieved plugins by type", map[string]interface{}{
			"component": "platform_plugin_manager",
			"platform":  platformName,
			"type":      string(pluginType),
			"count":     len(platformPlugins),
		})
	}

	return plugins
}

// GetAllPlugins retrieves all plugins across all platforms
func (ppm *PlatformPluginManager) GetAllPlugins() map[string]interface{} {
	ppm.mutex.RLock()
	defer ppm.mutex.RUnlock()

	allPlugins := make(map[string]interface{})
	for platformName, pm := range ppm.platforms {
		platformPlugins := pm.GetAllPlugins()
		for name, plugin := range platformPlugins {
			allPlugins[name] = plugin
		}

		ppm.logger.Debug("Retrieved all plugins for platform", map[string]interface{}{
			"component": "platform_plugin_manager",
			"platform":  platformName,
			"count":     len(platformPlugins),
		})
	}

	return allPlugins
}

// GetPluginInfo retrieves plugin information across all platforms
func (ppm *PlatformPluginManager) GetPluginInfo() map[string]PluginInfo {
	ppm.mutex.RLock()
	defer ppm.mutex.RUnlock()

	allPluginInfo := make(map[string]PluginInfo)
	for platformName, pm := range ppm.platforms {
		platformPluginInfo := pm.GetPluginInfo()
		for name, info := range platformPluginInfo {
			// Add platform information to plugin info
			info.Platform = platformName
			info.Runtime = ppm.getRuntimeForPlatform(platformName)
			info.PlatformInfo = ppm.getPlatformInfo(platformName)
			allPluginInfo[name] = info
		}
	}

	return allPluginInfo
}

// ExecutePlugin executes a plugin by name across all platforms
func (ppm *PlatformPluginManager) ExecutePlugin(name string, params map[string]interface{}) (interface{}, error) {
	ppm.mutex.RLock()
	defer ppm.mutex.RUnlock()

	for platformName, pm := range ppm.platforms {
		if _, exists := pm.GetPlugin(name); exists {
			ppm.logger.Info("Executing plugin", map[string]interface{}{
				"component": "platform_plugin_manager",
				"plugin":    name,
				"platform":  platformName,
			})

			// Use platform-specific timeout
			platformConfig := ppm.config[platformName]
			if platformConfig.Timeout > 0 {
				// TODO: Implement timeout mechanism
			}

			return pm.ExecutePlugin(name, params)
		}
	}

	return nil, fmt.Errorf("plugin not found: %s", name)
}

// GetPluginsByPlatform retrieves all plugins for a specific platform
func (ppm *PlatformPluginManager) GetPluginsByPlatform(platformName string) map[string]interface{} {
	ppm.mutex.RLock()
	defer ppm.mutex.RUnlock()

	if pm, exists := ppm.platforms[platformName]; exists {
		return pm.GetAllPlugins()
	}

	return make(map[string]interface{})
}

// GetPlatformInfo retrieves information about a specific platform
func (ppm *PlatformPluginManager) GetPlatformInfo(platformName string) (PlatformConfig, bool) {
	ppm.mutex.RLock()
	defer ppm.mutex.RUnlock()

	config, exists := ppm.config[platformName]
	return config, exists
}

// GetEnabledPlatforms returns a list of enabled platforms
func (ppm *PlatformPluginManager) GetEnabledPlatforms() []string {
	ppm.mutex.RLock()
	defer ppm.mutex.RUnlock()

	var platforms []string
	for platformName := range ppm.platforms {
		platforms = append(platforms, platformName)
	}

	return platforms
}

// Close closes all platform plugin managers
func (ppm *PlatformPluginManager) Close() error {
	ppm.mutex.Lock()
	defer ppm.mutex.Unlock()

	var errors []string
	for platformName, pm := range ppm.platforms {
		if err := pm.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("platform %s: %v", platformName, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing platform managers: %s", strings.Join(errors, "; "))
	}

	return nil
}

// getRuntimeForPlatform returns the runtime for a given platform
func (ppm *PlatformPluginManager) getRuntimeForPlatform(platformName string) string {
	switch platformName {
	case "python":
		return "python3"
	case "go":
		return "go"
	case "windows", "linux":
		return "executable"
	default:
		return "unknown"
	}
}

// getPlatformInfo returns platform-specific information
func (ppm *PlatformPluginManager) getPlatformInfo(platformName string) PlatformInfo {
	info := PlatformInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}

	switch platformName {
	case "python":
		info.Dependencies = []string{"python3"}
		info.Requirements = map[string]string{
			"python_version": ">=3.8",
		}
	case "go":
		info.Dependencies = []string{"go"}
		info.Requirements = map[string]string{
			"go_version": ">=1.19",
		}
	case "windows":
		info.Dependencies = []string{}
		info.Requirements = map[string]string{
			"os": "windows",
		}
	case "linux":
		info.Dependencies = []string{}
		info.Requirements = map[string]string{
			"os": "linux",
		}
	}

	return info
}

// DetectPlatformForFile determines the platform for a given file
func (ppm *PlatformPluginManager) DetectPlatformForFile(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Check platform-specific extensions
	for platformName, config := range ppm.config {
		for _, supportedExt := range config.SupportedExtensions {
			if ext == supportedExt {
				return platformName
			}
		}
	}

	// Fallback based on file extension
	switch ext {
	case ".py":
		return "python"
	case ".go", ".so":
		return "go"
	case ".exe":
		if runtime.GOOS == "windows" {
			return "windows"
		}
		return "go"
	default:
		return "unknown"
	}
}

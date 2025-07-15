package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// PluginType represents the type of plugin
type PluginType string

const (
	PluginTypeAutomation  PluginType = "automation"
	PluginTypePlaybook    PluginType = "playbook"
	PluginTypeIntegration PluginType = "integration"
	PluginTypeValidator   PluginType = "validator"
)

// PluginStatus represents the status of a plugin
type PluginStatus string

const (
	PluginStatusLoaded    PluginStatus = "loaded"
	PluginStatusError     PluginStatus = "error"
	PluginStatusDisabled  PluginStatus = "disabled"
	PluginStatusReloading PluginStatus = "reloading"
)

// PluginInfo contains metadata about a plugin
type PluginInfo struct {
	Name        string       `json:"name"`
	Type        PluginType   `json:"type"`
	Platform    string       `json:"platform"` // "windows", "linux", "python", "go"
	Runtime     string       `json:"runtime"`  // "python3", "go", "executable"
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Author      string       `json:"author"`
	Status      PluginStatus `json:"status"`
	Error       string       `json:"error,omitempty"`
	LoadedAt    time.Time    `json:"loaded_at"`
	LastReload  time.Time    `json:"last_reload,omitempty"`
	Config      interface{}  `json:"config,omitempty"`

	// Platform-specific metadata
	PlatformInfo PlatformInfo `json:"platform_info,omitempty"`
}

// PluginInterface defines the interface that all plugins must implement
type PluginInterface interface {
	// GetInfo returns plugin metadata
	GetInfo() PluginInfo

	// Initialize is called when the plugin is loaded
	Initialize(config map[string]interface{}) error

	// Execute runs the plugin with given parameters
	Execute(params map[string]interface{}) (interface{}, error)

	// Cleanup is called when the plugin is unloaded
	Cleanup() error
}

// AutomationPlugin extends PluginInterface for automation plugins
type AutomationPlugin interface {
	PluginInterface
	// GetSupportedOperations returns list of operations this automation supports
	GetSupportedOperations() []string
}

// PlaybookPlugin extends PluginInterface for playbook plugins
type PlaybookPlugin interface {
	PluginInterface
	// ValidatePlaybook validates a playbook structure
	ValidatePlaybook(playbook []interface{}) error
	// ExecutePlaybook executes a playbook
	ExecutePlaybook(playbook []interface{}, context map[string]interface{}) ([]interface{}, error)
}

// IntegrationPlugin extends PluginInterface for integration plugins
type IntegrationPlugin interface {
	PluginInterface
	// GetSupportedIntegrations returns list of integrations this plugin supports
	GetSupportedIntegrations() []string
	// TestConnection tests the integration connection
	TestConnection(config map[string]interface{}) error
}

// ValidatorPlugin extends PluginInterface for validator plugins
type ValidatorPlugin interface {
	PluginInterface
	// Validate validates data according to plugin rules
	Validate(data interface{}) (bool, []string, error)
}

// PluginManager manages the loading, unloading, and hot-reloading of plugins
type PluginManager struct {
	pluginsDir  string
	plugins     map[string]interface{}
	pluginInfos map[string]PluginInfo
	watcher     *fsnotify.Watcher
	mutex       sync.RWMutex
	reloadChan  chan string
	stopChan    chan struct{}
	config      map[string]interface{}
	logger      *StructuredLogger
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(pluginsDir string, config map[string]interface{}) (*PluginManager, error) {
	pm := &PluginManager{
		pluginsDir:  pluginsDir,
		plugins:     make(map[string]interface{}),
		pluginInfos: make(map[string]PluginInfo),
		reloadChan:  make(chan string, 100),
		stopChan:    make(chan struct{}),
		config:      config,
		logger:      logger,
	}

	// Create plugins directory if it doesn't exist
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugins directory: %v", err)
	}

	// Initialize file watcher for hot-reload
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %v", err)
	}
	pm.watcher = watcher

	// Start hot-reload goroutine
	go pm.hotReloadWorker()

	// Load existing plugins
	if err := pm.loadAllPlugins(); err != nil {
		return nil, fmt.Errorf("failed to load plugins: %v", err)
	}

	// Start file watching
	if err := pm.startFileWatching(); err != nil {
		return nil, fmt.Errorf("failed to start file watching: %v", err)
	}

	pm.logger.Info("Plugin manager initialized", map[string]interface{}{
		"component":      "plugin_manager",
		"plugins_dir":    pluginsDir,
		"loaded_plugins": len(pm.plugins),
	})

	return pm, nil
}

// loadAllPlugins loads all plugins from the plugins directory
func (pm *PluginManager) loadAllPlugins() error {
	return filepath.WalkDir(pm.pluginsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Check if it's a plugin file
		if pm.isPluginFile(path) {
			if err := pm.loadPlugin(path); err != nil {
				pm.logger.Error("Failed to load plugin", map[string]interface{}{
					"component":   "plugin_manager",
					"plugin_path": path,
					"error":       err.Error(),
				})
			}
		}

		return nil
	})
}

// isPluginFile checks if a file is a plugin file
func (pm *PluginManager) isPluginFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	// On Windows, only support Python plugins and Go executables
	// Go source files (.go) and plugins (.so) are not supported on Windows
	if runtime.GOOS == "windows" {
		return ext == ".py" || ext == ".exe"
	}
	return ext == ".py" || ext == ".exe" || ext == ".go" || ext == ".so"
}

// loadPlugin loads a single plugin
func (pm *PluginManager) loadPlugin(pluginPath string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pluginName := filepath.Base(pluginPath)
	ext := strings.ToLower(filepath.Ext(pluginPath))

	var pluginInstance interface{}
	var err error

	switch ext {
	case ".so":
		if runtime.GOOS == "windows" {
			return fmt.Errorf("go plugins (.so) are not supported on Windows")
		}
		pluginInstance, err = pm.loadGoPlugin(pluginPath)
	case ".py":
		pluginInstance, err = pm.loadPythonPlugin(pluginPath)
	case ".exe":
		pluginInstance, err = pm.loadGoExecutablePlugin(pluginPath)
	case ".go":
		if runtime.GOOS == "windows" {
			return fmt.Errorf("go source plugins are not supported on Windows, use .exe files instead")
		}
		// For Go source files, we'll compile them first
		pluginInstance, err = pm.loadGoSourcePlugin(pluginPath)
	default:
		return fmt.Errorf("unsupported plugin type: %s", ext)
	}

	if err != nil {
		pm.updatePluginInfo(pluginName, PluginInfo{
			Name:     pluginName,
			Status:   PluginStatusError,
			Error:    err.Error(),
			LoadedAt: time.Now(),
		})
		return err
	}

	// Initialize the plugin
	if err := pm.initializePlugin(pluginInstance, pluginName); err != nil {
		pm.updatePluginInfo(pluginName, PluginInfo{
			Name:     pluginName,
			Status:   PluginStatusError,
			Error:    err.Error(),
			LoadedAt: time.Now(),
		})
		return err
	}

	// Get the plugin's actual name from its info
	plugin, ok := pluginInstance.(PluginInterface)
	if !ok {
		return fmt.Errorf("plugin does not implement PluginInterface")
	}

	info := plugin.GetInfo()
	actualPluginName := info.Name

	// Store the plugin by its actual name, not the filename
	pm.plugins[actualPluginName] = pluginInstance
	pm.logger.Info("Plugin loaded successfully", map[string]interface{}{
		"component":   "plugin_manager",
		"plugin_name": actualPluginName,
		"plugin_path": pluginPath,
		"filename":    pluginName,
	})

	return nil
}

// loadGoPlugin loads a compiled Go plugin (.so file)
func (pm *PluginManager) loadGoPlugin(pluginPath string) (interface{}, error) {
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %v", err)
	}

	// Look for the plugin symbol
	sym, err := p.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("plugin symbol not found: %v", err)
	}

	pluginInstance, ok := sym.(PluginInterface)
	if !ok {
		return nil, fmt.Errorf("plugin does not implement PluginInterface")
	}

	return pluginInstance, nil
}

// loadGoSourcePlugin compiles and loads a Go source plugin
func (pm *PluginManager) loadGoSourcePlugin(pluginPath string) (interface{}, error) {
	// Create temporary directory for compilation
	tempDir, err := os.MkdirTemp("", "secauto_plugin_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Copy plugin source to temp directory
	pluginName := strings.TrimSuffix(filepath.Base(pluginPath), ".go")
	tempPluginPath := filepath.Join(tempDir, pluginName+".go")

	// Read and modify the source to make it a proper plugin
	source, err := os.ReadFile(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin source: %v", err)
	}

	// Add plugin wrapper if needed
	pluginSource := pm.wrapGoSource(string(source), pluginName)

	if err := os.WriteFile(tempPluginPath, []byte(pluginSource), 0644); err != nil {
		return nil, fmt.Errorf("failed to write temp plugin: %v", err)
	}

	// Create go.mod for the plugin
	goModContent := fmt.Sprintf(`module %s
go 1.23.1
`, pluginName)

	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to create go.mod: %v", err)
	}

	// Compile the plugin
	outputPath := filepath.Join(tempDir, pluginName+".so")
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", outputPath, tempPluginPath)
	cmd.Dir = tempDir

	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to compile plugin: %v, output: %s", err, string(output))
	}

	// Load the compiled plugin
	return pm.loadGoPlugin(outputPath)
}

// wrapGoSource wraps Go source code to make it a proper plugin
func (pm *PluginManager) wrapGoSource(source, pluginName string) string {
	// Simple wrapper - in a real implementation, you'd want more sophisticated parsing
	wrapper := fmt.Sprintf(`
package main

import (
	"time"
)

// Plugin is the main plugin instance
var Plugin = &%sPlugin{}

type %sPlugin struct {
	info PluginInfo
}

func (p *%sPlugin) GetInfo() PluginInfo {
	return p.info
}

func (p *%sPlugin) Initialize(config map[string]interface{}) error {
	p.info = PluginInfo{
		Name:     "%s",
		Type:     PluginTypeAutomation,
		Version:  "1.0.0",
		Status:   PluginStatusLoaded,
		LoadedAt: time.Now(),
	}
	return nil
}

func (p *%sPlugin) Execute(params map[string]interface{}) (interface{}, error) {
	// Plugin implementation goes here
	return params, nil
}

func (p *%sPlugin) Cleanup() error {
	return nil
}

%s
`, pluginName, pluginName, pluginName, pluginName, pluginName, pluginName, pluginName, source)

	return wrapper
}

// loadGoExecutablePlugin loads a Go executable as a plugin
func (pm *PluginManager) loadGoExecutablePlugin(pluginPath string) (interface{}, error) {
	// Create a Go executable plugin wrapper
	execWrapper := &GoExecutablePluginWrapper{
		execPath: pluginPath,
		manager:  pm,
	}

	return execWrapper, nil
}

// GoExecutablePluginWrapper wraps Go executables for plugin integration
type GoExecutablePluginWrapper struct {
	execPath string
	manager  *PluginManager
	info     PluginInfo
}

func (gew *GoExecutablePluginWrapper) GetInfo() PluginInfo {
	return gew.info
}

func (gew *GoExecutablePluginWrapper) Initialize(config map[string]interface{}) error {
	// Execute Go binary to get info
	cmd := exec.Command(gew.execPath, "info")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get plugin info: %v", err)
	}

	if err := json.Unmarshal(output, &gew.info); err != nil {
		return fmt.Errorf("failed to parse plugin info: %v", err)
	}

	gew.info.Status = PluginStatusLoaded
	gew.info.LoadedAt = time.Now()

	return nil
}

func (gew *GoExecutablePluginWrapper) Execute(params map[string]interface{}) (interface{}, error) {
	// Execute Go binary with parameters
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %v", err)
	}

	cmd := exec.Command(gew.execPath, "execute", string(paramsJSON))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute plugin: %v", err)
	}

	var result interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse plugin result: %v", err)
	}

	return result, nil
}

func (gew *GoExecutablePluginWrapper) Cleanup() error {
	// Execute cleanup if needed
	cmd := exec.Command(gew.execPath, "cleanup")
	return cmd.Run()
}

// loadPythonPlugin loads a Python plugin
func (pm *PluginManager) loadPythonPlugin(pluginPath string) (interface{}, error) {
	// Create a Python plugin wrapper
	pythonWrapper := &PythonPluginWrapper{
		scriptPath: pluginPath,
		manager:    pm,
	}

	return pythonWrapper, nil
}

// PythonPluginWrapper wraps Python plugins for Go integration
type PythonPluginWrapper struct {
	scriptPath string
	manager    *PluginManager
	info       PluginInfo
}

func (pw *PythonPluginWrapper) GetInfo() PluginInfo {
	return pw.info
}

func (pw *PythonPluginWrapper) Initialize(config map[string]interface{}) error {
	// Get virtual environment path from config
	venvPath, ok := pw.manager.config["venv_path"].(string)
	if !ok {
		return fmt.Errorf("venv_path not found in plugin manager config")
	}

	// Execute Python script to get info using virtual environment
	output, err := RunPythonFromVenv(venvPath, pw.scriptPath, "info")
	if err != nil {
		return fmt.Errorf("failed to get plugin info: %v", err)
	}

	if err := json.Unmarshal(output, &pw.info); err != nil {
		return fmt.Errorf("failed to parse plugin info: %v", err)
	}

	pw.info.Status = PluginStatusLoaded
	pw.info.LoadedAt = time.Now()

	return nil
}

func (pw *PythonPluginWrapper) Execute(params map[string]interface{}) (interface{}, error) {
	// Get virtual environment path from config
	venvPath, ok := pw.manager.config["venv_path"].(string)
	if !ok {
		return nil, fmt.Errorf("venv_path not found in plugin manager config")
	}

	// Execute Python script with parameters using virtual environment
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %v", err)
	}

	output, err := RunPythonFromVenvStdoutOnly(venvPath, pw.scriptPath, "execute", string(paramsJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to execute plugin: %v", err)
	}

	var result interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse plugin result: %v", err)
	}

	return result, nil
}

func (pw *PythonPluginWrapper) Cleanup() error {
	// Get virtual environment path from config
	venvPath, ok := pw.manager.config["venv_path"].(string)
	if !ok {
		return fmt.Errorf("venv_path not found in plugin manager config")
	}

	// Execute cleanup if needed using virtual environment
	_, err := RunPythonFromVenv(venvPath, pw.scriptPath, "cleanup")
	return err
}

// initializePlugin initializes a loaded plugin
func (pm *PluginManager) initializePlugin(pluginInstance interface{}, pluginName string) error {
	plugin, ok := pluginInstance.(PluginInterface)
	if !ok {
		return fmt.Errorf("plugin does not implement PluginInterface")
	}

	// Get plugin configuration
	pluginConfig := pm.getPluginConfig(pluginName)

	// Initialize the plugin
	if err := plugin.Initialize(pluginConfig); err != nil {
		return fmt.Errorf("failed to initialize plugin: %v", err)
	}

	// Get plugin info
	info := plugin.GetInfo()
	info.Status = PluginStatusLoaded
	info.LoadedAt = time.Now()

	pm.updatePluginInfo(pluginName, info)

	return nil
}

// getPluginConfig gets configuration for a specific plugin
func (pm *PluginManager) getPluginConfig(pluginName string) map[string]interface{} {
	if pm.config == nil {
		return make(map[string]interface{})
	}

	if pluginConfig, exists := pm.config[pluginName]; exists {
		if configMap, ok := pluginConfig.(map[string]interface{}); ok {
			return configMap
		}
	}

	return make(map[string]interface{})
}

// updatePluginInfo updates plugin information
func (pm *PluginManager) updatePluginInfo(pluginName string, info PluginInfo) {
	pm.pluginInfos[pluginName] = info
}

// startFileWatching starts watching for file changes
func (pm *PluginManager) startFileWatching() error {
	// Watch the plugins directory
	if err := pm.watcher.Add(pm.pluginsDir); err != nil {
		return fmt.Errorf("failed to watch plugins directory: %v", err)
	}

	// Watch subdirectories
	return filepath.WalkDir(pm.pluginsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return pm.watcher.Add(path)
		}

		return nil
	})
}

// hotReloadWorker handles hot-reloading of plugins
func (pm *PluginManager) hotReloadWorker() {
	for {
		select {
		case pluginPath := <-pm.reloadChan:
			pm.reloadPlugin(pluginPath)
		case event := <-pm.watcher.Events:
			pm.handleFileEvent(event)
		case err := <-pm.watcher.Errors:
			pm.logger.Error("File watcher error", map[string]interface{}{
				"component": "plugin_manager",
				"error":     err.Error(),
			})
		case <-pm.stopChan:
			return
		}
	}
}

// handleFileEvent handles file system events
func (pm *PluginManager) handleFileEvent(event fsnotify.Event) {
	if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
		if pm.isPluginFile(event.Name) {
			// Debounce reload events
			select {
			case pm.reloadChan <- event.Name:
			default:
				// Channel is full, skip this reload
			}
		}
	}
}

// reloadPlugin reloads a specific plugin
func (pm *PluginManager) reloadPlugin(pluginPath string) {
	pluginName := filepath.Base(pluginPath)

	pm.logger.Info("Reloading plugin", map[string]interface{}{
		"component":   "plugin_manager",
		"plugin_name": pluginName,
		"plugin_path": pluginPath,
	})

	// Update status to reloading
	pm.updatePluginInfo(pluginName, PluginInfo{
		Name:       pluginName,
		Status:     PluginStatusReloading,
		LastReload: time.Now(),
	})

	// Unload existing plugin
	pm.unloadPlugin(pluginName)

	// Wait a bit for file system to settle
	time.Sleep(100 * time.Millisecond)

	// Reload the plugin
	if err := pm.loadPlugin(pluginPath); err != nil {
		pm.logger.Error("Failed to reload plugin", map[string]interface{}{
			"component":   "plugin_manager",
			"plugin_name": pluginName,
			"error":       err.Error(),
		})
	} else {
		pm.logger.Info("Plugin reloaded successfully", map[string]interface{}{
			"component":   "plugin_manager",
			"plugin_name": pluginName,
		})
	}
}

// unloadPlugin unloads a specific plugin
func (pm *PluginManager) unloadPlugin(pluginName string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if plugin, exists := pm.plugins[pluginName]; exists {
		if cleanupPlugin, ok := plugin.(PluginInterface); ok {
			if err := cleanupPlugin.Cleanup(); err != nil {
				pm.logger.Error("Failed to cleanup plugin", map[string]interface{}{
					"component":   "plugin_manager",
					"plugin_name": pluginName,
					"error":       err.Error(),
				})
			}
		}
		delete(pm.plugins, pluginName)
	}
}

// GetPlugin returns a plugin by filename
func (pm *PluginManager) GetPlugin(name string) (interface{}, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// GetPluginByName returns a plugin by its actual name (from plugin info)
func (pm *PluginManager) GetPluginByName(pluginName string) (interface{}, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// First try direct lookup (for backward compatibility)
	if plugin, exists := pm.plugins[pluginName]; exists {
		return plugin, exists
	}

	// Search by actual plugin name
	for filename, plugin := range pm.plugins {
		if info, exists := pm.pluginInfos[filename]; exists && info.Name == pluginName {
			return plugin, true
		}
	}

	return nil, false
}

// GetPluginsByType returns all plugins of a specific type
func (pm *PluginManager) GetPluginsByType(pluginType PluginType) []interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var plugins []interface{}
	for name, plugin := range pm.plugins {
		if info, exists := pm.pluginInfos[name]; exists && info.Type == pluginType {
			plugins = append(plugins, plugin)
		}
	}
	return plugins
}

// GetAllPlugins returns all loaded plugins
func (pm *PluginManager) GetAllPlugins() map[string]interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	result := make(map[string]interface{})
	for name, plugin := range pm.plugins {
		result[name] = plugin
	}
	return result
}

// GetPluginInfo returns information about all plugins
func (pm *PluginManager) GetPluginInfo() map[string]PluginInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	result := make(map[string]PluginInfo)
	for name, info := range pm.pluginInfos {
		result[name] = info
	}
	return result
}

// ExecutePlugin executes a plugin with given parameters
func (pm *PluginManager) ExecutePlugin(name string, params map[string]interface{}) (interface{}, error) {
	plugin, exists := pm.GetPluginByName(name)
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	pluginInterface, ok := plugin.(PluginInterface)
	if !ok {
		return nil, fmt.Errorf("plugin does not implement PluginInterface")
	}

	return pluginInterface.Execute(params)
}

// Close closes the plugin manager
func (pm *PluginManager) Close() error {
	close(pm.stopChan)

	// Unload all plugins
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for name, plugin := range pm.plugins {
		if cleanupPlugin, ok := plugin.(PluginInterface); ok {
			if err := cleanupPlugin.Cleanup(); err != nil {
				pm.logger.Error("Failed to cleanup plugin", map[string]interface{}{
					"component":   "plugin_manager",
					"plugin_name": name,
					"error":       err.Error(),
				})
			}
		}
	}

	if pm.watcher != nil {
		return pm.watcher.Close()
	}

	return nil
}

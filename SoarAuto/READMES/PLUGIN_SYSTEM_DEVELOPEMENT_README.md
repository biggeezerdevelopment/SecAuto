# SecAuto Plugin System

---

## 🛠️ Step-by-Step Guide: Building and Using Plugins

### 1. Go Executable Plugins (Windows)

**Write your plugin as a Go program with a `main()` function that supports `info`, `execute`, and `cleanup` commands.**

**Directory structure:**
```
plugins/windows_plugins/example_windows_plugin.go
```

**Example:** See `plugins/windows_plugins/example_windows_plugin.go`

**Build the plugin:**
```powershell
cd plugins/windows_plugins
# Build the plugin as a Windows executable
# (You must have Go installed and in your PATH)
go build -o example_windows_plugin.exe example_windows_plugin.go
# Move/copy the .exe to the main plugins directory for auto-detection
Copy-Item example_windows_plugin.exe ../../plugins/
```

**Usage:**
- The plugin manager will detect `.exe` files in the `plugins/` directory and treat them as plugins.
- The plugin must support the following commands:
  - `info` (prints plugin info as JSON)
  - `execute <json_params>` (executes plugin logic)
  - `cleanup` (performs cleanup)

**Manual test:**
```powershell
# Get plugin info
./example_windows_plugin.exe info
# Execute plugin
./example_windows_plugin.exe execute '{"foo": "bar"}'
# Cleanup
./example_windows_plugin.exe cleanup
```

---

### 2. Go Plugin (.so) (Linux/macOS/FreeBSD)

**Write your plugin as a Go file that implements the PluginInterface and exposes a `var Plugin`.**

**Directory structure:**
```
plugins/go_plugins/example_automation.go
```

**Example:** See `plugins/go_plugins/example_automation.go`

**Build the plugin:**
```bash
cd plugins/go_plugins
# Build as a Go plugin (shared object)
go build -buildmode=plugin -o ../../plugins/example_automation.so example_automation.go
```

**Usage:**
- The plugin manager will detect `.so` files in the `plugins/` directory and load them as Go plugins.
- This only works on Linux, macOS, and FreeBSD (not Windows).

**Manual test:**
```bash
# You cannot run .so plugins directly; they are loaded by the Go app.
```

---

### 3. Python Plugins (All Platforms)

**Write your plugin as a Python script with a class that supports `info`, `execute`, and `cleanup` commands via the command line.**

**Directory structure:**
```
plugins/python_plugins/example_python_plugin.py
```

**Example:** See `plugins/python_plugins/example_python_plugin.py`

**Usage:**
- The plugin manager will detect `.py` files in the `plugins/` directory and treat them as plugins.
- The script must support the following commands:
  - `info` (prints plugin info as JSON)
  - `execute <json_params>` (executes plugin logic)
  - `cleanup` (performs cleanup)

**Manual test:**
```bash
# Get plugin info
python plugins/python_plugins/example_python_plugin.py info
# Execute plugin
python plugins/python_plugins/example_python_plugin.py execute '{"foo": "bar"}'
# Cleanup
python plugins/python_plugins/example_python_plugin.py cleanup
```

---

### 4. General Notes
- Place your built plugin files (`.exe`, `.so`, or `.py`) in the `plugins/` directory for auto-detection.
- The plugin manager will hot-reload plugins on file changes.
- See the API section above for how to interact with plugins via REST endpoints.

---

For more details and examples, see the rest of this README and the `plugins/` directory. 
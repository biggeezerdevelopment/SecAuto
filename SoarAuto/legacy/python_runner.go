package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Run a Python script from the virtual environment
func RunPythonFromVenv(venvPath, scriptPath string, args ...string) ([]byte, error) {
	var pythonExe string
	if runtime.GOOS == "windows" {
		pythonExe = filepath.Join(venvPath, "Scripts", "python.exe")
	} else {
		pythonExe = filepath.Join(venvPath, "bin", "python")
	}
	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command(pythonExe, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("python execution failed: %v, output: %s", err, string(output))
	}
	return output, nil
}

// Run Python script with JSON input via stdin
func RunPythonFromVenvWithJSON(venvPath, scriptPath string, jsonInput interface{}, args ...string) ([]byte, error) {
	var pythonExe string
	if runtime.GOOS == "windows" {
		pythonExe = filepath.Join(venvPath, "Scripts", "python.exe")
	} else {
		pythonExe = filepath.Join(venvPath, "bin", "python")
	}
	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command(pythonExe, cmdArgs...)
	if jsonInput != nil {
		jsonBytes, err := json.Marshal(jsonInput)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON input: %v", err)
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
		}
		go func() {
			defer stdin.Close()
			stdin.Write(jsonBytes)
		}()
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("python execution failed: %v, output: %s", err, string(output))
	}
	return output, nil
}

// Run Python script and parse JSON output
func RunPythonScriptAndParseJSON(venvPath, scriptPath string, args ...string) (*PythonOutput, error) {
	output, err := RunPythonFromVenv(venvPath, scriptPath, args...)
	if err != nil {
		return nil, err
	}
	var result PythonOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON output: %v, raw output: %s", err, string(output))
	}
	return &result, nil
}

// Run Python script with JSON input and parse JSON output
func RunPythonWithJSONInputAndParseOutput(venvPath, scriptPath string, jsonInput interface{}, args ...string) (*PythonOutput, error) {
	output, err := RunPythonFromVenvWithJSON(venvPath, scriptPath, jsonInput, args...)
	if err != nil {
		return nil, err
	}
	var result PythonOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON output: %v, raw output: %s", err, string(output))
	}
	return &result, nil
}

// Run Python code directly (without script file)
func RunPythonCodeFromVenv(venvPath, pythonCode string) ([]byte, error) {
	var pythonExe string
	if runtime.GOOS == "windows" {
		pythonExe = filepath.Join(venvPath, "Scripts", "python.exe")
	} else {
		pythonExe = filepath.Join(venvPath, "bin", "python")
	}
	cmd := exec.Command(pythonExe, "-c", pythonCode)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("python execution failed: %v, output: %s", err, string(output))
	}
	return output, nil
}

// Run Python code directly with JSON input via stdin
func RunPythonCodeFromVenvWithJSON(venvPath, pythonCode string, jsonInput interface{}) ([]byte, error) {
	var pythonExe string
	if runtime.GOOS == "windows" {
		pythonExe = filepath.Join(venvPath, "Scripts", "python.exe")
	} else {
		pythonExe = filepath.Join(venvPath, "bin", "python")
	}
	cmd := exec.Command(pythonExe, "-c", pythonCode)
	if jsonInput != nil {
		jsonBytes, err := json.Marshal(jsonInput)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON input: %v", err)
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
		}
		go func() {
			defer stdin.Close()
			stdin.Write(jsonBytes)
		}()
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("python execution failed: %v, output: %s", err, string(output))
	}
	return output, nil
}

// Run PythonFromVenvStdoutOnly runs a Python script and returns only stdout (stderr is ignored)
func RunPythonFromVenvStdoutOnly(venvPath, scriptPath string, args ...string) ([]byte, error) {
	var pythonExe string
	if runtime.GOOS == "windows" {
		pythonExe = filepath.Join(venvPath, "Scripts", "python.exe")
	} else {
		pythonExe = filepath.Join(venvPath, "bin", "python")
	}
	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command(pythonExe, cmdArgs...)

	// Create a pipe for stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	// Redirect stderr to discard
	cmd.Stderr = nil

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start python command: %v", err)
	}

	// Read stdout
	output, err := io.ReadAll(stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdout: %v", err)
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("python execution failed: %v, output: %s", err, string(output))
	}

	return output, nil
}

// Run Python script with JSON input via stdin and separate stdout/stderr
func RunPythonFromVenvWithJSONSeparateOutput(venvPath, scriptPath string, jsonInput interface{}, args ...string) ([]byte, error) {
	var pythonExe string
	if runtime.GOOS == "windows" {
		pythonExe = filepath.Join(venvPath, "Scripts", "python.exe")
	} else {
		pythonExe = filepath.Join(venvPath, "bin", "python")
	}
	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command(pythonExe, cmdArgs...)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if jsonInput != nil {
		jsonBytes, err := json.Marshal(jsonInput)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON input: %v", err)
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
		}
		go func() {
			defer stdin.Close()
			stdin.Write(jsonBytes)
		}()
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start python command: %v", err)
	}

	// Read stdout and stderr concurrently
	stdoutChan := make(chan []byte, 1)
	stderrChan := make(chan []byte, 1)

	go func() {
		output, _ := io.ReadAll(stdout)
		stdoutChan <- output
	}()

	go func() {
		output, _ := io.ReadAll(stderr)
		stderrChan <- output
	}()

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		stderrOutput := <-stderrChan
		return nil, fmt.Errorf("python execution failed: %v, stderr: %s", err, string(stderrOutput))
	}

	// Get stdout output
	stdoutOutput := <-stdoutChan
	stderrOutput := <-stderrChan

	// Log stderr output if any (for debugging)
	if len(stderrOutput) > 0 {
		logger.Debug("Python script stderr output", map[string]interface{}{
			"component": "python_runner",
			"script":    scriptPath,
			"stderr":    string(stderrOutput),
		})
	}

	return stdoutOutput, nil
}

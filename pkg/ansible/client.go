package ansible

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	log "winterflow-agent/pkg/log"
)

type Config struct {
	// AnsiblePath is the path where ansible files are stored
	AnsiblePath string `json:"ansible_path,omitempty"`
	// AnsibleAppsPath is the path where ansible application files are stored
	AnsibleAppsPath string `json:"ansible_apps_path,omitempty"`
	// AnsibleLogsPath defines the directory path where ansible log files are stored.
	AnsibleLogsPath string `json:"ansible_logs_path,omitempty"`
	// AppsPath is the path where application files are stored
	AppsPath string `json:"apps_path,omitempty"`
}

// Result represents the result of an Ansible command execution
type Result struct {
	// ExitCode is the exit code of the Ansible command
	ExitCode int
	// LogPath is the path to the log file
	LogPath string
	// Error is any error that occurred during execution
	Error error
}

// Client is the interface for the Ansible client
type Client interface {
	// RunSync executes an Ansible command synchronously and returns the result
	RunSync(id string, args []string) Result

	// RunAsync executes an Ansible command asynchronously and returns the log path
	// The caller can use the returned context.CancelFunc to cancel the execution
	RunAsync(id string, args []string) (string, context.CancelFunc, error)
}

// client implements the Client interface
type client struct {
	config    *Config
	mu        sync.Mutex
	processes map[string]*exec.Cmd
}

// NewClient creates a new Ansible client
func NewClient(config *Config) Client {
	return &client{
		config:    config,
		processes: make(map[string]*exec.Cmd),
	}
}

// getLogPath returns the path to the log file for the given ID
func getLogPath(logsDir, id string) string {
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
	}
	return filepath.Join(logsDir, fmt.Sprintf("%s.log", id))
}

// RunSync executes an Ansible command synchronously and returns the result
func (c *client) RunSync(id string, args []string) Result {
	logPath := getLogPath(c.config.AnsibleLogsPath, id)

	// Create log file
	logFile, err := os.Create(logPath)
	if err != nil {
		return Result{
			ExitCode: -1,
			LogPath:  logPath,
			Error:    fmt.Errorf("failed to create log file: %w", err),
		}
	}
	defer logFile.Close()

	// Write header to log file
	fmt.Fprintf(logFile, "=== Ansible Command Execution (ID: %s) ===\n", id)
	fmt.Fprintf(logFile, "Command: ansible-playbook %s\n", args)
	fmt.Fprintf(logFile, "Working Directory: %s\n", c.config.AnsiblePath)
	fmt.Fprintf(logFile, "=== Output ===\n\n")

	// Create command
	cmd := exec.Command("ansible-playbook", args...)
	cmd.Dir = c.config.AnsiblePath

	// Set up output to both log file and standard output
	cmd.Stdout = io.MultiWriter(logFile, os.Stdout)
	cmd.Stderr = io.MultiWriter(logFile, os.Stderr)

	// Execute command
	err = cmd.Run()

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}

		// Write error to log file
		fmt.Fprintf(logFile, "\n=== Error ===\n%v\n", err)
	}

	// Write footer to log file
	fmt.Fprintf(logFile, "\n=== Execution Completed (Exit Code: %d) ===\n", exitCode)

	return Result{
		ExitCode: exitCode,
		LogPath:  logPath,
		Error:    err,
	}
}

// RunAsync executes an Ansible command asynchronously and returns the log path
func (c *client) RunAsync(id string, args []string) (string, context.CancelFunc, error) {
	logPath := getLogPath(c.config.AnsibleLogsPath, id)

	// Create log file
	logFile, err := os.Create(logPath)
	if err != nil {
		return logPath, nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Write header to log file
	fmt.Fprintf(logFile, "=== Ansible Command Execution (ID: %s) ===\n", id)
	fmt.Fprintf(logFile, "Command: ansible-playbook %s\n", args)
	fmt.Fprintf(logFile, "Working Directory: %s\n", c.config.AnsiblePath)
	fmt.Fprintf(logFile, "=== Output ===\n\n")

	// Create command
	cmd := exec.Command("ansible-playbook", args...)
	cmd.Dir = c.config.AnsiblePath

	// Set up output to log file
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Create a context that can be used to cancel the command
	ctx, cancel := context.WithCancel(context.Background())

	// Store the command in the processes map
	c.mu.Lock()
	c.processes[id] = cmd
	c.mu.Unlock()

	// Create a cancel function that will clean up resources
	cancelFunc := func() {
		cancel()

		// Remove the process from the map
		c.mu.Lock()
		delete(c.processes, id)
		c.mu.Unlock()

		// Kill the process if it's still running
		if cmd.Process != nil {
			cmd.Process.Kill()
		}

		// Close the log file
		logFile.Close()
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		cancelFunc()
		return logPath, nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Run the command in a goroutine
	go func() {
		defer cancelFunc()

		// Wait for the command to complete or be cancelled
		select {
		case <-ctx.Done():
			// Command was cancelled
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			fmt.Fprintf(logFile, "\n=== Execution Cancelled ===\n")
		default:
			// Command completed normally
			err := cmd.Wait()

			// Get exit code
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					exitCode = -1
				}

				// Write error to log file
				fmt.Fprintf(logFile, "\n=== Error ===\n%v\n", err)
			}

			// Write footer to log file
			fmt.Fprintf(logFile, "\n=== Execution Completed (Exit Code: %d) ===\n", exitCode)
		}
	}()

	return logPath, cancelFunc, nil
}

package ansible

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	log "winterflow-agent/pkg/log"

	"github.com/google/uuid"
)

type Config struct {
	// Orchestrator specifies the orchestration tool to be used, such as Kubernetes or Ansible.
	Orchestrator string
	// AnsiblePath is the path where ansible files are stored
	AnsiblePath string
	// AppsTemplatesPath is the path where application templates are stored
	AppsTemplatesPath string
	// AnsibleLogsPath defines the directory path where ansible log files are stored.
	AnsibleLogsPath string
	// AppsCurrentVersion represents the current version folder name of application templates.
	AppsCurrentVersion string
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

type Command struct {
	Id       string
	Playbook string
	Env      map[string]string
	Args     []string
}

// Client is the interface for the Ansible client
type Client interface {
	// RunSync executes an Ansible command synchronously and returns the result
	RunSync(cmd Command) Result

	// RunAsync executes an Ansible command asynchronously and returns the log path
	// The caller can use the returned context.CancelFunc to cancel the execution
	RunAsync(cmd Command) (string, context.CancelFunc, error)

	GetConfig() *Config
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

// getLogPath returns the path to the log file for the given ID with timestamp and playbook name
func getLogPath(logsDir, playbook, id string) string {
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
	}

	// Generate timestamp in YYYYMMDDHHIISS format
	timestamp := time.Now().Format("20060102150405")

	// Clean playbook name to be filesystem-safe
	cleanPlaybook := strings.ReplaceAll(playbook, "/", "_")
	cleanPlaybook = strings.ReplaceAll(cleanPlaybook, "\\", "_")
	cleanPlaybook = strings.ReplaceAll(cleanPlaybook, ".yml", "")
	cleanPlaybook = strings.ReplaceAll(cleanPlaybook, ".yaml", "")

	filename := fmt.Sprintf("%s_%s_%s.log", timestamp, cleanPlaybook, id)
	return filepath.Join(logsDir, filename)
}

func (c *client) updateEnvironment(cmd Command) Command {
	if cmd.Env == nil {
		cmd.Env = make(map[string]string)
	}
	if cmd.Env["orchestrator"] == "" {
		cmd.Env["orchestrator"] = c.config.Orchestrator
	}
	if cmd.Env["apps_templates_path"] == "" {
		cmd.Env["apps_templates_path"] = c.config.AppsTemplatesPath
	}

	return cmd
}

// RunSync executes an Ansible command synchronously and returns the result
func (c *client) RunSync(cmd Command) Result {
	id := ""
	if cmd.Id != "" {
		id = cmd.Id
	} else {
		id = uuid.New().String()
	}

	logPath := getLogPath(c.config.AnsibleLogsPath, cmd.Playbook, id)
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

	cmd = c.updateEnvironment(cmd)
	cmdArgs := commandArgs(c.config.AnsiblePath, cmd)

	// Write header to log file
	fmt.Fprintf(logFile, "=== Ansible Command Execution (ID: %s) ===\n", id)
	fmt.Fprintf(logFile, "Command: ansible-playbook %s %s\n", cmd.Playbook, cmdArgs)
	fmt.Fprintf(logFile, "Working Directory: %s\n", c.config.AnsiblePath)
	fmt.Fprintf(logFile, "=== Output ===\n\n")

	runner := exec.Command("ansible-playbook", cmdArgs...)
	runner.Dir = c.config.AnsiblePath

	// Set up output to log file
	runner.Stdout = logFile
	runner.Stderr = logFile

	// Execute command
	err = runner.Run()

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			fmt.Fprintf(logFile, "\n=== Error (Exit Code: %d) ===\n%v\n", exitCode, err)
			log.Info(fmt.Sprintf("Ansible playbook %s failed with exit code %d: %v", cmd.Playbook, exitCode, err))
		} else {
			exitCode = -1
			fmt.Fprintf(logFile, "\n=== Error (Unknown) ===\n%v\n", err)
			log.Info(fmt.Sprintf("Ansible playbook %s failed with unknown error: %v", cmd.Playbook, err))
		}
	} else {
		log.Info(fmt.Sprintf("Ansible playbook %s completed successfully", cmd.Playbook))
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
func (c *client) RunAsync(cmd Command) (string, context.CancelFunc, error) {
	id := ""
	if cmd.Id != "" {
		id = cmd.Id
	} else {
		id = uuid.New().String()
	}
	logPath := getLogPath(c.config.AnsibleLogsPath, cmd.Playbook, id)

	// Create log file
	logFile, err := os.Create(logPath)
	if err != nil {
		return logPath, nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Write header to log file
	fmt.Fprintf(logFile, "=== Ansible Command Execution (ID: %s) ===\n", id)
	fmt.Fprintf(logFile, "Command: ansible-playbook %s\n", cmd.Playbook)
	fmt.Fprintf(logFile, "Working Directory: %s\n", c.config.AnsiblePath)
	fmt.Fprintf(logFile, "=== Output ===\n\n")

	// Create command
	c.updateEnvironment(cmd)
	cmdArgs := commandArgs(c.config.AnsiblePath, cmd)
	runner := exec.Command("ansible-playbook", cmdArgs...)
	runner.Dir = c.config.AnsiblePath

	// Set up output to log file
	runner.Stdout = logFile
	runner.Stderr = logFile

	// Create a context that can be used to cancel the command
	ctx, cancel := context.WithCancel(context.Background())

	// Store the command in the processes map
	c.mu.Lock()
	c.processes[id] = runner
	c.mu.Unlock()

	// Create a cancel function that will clean up resources
	cancelFunc := func() {
		cancel()

		// Remove the process from the map
		c.mu.Lock()
		delete(c.processes, id)
		c.mu.Unlock()

		// Kill the process if it's still running
		if runner.Process != nil {
			runner.Process.Kill()
		}

		// Close the log file
		logFile.Close()
	}

	// Start the command
	if err := runner.Start(); err != nil {
		cancelFunc()
		return logPath, nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Run the command in a goroutine
	go func() {
		defer cancelFunc()

		// Create a channel to signal command completion
		done := make(chan error, 1)
		go func() {
			done <- runner.Wait()
		}()

		// Wait for the command to complete or be cancelled
		select {
		case <-ctx.Done():
			// Command was cancelled
			if runner.Process != nil {
				runner.Process.Kill()
			}
			fmt.Fprintf(logFile, "\n=== Execution Cancelled ===\n")
		case err := <-done:
			// Command completed normally
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

func (c *client) GetConfig() *Config {
	return c.config
}

func commandArgs(ansiblePath string, cmd Command) []string {
	cmdArgs := []string{
		fmt.Sprintf("playbooks/%s", cmd.Playbook),
		"-i", "inventory/defaults.yml",
	}

	if _, err := os.Stat(fmt.Sprintf("%s/inventory/defaults.override.yml", ansiblePath)); err == nil {
		cmdArgs = append(cmdArgs, "-i", "inventory/defaults.override.yml")
	}

	for k, v := range cmd.Env {
		cmdArgs = append(cmdArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	return append(cmdArgs, cmd.Args...)
}

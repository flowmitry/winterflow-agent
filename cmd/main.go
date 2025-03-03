package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"winterflow-agent/internal/config"
	"winterflow-agent/internal/system"
	"winterflow-agent/internal/winterflow"
)

// version is set during build using -X linker flag
var version = "dev"

// getDefaultConfigPath returns the default configuration file path relative to the executable
func getDefaultConfigPath() string {
	// Get the executable path
	exe, err := os.Executable()
	if err != nil {
		// Fallback to current working directory if executable path cannot be determined
		return "agent.config.json"
	}

	// Get the directory containing the executable
	exeDir := filepath.Dir(exe)

	// Return config path relative to the executable directory
	return filepath.Join(exeDir, "agent.config.json")
}

func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	register := flag.Bool("register", false, "Register agent")
	configPath := flag.String("config", getDefaultConfigPath(), "Path to configuration file")
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("winterflow-agent version %s\n", version)
		os.Exit(0)
	}

	// Create configuration manager
	configManager := config.NewManager(*configPath)

	// Load configuration
	cfg, err := configManager.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Verify device ID matches the system
	systemDeviceID, err := system.GetDeviceID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get system device ID: %v\n", err)
		os.Exit(1)
	}
	if cfg.DeviceID != systemDeviceID {
		fmt.Fprintf(os.Stderr, "Device ID mismatch: config has %s but system has %s\n", cfg.DeviceID, systemDeviceID)
		os.Exit(1)
	}

	// Handle registration
	if *register {
		fmt.Println("Starting agent registration...")
		if err := winterflow.Register(*configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Registration failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\nRegistration successful!")
		os.Exit(0)
	}

	fmt.Println("Winterflow Agent starting...")

	// Update system packages
	fmt.Println("Updating system packages...")
	if err := system.FetchPackagesUpdates(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update system: %v\n", err)
		os.Exit(1)
	}

	// Install required packages
	fmt.Println("Installing required packages...")
	if err := system.InstallRequiredPackages(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to install required packages: %v\n", err)
		os.Exit(1)
	}

	// Manage playbooks repository
	fmt.Println("Managing playbooks repository...")
	if err := system.DownloadPlaybooks(cfg.PlaybooksPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to manage playbooks repository: %v\n", err)
		os.Exit(1)
	}

	// TODO: Add your agent initialization and main logic here
}

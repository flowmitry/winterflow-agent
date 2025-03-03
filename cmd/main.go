package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

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

	// TODO: Add your agent initialization and main logic here
	fmt.Println("Winterflow Agent starting...")
}

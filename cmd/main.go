package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"winterflow-agent/internal/agent"
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/api"
	"winterflow-agent/pkg/version"
)

func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")
	configPath := flag.String("config", "agent.config.json", "Path to configuration file")
	register := flag.Bool("register", false, "Register the agent with the server")
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("WinterFlow.io Agent version: %s (#%d)\n", version.GetVersion(), version.GetNumericVersion())
		os.Exit(0)
	}

	if *showHelp {
		fmt.Println("WinterFlow.io Agent")
		fmt.Println("Usage: winterflow-agent [options]")
		fmt.Println("Options:")
		fmt.Println("  --version  Show version information")
		fmt.Println("  --help     Show help information")
		fmt.Println("  --config   Path to configuration file (default: agent.config.json)")
		fmt.Println("  --register Register the agent with the server")
		os.Exit(0)
	}

	// Handle registration if requested
	if *register {
		if err := api.RegisterAgent(*configPath); err != nil {
			log.Fatalf("Registration failed: %v", err)
		}
		return
	}

	// Load configuration for normal operation
	cfg, err := config.WaitUntilReady(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and initialize agent
	agent, err := agent.NewAgent(cfg)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Register the agent
	accessToken, err := agent.Register()
	if err != nil {
		log.Fatalf("Failed to register agent: %v", err)
	}

	// Start heartbeat stream
	if err := agent.StartHeartbeat(accessToken); err != nil {
		log.Fatalf("Failed to start heartbeat stream: %v", err)
	}

	// Wait for interrupt signal
	agent.WaitForSignal()

	// Unregister the agent
	if err := agent.Unregister(accessToken); err != nil {
		log.Printf("Failed to unregister agent: %v", err)
	}
}

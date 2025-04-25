package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"winterflow-agent/internal/agent"
	"winterflow-agent/internal/config"
	"winterflow-agent/pkg/version"
)

func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")
	configPath := flag.String("config", "agent.config.json", "Path to configuration file")
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
		fmt.Println("  --config   Path to configuration file (default: /opt/winterflow/agent.config.json)")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and initialize agent
	app, err := agent.NewAgent(cfg)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer app.Close()

	// Register the agent
	accessToken, err := app.Register()
	if err != nil {
		log.Fatalf("Failed to register agent: %v", err)
	}

	// Start heartbeat stream
	if err := app.StartHeartbeat(accessToken); err != nil {
		log.Fatalf("Failed to start heartbeat stream: %v", err)
	}

	// Wait for interrupt signal
	app.WaitForSignal()

	// Unregister the agent
	if err := app.Unregister(accessToken); err != nil {
		log.Printf("Failed to unregister agent: %v", err)
	}
}

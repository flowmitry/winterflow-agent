package main

import (
	"fmt"
	"log"
	"os"

	"winterflow-agent/internal/agent"
	"winterflow-agent/pkg/version"
)

func main() {
	// Parse configuration
	config := agent.NewConfig()

	// Show version if requested
	if config.ShowVersion {
		fmt.Printf("WinterFlow.io Agent version: %s (#%d)\n", version.GetVersion(), version.GetNumericVersion())
		os.Exit(0)
	}

	if config.ShowHelp {
		fmt.Println("WinterFlow.io Agent")
		fmt.Println("Usage: winterflow-agent [options]")
		fmt.Println("Options:")
		fmt.Println("  --version  Show version information")
		fmt.Println("  --help     Show help information")
		os.Exit(0)
	}

	// Create and initialize agent
	app, err := agent.NewAgent(config)
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

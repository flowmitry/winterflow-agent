package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"syscall"
	"time"
	log "winterflow-agent/pkg/log"

	"winterflow-agent/internal/agent"
	"winterflow-agent/internal/ansible"
	"winterflow-agent/internal/winterflow/api"
)

//go:embed ansible/inventory/** ansible/playbooks/** ansible/roles/** all:ansible/apps/** ansible/ansible.cfg
var ansibleFS embed.FS

func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")
	configPath := flag.String("config", "agent.config.json", "Path to configuration file")
	register := flag.Bool("register", false, "Register the agent with the server")
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("WinterFlow.io Agent version: %s (#%d)\n", agent.GetVersion(), agent.GetNumericVersion())
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

	ansibleManager := ansible.NewManager(GetAnsibleFS(), *configPath)
	if err := ansibleManager.SyncAnsibleFiles(); err != nil {
		log.Fatalf("Error syncing ansible files: %v", err)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Handle signals in a separate goroutine
	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		cancel() // Cancel the context to abort operations
		// Give a short time for cleanup, then exit
		time.Sleep(500 * time.Millisecond)
		log.Printf("Shutting down agent")
		os.Exit(0)
	}()

	log.Printf("Loading configuration from %s", *configPath)
	cfg, err := agent.WaitUntilReady(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and initialize agent
	log.Printf("Creating agent")
	a, err := agent.NewAgent(cfg)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer a.Close()

	// Run the agent
	if err := a.Run(ctx); err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	// Wait indefinitely
	select {}
}

func GetAnsibleFS() fs.FS {
	fsys, err := fs.Sub(ansibleFS, "ansible")
	if err != nil {
		return nil
	}
	return fsys
}

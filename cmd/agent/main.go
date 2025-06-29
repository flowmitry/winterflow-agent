package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"winterflow-agent/internal/application"
	certsEmbedded "winterflow-agent/internal/infra/winterflow/certs"
	"winterflow-agent/pkg/log"

	"winterflow-agent/internal/application/agent"
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/application/version"
	"winterflow-agent/internal/infra/winterflow/api"
)

// Global variables to manage agent lifecycle
var (
	currentAgent *agent.Agent
	agentMutex   sync.Mutex
)

func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")
	configPath := flag.String("config", "agent.config.json", "Path to configuration file")
	register := flag.Bool("register", false, "Register the agent with the server. Optionally specify orchestrator as positional argument (e.g., --register docker_compose)")
	// New flag to trigger data restoration flow
	restore := flag.Bool("restore", false, "Restore agent data and templates after reinstall or migration")
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("\nWinterFlow.io Agent version: %s (#%d)\n", version.GetVersion(), version.GetNumericVersion())
		os.Exit(0)
	}

	if *showHelp {
		fmt.Println("\nWinterFlow.io Agent")
		fmt.Println("Usage: winterflow-agent [options]")
		fmt.Println("Options:")
		fmt.Println("  --version   Show version information")
		fmt.Println("  --help      Show help information")
		fmt.Println("  --config    Path to configuration file (default: agent.config.json)")
		fmt.Println("  --register  Register the agent with the server. Optionally specify orchestrator as positional argument (e.g., --register docker_compose)")
		fmt.Println("  --restore   Restore local state and notify the WinterFlow backend (used after agent re-installation)")
		os.Exit(0)
	}

	// Handle registration if requested
	if *register {
		// Determine orchestrator if provided as positional argument after flags
		var orchestrator string
		remainingArgs := flag.Args()
		if len(remainingArgs) > 0 {
			orchestrator = remainingArgs[0]
		}
		if err := api.RegisterAgent(*configPath, orchestrator); err != nil {
			fmt.Printf("Registration failed: %v\n", err)
		}
		return
	}

	// Handle data restoration if requested
	if *restore {
		if err := api.RestoreAgentData(*configPath); err != nil {
			fmt.Printf("Restore failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("WinterFlow.io Agent initialization...")
	if err := syncEmbeddedFiles(*configPath); err != nil {
		fmt.Printf("\nFailed to sync embedded files: %v", err)
		os.Exit(1)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Handle signals in a separate goroutine
	go func() {
		sig := <-sigChan
		log.Info("Received signal", "signal", sig.String())
		log.Info("Initiating graceful shutdown")

		// Cancel the context to abort operations
		cancel()

		// The agent will be closed by the defer a.Close() statement
		// which will handle graceful shutdown of the command bus

		// The main function will exit naturally
		// after the agent is closed and all commands have completed
		// Having a timeout to quit if the agent stuck
		time.Sleep(5 * time.Second)
		log.Info("Shutting down agent")
		os.Exit(0)
	}()

	// Start the agent with the given configuration
	startAgent(ctx, cancel, *configPath)

	// Wait for context cancellation
	<-ctx.Done()
	log.Info("Context canceled, shutting down agent")

	// Close the current agent if it exists
	stopCurrentAgent()
}

// startAgent initializes and starts the agent with the given configuration
func startAgent(ctx context.Context, cancel context.CancelFunc, configPath string) {
	// Load configuration
	fmt.Printf("\nLoading configuration from %s", configPath)
	cfg, err := config.WaitUntilReady(configPath)
	if err != nil {
		fmt.Printf("\nFailed to load configuration: %v", err)
		os.Exit(1)
	}

	// Initialize logger with configured log level
	log.InitLog(cfg.LogLevel)
	fmt.Printf("\nWinterFlow.io Agent initialized with Log Level \"%s\"\n", cfg.LogLevel)

	appRepository := application.NewAppRepository(cfg)
	registryRepository := application.NewRegistryRepository()
	networkRepository := application.NewNetworkRepository()

	// Create and initialize agent
	log.Debug("Creating agent")
	a, err := agent.NewAgent(cfg, appRepository, registryRepository, networkRepository)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Store the agent in the global variable
	agentMutex.Lock()
	if currentAgent != nil {
		currentAgent.Close()
	}
	currentAgent = a
	agentMutex.Unlock()

	// Run the agent
	if err := a.Run(ctx); err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	// Set up configuration file watcher
	watcher := application.NewConfigWatcher(configPath, func(newConfig *config.Config) {
		log.Info("Configuration changed, restarting agent")

		// Create a new context for the new agent
		newCtx, newCancel := context.WithCancel(context.Background())

		// Stop the current agent
		cancel()

		// Start a new agent with the new configuration
		go startAgent(newCtx, newCancel, configPath)
	})

	if err := watcher.Start(ctx); err != nil {
		log.Error("Failed to start config watcher", "error", err)
	}
}

// stopCurrentAgent safely stops the current agent if it exists
func stopCurrentAgent() {
	agentMutex.Lock()
	defer agentMutex.Unlock()

	if currentAgent != nil {
		log.Info("Closing current agent")
		currentAgent.Close()
		currentAgent = nil
	}
}

func syncEmbeddedFiles(configPath string) error {
	certsManager := certsEmbedded.NewManager(configPath)
	if err := certsManager.SyncFiles(); err != nil {
		log.Error("Error syncing embedded files", "config_path", configPath, "error", err)
		return err
	}

	return nil
}

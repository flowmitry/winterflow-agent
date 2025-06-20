package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"winterflow-agent/internal/application"
	"winterflow-agent/internal/infra/winterflow/grpc/certs"
	log "winterflow-agent/pkg/log"

	"winterflow-agent/internal/application/agent"
	"winterflow-agent/internal/application/version"
	"winterflow-agent/internal/config"
	ansiblefiles "winterflow-agent/internal/infra/ansible/files"
	"winterflow-agent/internal/infra/winterflow/api"
)

//go:embed ansible/inventory/** ansible/playbooks/** ansible/roles/** ansible/apps_roles/README.md ansible/ansible.cfg
var ansibleFS embed.FS

//go:embed .certs/ca.crt
var certsFS embed.FS

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
	register := flag.Bool("register", false, "Register the agent with the server")
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
		fmt.Println("  --version  Show version information")
		fmt.Println("  --help     Show help information")
		fmt.Println("  --config   Path to configuration file (default: agent.config.json)")
		fmt.Println("  --register Register the agent with the server")
		os.Exit(0)
	}

	// Handle registration if requested
	if *register {
		if err := api.RegisterAgent(*configPath); err != nil {
			fmt.Printf("Registration failed: %v\n", err)
		}
		return
	}

	fmt.Printf("WinterFlow.io Agent initialization...")
	if err := syncEmbeddedFiles(*configPath, ansibleFS, certsFS); err != nil {
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
		log.Info("Received signal: %v", sig)
		log.Info("Initiating graceful shutdown...")

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
	log.Printf("Context canceled, shutting down agent...")

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

	// Create and initialize agent
	log.Debug("Creating agent")
	a, err := agent.NewAgent(cfg, appRepository)
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
		log.Errorf("Failed to start config watcher: %v", err)
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

func syncEmbeddedFiles(configPath string, ansibleFS embed.FS, certsFS embed.FS) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("\nFailed to load configuration: %v", err)
		return err
	}

	fsysAnsible, err := fs.Sub(ansibleFS, cfg.GetAnsibleFolder())
	if err != nil {
		fmt.Printf("\nError accessing ansible filesystem: %v", err)
		return err
	}
	ansibleManager := ansiblefiles.NewManager(fsysAnsible, configPath)
	if err := ansibleManager.SyncFiles(); err != nil {
		fmt.Printf("\nError syncing ansible files: %v", err)
		return err
	}

	fsysCerts, err := fs.Sub(certsFS, cfg.GetEmbeddedCertificatesFolder())
	if err != nil {
		fmt.Printf("\nError accessing certificates filesystem: %v", err)
		return err
	}
	certsManager := certs.NewManager(fsysCerts, configPath)
	if err := certsManager.SyncFiles(); err != nil {
		fmt.Printf("\nError syncing ansible files: %v", err)
		return err
	}

	return nil
}

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
	"winterflow-agent/internal/winterflow/grpc/certs"
	log "winterflow-agent/pkg/log"

	"winterflow-agent/internal/agent"
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/version"
	ansible "winterflow-agent/internal/winterflow/ansible"
	ansiblefiles "winterflow-agent/internal/winterflow/ansible/files"
	"winterflow-agent/internal/winterflow/api"
)

//go:embed ansible/inventory/** ansible/playbooks/** ansible/roles/** ansible/apps_roles/README.md ansible/ansible.cfg
var ansibleFS embed.FS

//go:embed .certs/ca.crt
var certsFS embed.FS

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

	if err := syncEmbeddedFiles(*configPath, ansibleFS, certsFS); err != nil {
		log.Fatalf("Error syncing embedded files: %v", err)
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

	log.Debug("Loading configuration from %s", *configPath)
	cfg, err := config.WaitUntilReady(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ansibleRepo := ansible.NewRepository(cfg)
	result := ansibleRepo.InitialConfiguration()
	if result.Error != nil {
		log.Warn(fmt.Sprintf("Initial configuration playbook failed: %v, log: %s", result.Error, result.LogPath))
	} else {
		log.Info(fmt.Sprintf("Initial configuration playbook completed successfully. Logs: %s", result.LogPath))
	}

	// Create and initialize agent
	log.Debug("Creating agent")
	a, err := agent.NewAgent(cfg, ansibleRepo)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer a.Close()

	// Run the agent
	if err := a.Run(ctx); err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Printf("Context canceled, shutting down agent...")
}

func syncEmbeddedFiles(configPath string, ansibleFS embed.FS, certsFS embed.FS) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
		return err
	}

	fsysAnsible, err := fs.Sub(ansibleFS, cfg.GetAnsibleFolder())
	if err != nil {
		log.Fatalf("Error accessing ansible filesystem: %v", err)
		return err
	}
	ansibleManager := ansiblefiles.NewManager(fsysAnsible, configPath)
	if err := ansibleManager.SyncFiles(); err != nil {
		log.Fatalf("Error syncing ansible files: %v", err)
		return err
	}

	fsysCerts, err := fs.Sub(certsFS, cfg.GetEmbeddedCertificateFolder())
	if err != nil {
		log.Fatalf("Error accessing certificates filesystem: %v", err)
		return err
	}
	certsManager := certs.NewManager(fsysCerts, configPath)
	if err := certsManager.SyncFiles(); err != nil {
		log.Fatalf("Error syncing ansible files: %v", err)
		return err
	}

	return nil
}

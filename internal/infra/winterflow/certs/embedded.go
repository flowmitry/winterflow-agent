package certs

import (
	"embed"
	"io/fs"

	log "winterflow-agent/pkg/log"

	"winterflow-agent/internal/application/config"
	"winterflow-agent/pkg/embedded"
)

//go:embed assets/**
var certsFS embed.FS

type Manager struct {
	embeddedManager *embedded.Manager
}

// NewManager creates a new Ansible manager
func NewManager(configPath string) *Manager {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create a sub-filesystem rooted at the "assets" directory so that the
	// extracted file paths do not include the top-level "assets" prefix.
	subFS, err := fs.Sub(certsFS, "assets")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem for embedded certificates: %v", err)
	}

	return &Manager{
		embeddedManager: embedded.NewManager(subFS, cfg.GetCertificatesDefaultFolder()),
	}
}

// SyncFiles synchronizes the certificate files using the embeddedManager's SyncFiles method.
func (m *Manager) SyncFiles() error {
	log.Printf("Syncing certificates")
	return m.embeddedManager.SyncFiles()
}

package certs

import (
	"io/fs"
	"winterflow-agent/internal/application/version"
	log "winterflow-agent/pkg/log"

	"winterflow-agent/internal/application/config"
	"winterflow-agent/pkg/embedded"
)

// Manager handles ansible-related operations
type Manager struct {
	embeddedManager *embedded.Manager
}

// NewManager creates a new Ansible manager
func NewManager(embeddedFS fs.FS, configPath string) *Manager {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	return &Manager{
		embeddedManager: embedded.NewManager(embeddedFS, cfg.GetEmbeddedCertificatesFolder(), version.GetVersion(), []string{
			cfg.GetCACertificateFile(),
		}),
	}
}

// SyncFiles synchronizes the certificate files using the embeddedManager's SyncFiles method.
func (m *Manager) SyncFiles() error {
	log.Printf("Syncing certificates")
	return m.embeddedManager.SyncFiles()
}

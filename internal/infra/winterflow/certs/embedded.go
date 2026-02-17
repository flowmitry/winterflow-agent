package certs

import (
	"embed"
	"io/fs"

	"winterflow-agent/pkg/log"

	"winterflow-agent/internal/application/config"
	"winterflow-agent/pkg/embedded"
)

//go:embed assets/**
var certsFS embed.FS

type Manager struct {
	embeddedManager *embedded.Manager
}

func NewManager(configPath string) *Manager {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	subFS, err := fs.Sub(certsFS, "assets")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem for embedded certificates: %v", err)
	}

	return &Manager{
		embeddedManager: embedded.NewManager(subFS, cfg.GetCertificatesDefaultFolder()),
	}
}

func (m *Manager) SyncFiles() error {
	log.Debug("Syncing certificates")
	return m.embeddedManager.SyncFiles()
}

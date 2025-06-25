package create_registry

import (
	"fmt"
	"strings"
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/pkg/certs"
	log "winterflow-agent/pkg/log"
)

// CreateRegistryHandler integrates with the DockerRegistryRepository to execute CreateRegistryCommand.
type CreateRegistryHandler struct {
	repository repository.DockerRegistryRepository
	config     *config.Config
}

// Handle executes the CreateRegistryCommand.
func (h *CreateRegistryHandler) Handle(cmd CreateRegistryCommand) error {
	// Check if registries feature is disabled
	if h.config != nil && h.config.IsFeatureEnabled(config.FeatureRegistriesDisabled) {
		return log.Errorf("registries operations are disabled by configuration")
	}

	log.Info("Processing create registry command", "address", cmd.Address)

	if cmd.Address == "" {
		return log.Errorf("registry address is required")
	}

	username := strings.TrimSpace(cmd.Username)
	password := strings.TrimSpace(cmd.Password)

	// Attempt to decrypt credentials when a private key is configured and the value is not empty.
	if h.config.GetPrivateKeyPath() == "" {
		return log.Errorf("private key path is not configured")
	}

	if username != "" {
		if dec, err := certs.DecryptWithPrivateKey(h.config.GetPrivateKeyPath(), username); err == nil {
			username = dec
		} else {
			log.Warn("Failed to decrypt registry username", "error", err)
		}
	}

	if password != "" {
		if dec, err := certs.DecryptWithPrivateKey(h.config.GetPrivateKeyPath(), password); err == nil {
			password = dec
		} else {
			log.Warn("Failed to decrypt registry password", "error", err)
		}
	}

	reg := model.Registry{Address: cmd.Address}
	if err := h.repository.CreateRegistry(reg, username, password); err != nil {
		return fmt.Errorf("failed to create registry: %w", err)
	}

	log.Info("Registry created successfully", "address", cmd.Address)
	return nil
}

// NewCreateRegistryHandler returns a configured CreateRegistryHandler.
func NewCreateRegistryHandler(repo repository.DockerRegistryRepository, cfg *config.Config) *CreateRegistryHandler {
	return &CreateRegistryHandler{
		repository: repo,
		config:     cfg,
	}
}

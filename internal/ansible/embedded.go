package ansible

import (
	"io/fs"
	"log"

	"winterflow-agent/internal/agent"
	"winterflow-agent/pkg/embedded"
)

const (
	// AnsibleDir is the name of the ansible directory
	AnsibleDir = "ansible"
)

// Manager handles ansible-related operations
type Manager struct {
	embeddedManager *embedded.Manager
}

// NewManager creates a new Ansible manager
func NewManager(embeddedFS fs.FS) *Manager {
	return &Manager{
		embeddedManager: embedded.NewManager(embeddedFS, AnsibleDir, agent.GetVersion(), []string{
			"inventory/defaults.yml",
			"playbooks",
			"roles",
			"ansible.cfg",
		}),
	}
}

// SyncAnsibleFiles ensures the ansible directory is up to date with the embedded files
func (m *Manager) SyncAnsibleFiles() error {
	log.Printf("Syncing ansible files")
	return m.embeddedManager.SyncFiles()
}

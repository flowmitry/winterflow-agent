package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/service/util"
)

type RevisionServiceInterface interface {
	GetAppRevisions(appID string) ([]uint32, error)

	ValidateAppRevision(appID string, revision uint32) (bool, error)

	DeleteAppRevision(appID string, revision uint32) error

	DeleteOldRevisions(appID string) error

	CreateRevision(appID string) (uint32, error)

	GetLatestAppRevision(appID string) (uint32, error)

	GetRevisionDir(appID string, revision uint32) string

	GetVarsDir(appID string, revision uint32) string

	GetFilesDir(appID string, revision uint32) string
}

type RevisionService struct {
	config *config.Config
}

// Ensure RevisionService implements RevisionServiceInterface
var _ RevisionServiceInterface = (*RevisionService)(nil)

// NewRevisionService creates a new RevisionService
func NewRevisionService(config *config.Config) *RevisionService {
	return &RevisionService{
		config: config,
	}
}

func (s *RevisionService) GetAppRevisions(appID string) ([]uint32, error) {
	appDir := filepath.Join(s.config.GetAppsTemplatesPath(), appID)

	// Check if the app directory exists
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return []uint32{}, nil // Return empty slice if app doesn't exist
	}

	// Read the app directory to get revision folders
	entries, err := os.ReadDir(appDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read app directory %s: %w", appDir, err)
	}

	var revisions []uint32
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip non-directory entries
		}

		revisionStr := entry.Name()

		// Try to parse the revision as a number
		revision, err := strconv.ParseUint(revisionStr, 10, 32)
		if err != nil {
			// Skip non-numeric revision folders
			continue
		}

		// Check if the revision directory contains a config.json file
		configPath := filepath.Join(appDir, revisionStr, "config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Skip revision directories without config.json
			continue
		}

		revisions = append(revisions, uint32(revision))
	}

	// Sort revisions in ascending order
	sort.Slice(revisions, func(i, j int) bool {
		return revisions[i] < revisions[j]
	})

	return revisions, nil
}

func (s *RevisionService) ValidateAppRevision(appID string, revision uint32) (bool, error) {
	revisions, err := s.GetAppRevisions(appID)
	if err != nil {
		return false, err
	}

	for _, v := range revisions {
		if v == revision {
			return true, nil
		}
	}

	return false, nil
}

func (s *RevisionService) DeleteAppRevision(appID string, revision uint32) error {
	// First validate that the revision exists
	exists, err := s.ValidateAppRevision(appID, revision)
	if err != nil {
		return fmt.Errorf("failed to validate app revision: %w", err)
	}

	if !exists {
		return fmt.Errorf("revision %d does not exist for app %s", revision, appID)
	}

	// Construct the revision directory path
	revisionDir := filepath.Join(s.config.GetAppsTemplatesPath(), appID, fmt.Sprintf("%d", revision))

	// Remove the revision directory and all its contents
	err = os.RemoveAll(revisionDir)
	if err != nil {
		return fmt.Errorf("failed to delete revision directory %s: %w", revisionDir, err)
	}

	return nil
}

func (s *RevisionService) DeleteOldRevisions(appID string) error {
	revisions, err := s.GetAppRevisions(appID)
	if err != nil {
		return fmt.Errorf("failed to get app revisions for %s: %w", appID, err)
	}

	// If we have fewer revisions than the keep limit, no need to delete anything
	keepAppRevisions := s.config.GetKeepAppRevisions()
	if len(revisions) <= keepAppRevisions {
		return nil
	}

	// Sort revisions in descending order (newest first)
	sort.Slice(revisions, func(i, j int) bool {
		return revisions[i] > revisions[j]
	})

	// Calculate how many revisions to delete
	revisionsToDelete := len(revisions) - keepAppRevisions

	// Delete the oldest revisions (they are at the end of the sorted slice)
	for i := 0; i < revisionsToDelete; i++ {
		revisionToDelete := revisions[len(revisions)-1-i] // Get the oldest revision
		err := s.DeleteAppRevision(appID, revisionToDelete)
		if err != nil {
			return fmt.Errorf("failed to delete revision %d for app %s: %w", revisionToDelete, appID, err)
		}
	}

	return nil
}

func (s *RevisionService) CreateRevision(appID string) (uint32, error) {
	// Determine the latest existing revision for the app.
	latestRevision, err := s.GetLatestAppRevision(appID)
	if err != nil {
		return 0, fmt.Errorf("failed to determine latest revision for %s: %w", appID, err)
	}

	// If there are no revisions yet, bootstrap the first one.
	if latestRevision == 0 {
		return s.createFirstRevision(appID)
	}

	// Build source (latest) and destination (new) revision paths.
	sourceDir := filepath.Join(s.config.GetAppsTemplatesPath(), appID, fmt.Sprintf("%d", latestRevision))
	newRevision := latestRevision + 1
	destDir := filepath.Join(s.config.GetAppsTemplatesPath(), appID, fmt.Sprintf("%d", newRevision))

	// Copy the directory recursively to create the new revision.
	if err := util.CopyDirectory(sourceDir, destDir); err != nil {
		return 0, fmt.Errorf("failed to create new revision: %w", err)
	}

	return newRevision, nil
}

func (s *RevisionService) createFirstRevision(appID string) (uint32, error) {
	// Create the app directory if it doesn't exist
	appDir := filepath.Join(s.config.GetAppsTemplatesPath(), appID)
	err := os.MkdirAll(appDir, 0755)
	if err != nil {
		return 0, fmt.Errorf("failed to create app directory %s: %w", appDir, err)
	}

	// Create the first revision directory
	firstRevisionDir := filepath.Join(appDir, "1")
	err = os.MkdirAll(firstRevisionDir, 0755)
	if err != nil {
		return 0, fmt.Errorf("failed to create first revision directory %s: %w", firstRevisionDir, err)
	}

	// Create files directory
	filesDir := filepath.Join(firstRevisionDir, "files")
	err = os.MkdirAll(filesDir, 0755)
	if err != nil {
		return 0, fmt.Errorf("failed to create files directory %s: %w", filesDir, err)
	}

	// Create vars directory
	varsDir := filepath.Join(firstRevisionDir, "vars")
	err = os.MkdirAll(varsDir, 0755)
	if err != nil {
		return 0, fmt.Errorf("failed to create vars directory %s: %w", varsDir, err)
	}

	// Create vars/values.json with empty object
	valuesPath := filepath.Join(varsDir, "values.json")
	emptyValues := map[string]interface{}{}
	valuesData, err := json.MarshalIndent(emptyValues, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed to marshal empty values: %w", err)
	}

	err = os.WriteFile(valuesPath, valuesData, 0600)
	if err != nil {
		return 0, fmt.Errorf("failed to write values file %s: %w", valuesPath, err)
	}

	// Create a basic config.json file
	configPath := filepath.Join(firstRevisionDir, "config.json")
	basicConfig := map[string]interface{}{
		"id":        appID,
		"name":      "",
		"files":     []interface{}{},
		"variables": []interface{}{},
	}

	configData, err := json.MarshalIndent(basicConfig, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed to marshal basic config: %w", err)
	}

	err = os.WriteFile(configPath, configData, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to write config file %s: %w", configPath, err)
	}

	return 1, nil
}

func (s *RevisionService) GetLatestAppRevision(appID string) (uint32, error) {
	// Obtain the list of all existing revisions for the application. The helper
	// already filters out invalid directories so we can rely on its output.
	revisions, err := s.GetAppRevisions(appID)
	if err != nil {
		return 0, err
	}

	// When no revisions are found we treat this as an error condition so that
	// callers are forced to handle the un-initialised state explicitly.
	if len(revisions) == 0 {
		return 0, nil
	}

	// Revisions slice is sorted in ascending order, therefore the last element
	// represents the most recent revision.
	return revisions[len(revisions)-1], nil
}

func (s *RevisionService) GetRevisionDir(appID string, revision uint32) string {
	return filepath.Join(s.config.GetAppsTemplatesPath(), appID, fmt.Sprintf("%d", revision))
}

// GetVarsDir returns the vars/ directory within a specific revision.
func (s *RevisionService) GetVarsDir(appID string, revision uint32) string {
	return filepath.Join(s.GetRevisionDir(appID, revision), "vars")
}

// GetFilesDir returns the files/ directory within a specific revision.
func (s *RevisionService) GetFilesDir(appID string, revision uint32) string {
	return filepath.Join(s.GetRevisionDir(appID, revision), "files")
}

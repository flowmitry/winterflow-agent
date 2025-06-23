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

// AppVersionServiceInterface defines the contract for app version operations
type AppVersionServiceInterface interface {
	// GetAppVersions retrieves all available versions for a given app ID
	GetAppVersions(appID string) ([]uint32, error)

	// ValidateAppVersion checks if a specific version exists for an app
	ValidateAppVersion(appID string, version uint32) (bool, error)

	// DeleteAppVersion deletes a specific version for an app
	DeleteAppVersion(appID string, version uint32) error

	// DeleteOldVersions deletes old versions of an app, keeping only the specified number of recent versions
	DeleteOldVersions(appID string) error

	// CreateVersion creates a new version by copying the latest available version
	CreateVersion(appID string) (uint32, error)
}

// AppVersionService provides functionality to retrieve app versions
type AppVersionService struct {
	config *config.Config
}

// Ensure AppVersionService implements AppVersionServiceInterface
var _ AppVersionServiceInterface = (*AppVersionService)(nil)

// NewAppVersionService creates a new AppVersionService
func NewAppVersionService(config *config.Config) *AppVersionService {
	return &AppVersionService{
		config: config,
	}
}

// GetAppVersions retrieves all available versions for a given app ID
func (s *AppVersionService) GetAppVersions(appID string) ([]uint32, error) {
	appDir := filepath.Join(s.config.GetAppsTemplatesPath(), appID)

	// Check if the app directory exists
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return []uint32{}, nil // Return empty slice if app doesn't exist
	}

	// Read the app directory to get version folders
	entries, err := os.ReadDir(appDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read app directory %s: %w", appDir, err)
	}

	var versions []uint32
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip non-directory entries
		}

		versionStr := entry.Name()

		// Try to parse the version as a number
		version, err := strconv.ParseUint(versionStr, 10, 32)
		if err != nil {
			// Skip non-numeric version folders
			continue
		}

		// Check if the version directory contains a config.json file
		configPath := filepath.Join(appDir, versionStr, "config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Skip version directories without config.json
			continue
		}

		versions = append(versions, uint32(version))
	}

	// Sort versions in ascending order
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] < versions[j]
	})

	return versions, nil
}

// ValidateAppVersion checks if a specific version exists for an app
func (s *AppVersionService) ValidateAppVersion(appID string, version uint32) (bool, error) {
	versions, err := s.GetAppVersions(appID)
	if err != nil {
		return false, err
	}

	for _, v := range versions {
		if v == version {
			return true, nil
		}
	}

	return false, nil
}

// DeleteAppVersion deletes a specific version for an app
func (s *AppVersionService) DeleteAppVersion(appID string, version uint32) error {
	// First validate that the version exists
	exists, err := s.ValidateAppVersion(appID, version)
	if err != nil {
		return fmt.Errorf("failed to validate app version: %w", err)
	}

	if !exists {
		return fmt.Errorf("version %d does not exist for app %s", version, appID)
	}

	// Construct the version directory path
	versionDir := filepath.Join(s.config.GetAppsTemplatesPath(), appID, fmt.Sprintf("%d", version))

	// Remove the version directory and all its contents
	err = os.RemoveAll(versionDir)
	if err != nil {
		return fmt.Errorf("failed to delete version directory %s: %w", versionDir, err)
	}

	return nil
}

// DeleteOldVersions deletes old versions of an app, keeping only the specified number of recent versions
func (s *AppVersionService) DeleteOldVersions(appID string) error {
	// Get all available versions for the app
	versions, err := s.GetAppVersions(appID)
	if err != nil {
		return fmt.Errorf("failed to get app versions for %s: %w", appID, err)
	}

	// If we have fewer versions than the keep limit, no need to delete anything
	keepVersions := s.config.GetKeepAppVersions()
	if len(versions) <= keepVersions {
		return nil
	}

	// Sort versions in descending order (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] > versions[j]
	})

	// Calculate how many versions to delete
	versionsToDelete := len(versions) - keepVersions

	// Delete the oldest versions (they are at the end of the sorted slice)
	for i := 0; i < versionsToDelete; i++ {
		versionToDelete := versions[len(versions)-1-i] // Get the oldest version
		err := s.DeleteAppVersion(appID, versionToDelete)
		if err != nil {
			return fmt.Errorf("failed to delete version %d for app %s: %w", versionToDelete, appID, err)
		}
	}

	return nil
}

// CreateVersion creates a new version by copying the latest available version
func (s *AppVersionService) CreateVersion(appID string) (uint32, error) {
	// Get all available versions for the app
	versions, err := s.GetAppVersions(appID)
	if err != nil {
		return 0, fmt.Errorf("failed to get app versions for %s: %w", appID, err)
	}

	// If there are no versions, create the first version
	if len(versions) == 0 {
		return s.createFirstVersion(appID)
	}

	// Get the latest version
	latestVersion := versions[len(versions)-1]

	// Construct the source and destination paths
	sourceDir := filepath.Join(s.config.GetAppsTemplatesPath(), appID, fmt.Sprintf("%d", latestVersion))
	newVersion := latestVersion + 1
	destDir := filepath.Join(s.config.GetAppsTemplatesPath(), appID, fmt.Sprintf("%d", newVersion))

	// Copy the directory recursively
	err = util.CopyDirectory(sourceDir, destDir)
	if err != nil {
		return 0, fmt.Errorf("failed to create new version: %w", err)
	}

	return newVersion, nil
}

// createFirstVersion creates the first version (version 1) for an app
func (s *AppVersionService) createFirstVersion(appID string) (uint32, error) {
	// Create the app directory if it doesn't exist
	appDir := filepath.Join(s.config.GetAppsTemplatesPath(), appID)
	err := os.MkdirAll(appDir, 0755)
	if err != nil {
		return 0, fmt.Errorf("failed to create app directory %s: %w", appDir, err)
	}

	// Create the first version directory
	firstVersionDir := filepath.Join(appDir, "1")
	err = os.MkdirAll(firstVersionDir, 0755)
	if err != nil {
		return 0, fmt.Errorf("failed to create first version directory %s: %w", firstVersionDir, err)
	}

	// Create files directory
	filesDir := filepath.Join(firstVersionDir, "files")
	err = os.MkdirAll(filesDir, 0755)
	if err != nil {
		return 0, fmt.Errorf("failed to create files directory %s: %w", filesDir, err)
	}

	// Create vars directory
	varsDir := filepath.Join(firstVersionDir, "vars")
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

	err = os.WriteFile(valuesPath, valuesData, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to write values file %s: %w", valuesPath, err)
	}

	// Create a basic config.json file
	configPath := filepath.Join(firstVersionDir, "config.json")
	basicConfig := map[string]interface{}{
		"id":        appID,
		"name":      appID,
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

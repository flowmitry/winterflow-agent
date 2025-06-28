package docker_compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"winterflow-agent/internal/domain/model"
	appsvc "winterflow-agent/internal/domain/service/app"
	"winterflow-agent/pkg/log"
)

// fileExists returns true if the provided path exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// dirExists returns true if the provided path exists and **is** a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ensureDir checks if the given directory exists and creates it (including
// parent directories) if it does not. It returns an error only if the
// directory does not exist and cannot be created.
func ensureDir(path string) error {
	if dirExists(path) {
		return nil
	}
	return os.MkdirAll(path, 0o755) // Create with typical rwxr-xr-x permissions
}

// getAppName determines the human-readable application name by inspecting the
// config.json stored in the latest version directory.
//
// The function falls back to returning appID when it cannot reliably detect a
// name (e.g. missing config.json or empty name field). This preserves previous
// behaviour where the app ID served as a safe default.
func (r *composeRepository) getAppNameById(appID string) (string, error) {
	// Instantiate the version service on-demand – the constructor is cheap and
	// avoids having to store another dependency inside the repository struct.
	versionService := appsvc.NewAppVersionService(r.config)

	// Always work with the latest version of the application. If no version
	// exists yet we fall back to 1 (legacy default) to keep compatibility with
	// older deployments that did not support versioning.
	latest, err := versionService.GetLatestAppVersion(appID)
	if err != nil {
		return appID, fmt.Errorf("failed to determine latest version for app %s: %w", appID, err)
	}

	if latest == 0 {
		return appID, fmt.Errorf("application %s has no versions", appID)
	}
	version := latest

	return getAppName(versionService.GetVersionDir(appID, version))
}

func (r *composeRepository) getAppName(appPath string) (string, error) {
	return getAppName(appPath)
}

// getAppName reads the application configuration located at the provided path
func getAppName(appPath string) (string, error) {
	if strings.TrimSpace(appPath) == "" {
		return "", fmt.Errorf("app path cannot be empty")
	}

	// Construct path to config.json relative to the provided directory.
	configPath := filepath.Join(appPath, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read app config: %w", err)
	}

	appConfig, err := model.ParseAppConfig(data)
	if err != nil {
		return "", fmt.Errorf("failed to parse app config: %w", err)
	}

	name := strings.TrimSpace(appConfig.Name)
	if name == "" {
		return "", fmt.Errorf("application name is empty in config %s", configPath)
	}

	return name, nil
}

// removeDeployedFiles compares the file lists of the *previously* deployed configuration (oldCfg)
// and the *new* configuration that is about to be deployed (newCfg). It removes only those files
// from baseDir that existed in oldCfg but are absent in newCfg. This avoids unnecessary file
// deletions when a file persists across versions and helps preserve any runtime-generated data
// that might live next to the files.
//
// The function also attempts to prune now-empty parent directories, but it will never remove
// baseDir itself.
func (r *composeRepository) removeDeployedFiles(baseDir string, oldCfg, newCfg *model.AppConfig) error {
	if oldCfg == nil {
		return nil // Nothing to clean up.
	}

	log.Debug("Removing previously deployed files that are not part of the new version")

	// Build a lookup map of filenames that are present in the new configuration so we can perform
	// constant-time existence checks.
	newFiles := make(map[string]struct{})
	if newCfg != nil {
		for _, nf := range newCfg.Files {
			newFiles[nf.Filename] = struct{}{}
		}
	}

	for _, f := range oldCfg.Files {
		// If the file is still present in the new configuration, keep it.
		if _, keep := newFiles[f.Filename]; keep {
			continue
		}

		rel, err := sanitizeFileRelPath(f.Filename)
		if err != nil {
			log.Warn("[Cleanup] skipping invalid filename", "filename", f.Filename, "error", err)
			continue
		}

		absPath := filepath.Join(baseDir, rel)
		if err := os.Remove(absPath); err != nil {
			if os.IsNotExist(err) {
				continue // Already gone – nothing to do.
			}
			return fmt.Errorf("failed to remove file %s: %w", absPath, err)
		}
		log.Debug("Removed previously deployed file", "filename", rel)

		// Attempt to prune empty directories going up the tree, but never remove baseDir itself.
		dir := filepath.Dir(absPath)
		for dir != baseDir {
			entries, _ := os.ReadDir(dir)
			if len(entries) == 0 {
				_ = os.Remove(dir)
				dir = filepath.Dir(dir)
			} else {
				break
			}
		}
	}
	return nil
}

// sanitizeFileRelPath ensures that a file path from AppConfig cannot escape the application
// directory when joined with baseDir. The implementation mirrors the logic from the application
// layer (sanitizeTemplateFilename) but is replicated locally to avoid cross-layer dependencies.
func sanitizeFileRelPath(name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	clean = strings.TrimLeft(clean, string(os.PathSeparator))

	if clean == "" || clean == "." {
		return "", fmt.Errorf("invalid empty filename")
	}
	if filepath.IsAbs(clean) || strings.Contains(clean, "..") {
		return "", fmt.Errorf("invalid filename: potential path traversal detected")
	}
	return clean, nil
}

package docker_compose

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/infra/orchestrator"
	"winterflow-agent/pkg/log"
)

// loadTemplateVariables merges default and variable files into a single map used for template substitution.
func (r *composeRepository) loadTemplateVariables(templateDir string) (map[string]string, error) {
	vars := make(map[string]string)

	varsPath := filepath.Join(templateDir, "vars", "values.json")
	data, err := os.ReadFile(varsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return vars, nil // No vars file – that's fine.
		}
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse variables JSON: %w", err)
	}
	for k, v := range raw {
		vars[k] = fmt.Sprintf("%v", v)
	}
	return vars, nil
}

// renderTemplates processes all *.j2 files from templateDir/files into destDir performing a naïve variable substitution.
func (r *composeRepository) renderTemplates(templateDir, destDir string, vars map[string]string) error {
	filesRoot := filepath.Join(templateDir, "files")

	walkFn := func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Determine the destination relative to filesRoot.
		relPath, err := filepath.Rel(filesRoot, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		destPath := filepath.Join(destDir, relPath)

		// Handle directories by ensuring they exist in the destination.
		if d.IsDir() {
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			return nil
		}

		// For files, process depending on extension.
		if strings.HasSuffix(d.Name(), ".j2") {
			// Render template file.
			contentBytes, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read template %s: %w", path, err)
			}
			content := string(contentBytes)

			// Naïve variable substitution – full Jinja support not required.
			for name, value := range vars {
				patterns := []string{
					fmt.Sprintf("{{ %s }}", name),
					fmt.Sprintf("{{%s }}", name),
					fmt.Sprintf("{{ %s}}", name),
					fmt.Sprintf("{{%s}}", name),
				}
				for _, p := range patterns {
					content = strings.ReplaceAll(content, p, value)
				}
			}

			// Remove any leftover delimiters.
			content = strings.ReplaceAll(content, "{{", "")
			content = strings.ReplaceAll(content, "}}", "")

			// Drop the ".j2" extension for destination file.
			destPath = strings.TrimSuffix(destPath, ".j2")

			if err := os.WriteFile(destPath, []byte(content), 0o644); err != nil {
				return fmt.Errorf("failed to write rendered template to %s: %w", destPath, err)
			}
			return nil
		}

		// Non-template file – copy as-is.
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read source file %s: %w", path, err)
		}
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return fmt.Errorf("failed to copy file to %s: %w", destPath, err)
		}
		return nil
	}

	if err := filepath.WalkDir(filesRoot, walkFn); err != nil {
		return fmt.Errorf("failed to process templates: %w", err)
	}

	return nil
}

// renderApp prepares the application files for deployment by rendering templates from templateDir
// into destDir. It also performs differential cleanup of previously deployed files and writes
// a copy of the active configuration for external inspection. This function does NOT start or
// stop any containers – it merely ensures the on-disk representation of the application matches
// the requested version.
func (r *composeRepository) renderApp(appID, templateDir, destDir string) error {
	// Load configuration of the version to be rendered so we can compare it with the currently
	// deployed version (if any) and subsequently save a copy for external tools.
	cfgPath := filepath.Join(templateDir, "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration %s: %w", cfgPath, err)
	}

	newCfg, err := model.ParseAppConfig(data)
	if err != nil {
		return fmt.Errorf("failed to parse new configuration: %w", err)
	}

	// Remove files that belonged to the previously deployed version but are absent in the new one.
	if currentCfg, errCfg := orchestrator.GetCurrentConfig(r.config.GetAppsTemplatesPath(), appID); errCfg == nil {
		if err := r.removeDeployedFiles(destDir, currentCfg, newCfg); err != nil {
			return fmt.Errorf("failed to remove previously deployed files: %w", err)
		}
	} else if !os.IsNotExist(errCfg) {
		// An unexpected error occurred while attempting to load the active configuration – log it
		// and continue rendering instead of aborting the deployment.
		log.Warn("failed to load current configuration", "error", errCfg)
	}

	// Ensure the destination directory exists – template rendering relies on it being present.
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to ensure destination directory %s: %w", destDir, err)
	}

	// Collect substitution variables and run the rendering pipeline.
	vars, err := r.loadTemplateVariables(templateDir)
	if err != nil {
		return fmt.Errorf("failed to load template variables: %w", err)
	}

	if err := r.renderTemplates(templateDir, destDir, vars); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	// Persist a copy of the configuration that has just been rendered so that other components can
	// quickly inspect the active version without having to resolve templateDir themselves.
	if err := orchestrator.SaveCurrentConfigCopy(r.config, appID, templateDir); err != nil {
		return err
	}

	return nil
}

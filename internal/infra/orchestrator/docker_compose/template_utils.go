package docker_compose

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/infra/orchestrator"
	"winterflow-agent/pkg/env"
	"winterflow-agent/pkg/log"
	"winterflow-agent/pkg/template"
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

// renderTemplates processes template files from templateDir/files into destDir performing Docker-Compose-style
// variable substitution (see pkg/template.Substitute for supported syntax). Only files located under the
// "template" root are subject to variable substitution; files from the "router" and "user" roots are copied
// verbatim.
func (r *composeRepository) renderTemplates(templateDir, destDir string, vars map[string]string) error {
	filesRoot := filepath.Join(templateDir, "files")

	walkFn := func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(filesRoot, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		if relPath == "." {
			return nil // Skip root
		}

		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			return nil
		}

		// Always attempt variable substitution; it's a no-op when the file lacks placeholders.
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read source file %s: %w", path, err)
		}

		rendered, err := template.Substitute(string(contentBytes), vars)
		if err != nil {
			return fmt.Errorf("failed to render template %s: %w", path, err)
		}

		if err := os.WriteFile(destPath, []byte(rendered), 0o644); err != nil {
			return fmt.Errorf("failed to write file to %s: %w", destPath, err)
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

	// Generate .winterflow.env file so that compose commands can load variable values.
	if err := writeEnvFile(destDir, vars); err != nil {
		return fmt.Errorf("failed to write .winterflow.env: %w", err)
	}

	// Persist a copy of the configuration that has just been rendered so that other components can
	// quickly inspect the active version without having to resolve templateDir themselves.
	if err := orchestrator.SaveCurrentConfigCopy(r.config, appID, templateDir); err != nil {
		return err
	}

	return nil
}

// writeEnvFile creates (or overwrites) `.winterflow.env` in dir using the provided vars map.
// The file is written using a simple KEY=value format, one per line. It does NOT attempt to quote
// values – users should avoid characters that require shell escaping inside the values. This method
// is intentionally simple as the env-file format supported by Docker Compose does not mandate
// quoting unless special characters are present.
func writeEnvFile(dir string, vars map[string]string) error {
	if len(vars) == 0 {
		return nil
	}

	envPath := filepath.Join(dir, ".winterflow.env")
	return env.Save(envPath, vars)
}

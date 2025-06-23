package docker_compose

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

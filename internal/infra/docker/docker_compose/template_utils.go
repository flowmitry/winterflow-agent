package docker_compose

import (
	"encoding/json"
	"fmt"
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
	pattern := filepath.Join(templateDir, "files", "*.j2")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list template files: %w", err)
	}

	for _, src := range files {
		contentBytes, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", src, err)
		}
		content := string(contentBytes)

		// Very simple substitution – we don't need full Jinja support.
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

		dstFilename := strings.TrimSuffix(filepath.Base(src), ".j2")
		dstPath := filepath.Join(destDir, dstFilename)
		if err := os.WriteFile(dstPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write rendered template to %s: %w", dstPath, err)
		}
	}
	return nil
}

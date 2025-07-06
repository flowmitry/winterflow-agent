package env

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Save writes the provided key/value pairs to a file in .env format.
//
//   - path  - absolute or relative file path to create/overwrite.
//   - vars  - map of environment variables (keys MUST be non-empty).
//
// The function ensures deterministic ordering by sorting variable names
// alphabetically. Values containing whitespace or `#` characters are quoted
// to preserve their contents. Internal quotes and backslashes are escaped.
func Save(path string, vars map[string]string) error {
	if len(vars) == 0 {
		return nil // Nothing to write – no-op.
	}

	// Ensure destination directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create env directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create env file %s: %w", path, err)
	}
	defer f.Close()

	// Write variables in deterministic order to ease diffing.
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := vars[k]
		if strings.ContainsAny(v, " \t\n\r#") {
			// Escape internal backslashes and quotes before quoting the whole value.
			v = strings.ReplaceAll(v, `\\`, `\\\\`)
			v = strings.ReplaceAll(v, `"`, `\\"`)
			v = fmt.Sprintf("\"%s\"", v)
		}
		if _, err := fmt.Fprintf(f, "%s=%s\n", k, v); err != nil {
			return fmt.Errorf("failed to write env variable %s: %w", k, err)
		}
	}

	return nil
}

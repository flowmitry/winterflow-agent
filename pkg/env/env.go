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
		// Check for any special characters that would require quoting
		// This includes whitespace, quotes, and special characters like ?, =, etc.
		if strings.ContainsAny(v, " \t\n\r#\"'?=&$,;:{}[]()\\") {
			// Handle multiline values by replacing newlines with escaped versions
			v = strings.ReplaceAll(v, "\r\n", "\\n")
			v = strings.ReplaceAll(v, "\n", "\\n")
			v = strings.ReplaceAll(v, "\r", "\\r")

			// For Docker Compose compatibility, we need to ensure the value is properly quoted
			// and doesn't contain any unescaped special characters that could confuse the parser.
			// The safest approach is to wrap the value in double quotes and escape any internal quotes.

			// First, escape any existing backslashes to prevent double-escaping
			v = strings.ReplaceAll(v, `\`, `\\`)

			// Then escape any double quotes
			v = strings.ReplaceAll(v, `"`, `\"`)

			// Finally, wrap the entire value in double quotes
			v = fmt.Sprintf("\"%s\"", v)
		}
		if _, err := fmt.Fprintf(f, "%s=%s\n", k, v); err != nil {
			return fmt.Errorf("failed to write env variable %s: %w", k, err)
		}
	}

	return nil
}

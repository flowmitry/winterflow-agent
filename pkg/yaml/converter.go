package yaml

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// JSONToYAML converts JSON bytes to YAML bytes
func JSONToYAML(jsonBytes []byte) ([]byte, error) {
	// Parse JSON into a generic interface{}
	var jsonObj interface{}
	if err := json.Unmarshal(jsonBytes, &jsonObj); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	// Convert the parsed JSON to YAML
	var buf bytes.Buffer
	if err := writeYAML(&buf, jsonObj, 0); err != nil {
		return nil, fmt.Errorf("error converting to YAML: %w", err)
	}

	return buf.Bytes(), nil
}

// writeYAML writes a value to the buffer as YAML
func writeYAML(buf *bytes.Buffer, v interface{}, indent int) error {
	if v == nil {
		buf.WriteString("null")
		return nil
	}

	switch value := v.(type) {
	case map[string]interface{}:
		if len(value) == 0 {
			buf.WriteString("{}")
			return nil
		}

		// Sort keys for consistent output
		keys := make([]string, 0, len(value))
		for k := range value {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for i, k := range keys {
			if i > 0 {
				buf.WriteString(strings.Repeat(" ", indent))
			}
			buf.WriteString(k)
			buf.WriteString(": ")

			// For nested structures, add newline and increase indent
			switch value[k].(type) {
			case map[string]interface{}, []interface{}:
				buf.WriteString("\n")
				buf.WriteString(strings.Repeat(" ", indent+2))
				if err := writeYAML(buf, value[k], indent+2); err != nil {
					return err
				}
			default:
				if err := writeYAML(buf, value[k], indent+2); err != nil {
					return err
				}
			}
			if i < len(keys)-1 {
				buf.WriteString("\n")
			}
		}

	case []interface{}:
		if len(value) == 0 {
			buf.WriteString("[]")
			return nil
		}

		for i, item := range value {
			if i > 0 {
				buf.WriteString(strings.Repeat(" ", indent))
			}
			buf.WriteString("- ")

			// For nested structures, add newline and increase indent
			switch item.(type) {
			case map[string]interface{}, []interface{}:
				buf.WriteString("\n")
				buf.WriteString(strings.Repeat(" ", indent+2))
				if err := writeYAML(buf, item, indent+2); err != nil {
					return err
				}
			default:
				if err := writeYAML(buf, item, indent+2); err != nil {
					return err
				}
			}
			if i < len(value)-1 {
				buf.WriteString("\n")
			}
		}

	case string:
		// Check if string needs quoting
		needsQuotes := false
		for _, ch := range value {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
				needsQuotes = true
				break
			}
		}

		if needsQuotes {
			// Escape quotes in the string
			escaped := strings.ReplaceAll(value, "\"", "\\\"")
			buf.WriteString("\"")
			buf.WriteString(escaped)
			buf.WriteString("\"")
		} else {
			buf.WriteString(value)
		}

	case float64:
		// Check if it's an integer
		if value == float64(int(value)) {
			buf.WriteString(strconv.FormatInt(int64(value), 10))
		} else {
			buf.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
		}

	case bool:
		if value {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}

	default:
		return fmt.Errorf("unsupported type: %s", reflect.TypeOf(v).String())
	}

	return nil
}

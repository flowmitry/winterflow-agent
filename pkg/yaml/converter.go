package yaml

import (
	"encoding/json"
	"fmt"

	"github.com/goccy/go-yaml"
)

// JSONToYAML converts JSON bytes to YAML bytes
func JSONToYAML(jsonBytes []byte) ([]byte, error) {
	// Parse JSON into a generic interface{}
	var jsonObj interface{}
	if err := json.Unmarshal(jsonBytes, &jsonObj); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	// Use goccy/go-yaml to convert the object to YAML
	yamlBytes, err := yaml.Marshal(jsonObj)
	if err != nil {
		return nil, fmt.Errorf("error converting to YAML: %w", err)
	}

	return yamlBytes, nil
}

// YAMLToJSON converts YAML bytes to JSON bytes
func YAMLToJSON(yamlBytes []byte) ([]byte, error) {
	// Parse YAML into a generic interface{}
	var yamlObj interface{}
	if err := yaml.Unmarshal(yamlBytes, &yamlObj); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}

	// Convert the parsed YAML to JSON
	jsonBytes, err := json.Marshal(yamlObj)
	if err != nil {
		return nil, fmt.Errorf("error converting to JSON: %w", err)
	}

	return jsonBytes, nil
}

// UnmarshalYAML parses YAML bytes into the provided object
func UnmarshalYAML(yamlBytes []byte, obj interface{}) error {
	if err := yaml.Unmarshal(yamlBytes, obj); err != nil {
		return fmt.Errorf("error parsing YAML: %w", err)
	}
	return nil
}

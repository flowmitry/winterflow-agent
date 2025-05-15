package create_app

import (
	"encoding/json"
	"fmt"
)

// ReplaceIDsWithNames replaces IDs with their corresponding names from the config
// before transforming the data to YAML
func ReplaceIDsWithNames(configBytes, jsonBytes []byte) ([]byte, error) {
	// Parse config to get variable name mapping
	var config map[string]interface{}
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("error parsing config JSON: %w", err)
	}

	// Extract variables from config
	variables, ok := config["variables"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("variables not found in config or not an array")
	}

	// Create ID to name mapping
	idToName := make(map[string]string)
	for _, v := range variables {
		variable, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		id, ok1 := variable["id"].(string)
		name, ok2 := variable["name"].(string)
		if ok1 && ok2 {
			idToName[id] = name
		}
	}

	// Parse variables/secrets JSON
	var jsonObj map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &jsonObj); err != nil {
		return nil, fmt.Errorf("error parsing variables/secrets JSON: %w", err)
	}

	// Create new map with variable names as keys
	namedMap := make(map[string]interface{})
	for id, value := range jsonObj {
		name, ok := idToName[id]
		if ok {
			namedMap[name] = value
		} else {
			// Keep original ID if name not found
			namedMap[id] = value
		}
	}

	// Convert the named map to JSON
	namedJSON, err := json.Marshal(namedMap)
	if err != nil {
		return nil, fmt.Errorf("error marshaling named map to JSON: %w", err)
	}

	return namedJSON, nil
}

package get_app

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"winterflow-agent/internal/winterflow/grpc/pb"
)

// convertYAMLToIDValueJSON converts YAML to JSON with "id": "value" format
func convertYAMLToIDValueJSON(configBytes, yamlBytes []byte) ([]byte, error) {
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

	// Create name to ID mapping
	nameToID := make(map[string]string)
	for _, v := range variables {
		variable, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		id, ok1 := variable["id"].(string)
		name, ok2 := variable["name"].(string)
		if ok1 && ok2 {
			nameToID[name] = id
		}
	}

	// Parse YAML by simple line parsing
	// This is a simplified approach that works for the specific YAML format used in the project
	yamlStr := string(yamlBytes)
	lines := strings.Split(yamlStr, "\n")

	// Create new map with variable IDs as keys
	idMap := make(map[string]interface{})
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue // Skip invalid lines
		}

		name := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])

		// Try to parse the value as a number or boolean
		var value interface{} = valueStr
		if valueStr == "true" {
			value = true
		} else if valueStr == "false" {
			value = false
		} else if i, err := strconv.Atoi(valueStr); err == nil {
			value = i
		} else if f, err := strconv.ParseFloat(valueStr, 64); err == nil {
			value = f
		} else {
			// Remove quotes if present
			if (strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"")) ||
				(strings.HasPrefix(valueStr, "'") && strings.HasSuffix(valueStr, "'")) {
				value = valueStr[1 : len(valueStr)-1]
			}
		}

		// Map the name to ID if possible
		id, ok := nameToID[name]
		if ok {
			idMap[id] = value
		} else {
			// Keep original name if ID not found
			idMap[name] = value
		}
	}

	// Convert the ID map to JSON
	idJSON, err := json.Marshal(idMap)
	if err != nil {
		return nil, fmt.Errorf("error marshaling ID map to JSON: %w", err)
	}

	return idJSON, nil
}

// convertJSONToAppVars converts a JSON byte array to a slice of AppVarV1
func convertJSONToAppVars(jsonBytes []byte) ([]*pb.AppVarV1, error) {
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	var appVars []*pb.AppVarV1
	for id, value := range jsonMap {
		// Convert the value to string
		var strValue string
		switch v := value.(type) {
		case string:
			strValue = v
		case bool:
			strValue = fmt.Sprintf("%t", v)
		case int:
			strValue = fmt.Sprintf("%d", v)
		case float64:
			strValue = fmt.Sprintf("%g", v)
		default:
			strValue = fmt.Sprintf("%v", v)
		}

		appVar := &pb.AppVarV1{
			Id:      id,
			Content: []byte(strValue),
		}
		appVars = append(appVars, appVar)
	}

	return appVars, nil
}

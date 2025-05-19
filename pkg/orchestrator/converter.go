package orchestrator

import (
	"encoding/json"
	"fmt"
	"winterflow-agent/pkg/yaml"
)

// SupportedOrchestratorTypes defines the supported orchestrator types
var SupportedOrchestratorTypes = []string{"docker_compose", "docker_swarm"}

// Convert converts a file content from docker_swarm format to the target orchestrator type
// Supported target types: "docker_compose", "docker_swarm"
func Convert(fileContent []byte, targetType string) ([]byte, error) {
	// Validate target type
	if !isValidTargetType(targetType) {
		return nil, fmt.Errorf("unsupported target orchestrator type: %s", targetType)
	}

	// If target type is docker_swarm, return the original content (identity transformation)
	if targetType == "docker_swarm" {
		return fileContent, nil
	}

	// Parse the YAML input using goccy/go-yaml
	var swarmConfig map[string]interface{}
	if err := yaml.UnmarshalYAML(fileContent, &swarmConfig); err != nil {
		return nil, fmt.Errorf("error parsing Docker Swarm configuration: %w", err)
	}

	// Convert from docker_swarm to docker_compose
	composeConfig, err := swarmToCompose(swarmConfig)
	if err != nil {
		return nil, fmt.Errorf("error converting from Docker Swarm to Docker Compose: %w", err)
	}

	// Convert to JSON
	jsonBytes, err := json.Marshal(composeConfig)
	if err != nil {
		return nil, fmt.Errorf("error marshaling to JSON: %w", err)
	}

	// Convert JSON to YAML
	result, err := yaml.JSONToYAML(jsonBytes)
	if err != nil {
		return nil, fmt.Errorf("error converting JSON to YAML: %w", err)
	}

	return result, nil
}

// isValidTargetType checks if the target type is supported
func isValidTargetType(targetType string) bool {
	for _, t := range SupportedOrchestratorTypes {
		if t == targetType {
			return true
		}
	}
	return false
}

// swarmToCompose converts a Docker Swarm configuration to Docker Compose format
func swarmToCompose(swarmConfig map[string]interface{}) (map[string]interface{}, error) {
	composeConfig := make(map[string]interface{})

	// Copy version if present
	if version, ok := swarmConfig["version"]; ok {
		composeConfig["version"] = version
	} else {
		// Default to compose file version 3
		composeConfig["version"] = "3"
	}

	// Copy services if present
	if services, ok := swarmConfig["services"].(map[string]interface{}); ok {
		composeServices := make(map[string]interface{})

		for serviceName, serviceConfig := range services {
			serviceMap, ok := serviceConfig.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid service configuration for %s", serviceName)
			}

			// Create a new service configuration for Docker Compose
			composeService := make(map[string]interface{})

			// Copy common fields
			for key, value := range serviceMap {
				// Skip Docker Swarm specific fields
				if isSwarmSpecificField(key) {
					continue
				}
				composeService[key] = value
			}

			// Handle deploy section if present
			if deploy, ok := serviceMap["deploy"].(map[string]interface{}); ok {
				// Extract relevant information from deploy section
				handleDeploySection(deploy, composeService)
			}

			composeServices[serviceName] = composeService
		}

		composeConfig["services"] = composeServices
	}

	// Copy networks if present
	if networks, ok := swarmConfig["networks"].(map[string]interface{}); ok {
		composeNetworks := make(map[string]interface{})

		for networkName, networkConfig := range networks {
			networkMap, ok := networkConfig.(map[string]interface{})
			if !ok {
				// If it's not a map, just copy it as is
				composeNetworks[networkName] = networkConfig
				continue
			}

			// Create a new network configuration for Docker Compose
			composeNetwork := make(map[string]interface{})

			// Copy common fields
			for key, value := range networkMap {
				// Skip Docker Swarm specific fields
				if isSwarmSpecificNetworkField(key) {
					continue
				}
				composeNetwork[key] = value
			}

			composeNetworks[networkName] = composeNetwork
		}

		composeConfig["networks"] = composeNetworks
	}

	// Copy volumes if present
	if volumes, ok := swarmConfig["volumes"].(map[string]interface{}); ok {
		composeConfig["volumes"] = volumes
	}

	// Copy configs if present (as configs are not supported in Docker Compose, convert to volumes)
	if configs, ok := swarmConfig["configs"].(map[string]interface{}); ok && len(configs) > 0 {
		// If volumes doesn't exist, create it
		if _, ok := composeConfig["volumes"]; !ok {
			composeConfig["volumes"] = make(map[string]interface{})
		}

		volumes := composeConfig["volumes"].(map[string]interface{})
		for configName := range configs {
			// Add a comment that this was converted from a config
			volumes[configName+"_config"] = map[string]interface{}{
				"driver": "local",
				"driver_opts": map[string]interface{}{
					"type":   "none",
					"device": "/path/to/config/" + configName,
					"o":      "bind",
				},
			}
		}
	}

	// Copy secrets if present (as secrets are not supported in Docker Compose, convert to volumes)
	if secrets, ok := swarmConfig["secrets"].(map[string]interface{}); ok && len(secrets) > 0 {
		// If volumes doesn't exist, create it
		if _, ok := composeConfig["volumes"]; !ok {
			composeConfig["volumes"] = make(map[string]interface{})
		}

		volumes := composeConfig["volumes"].(map[string]interface{})
		for secretName := range secrets {
			// Add a comment that this was converted from a secret
			volumes[secretName+"_secret"] = map[string]interface{}{
				"driver": "local",
				"driver_opts": map[string]interface{}{
					"type":   "none",
					"device": "/path/to/secret/" + secretName,
					"o":      "bind",
				},
			}
		}
	}

	return composeConfig, nil
}

// isSwarmSpecificField checks if a field is specific to Docker Swarm
func isSwarmSpecificField(field string) bool {
	swarmSpecificFields := []string{
		"deploy",
		"configs",
		"secrets",
	}

	for _, f := range swarmSpecificFields {
		if f == field {
			return true
		}
	}
	return false
}

// isSwarmSpecificNetworkField checks if a network field is specific to Docker Swarm
func isSwarmSpecificNetworkField(field string) bool {
	swarmSpecificFields := []string{
		"attachable",
		"driver_opts",
		"labels",
	}

	for _, f := range swarmSpecificFields {
		if f == field {
			return true
		}
	}
	return false
}

// handleDeploySection extracts relevant information from the deploy section
// and adds it to the Docker Compose service configuration
func handleDeploySection(deploy map[string]interface{}, composeService map[string]interface{}) {
	// Handle replicas
	if replicas, ok := deploy["replicas"]; ok {
		if replicasFloat, ok := replicas.(float64); ok {
			composeService["scale"] = int(replicasFloat)
		} else if replicasInt, ok := replicas.(int); ok {
			composeService["scale"] = replicasInt
		} else if replicasUint, ok := replicas.(uint64); ok {
			composeService["scale"] = int(replicasUint)
		}
	}

	// Handle resources
	if resources, ok := deploy["resources"].(map[string]interface{}); ok {
		// Handle resource limits
		if limits, ok := resources["limits"].(map[string]interface{}); ok {
			if _, ok := composeService["mem_limit"]; !ok {
				if memory, ok := limits["memory"].(string); ok {
					composeService["mem_limit"] = memory
				}
			}
			if _, ok := composeService["cpus"]; !ok {
				if cpus, ok := limits["cpus"].(string); ok {
					composeService["cpus"] = cpus
				}
			}
		}

		// Handle resource reservations
		if reservations, ok := resources["reservations"].(map[string]interface{}); ok {
			if _, ok := composeService["mem_reservation"]; !ok {
				if memory, ok := reservations["memory"].(string); ok {
					composeService["mem_reservation"] = memory
				}
			}
		}
	}

	// Handle restart policy
	if restartPolicy, ok := deploy["restart_policy"].(map[string]interface{}); ok {
		if condition, ok := restartPolicy["condition"].(string); ok {
			// Map Swarm restart conditions to Compose restart values
			switch condition {
			case "on-failure":
				composeService["restart"] = "on-failure"
				if maxAttempts, ok := restartPolicy["max_attempts"]; ok {
					if maxAttemptsFloat, ok := maxAttempts.(float64); ok {
						composeService["restart"] = fmt.Sprintf("on-failure:%d", int(maxAttemptsFloat))
					} else if maxAttemptsInt, ok := maxAttempts.(int); ok {
						composeService["restart"] = fmt.Sprintf("on-failure:%d", maxAttemptsInt)
					} else if maxAttemptsUint, ok := maxAttempts.(uint64); ok {
						composeService["restart"] = fmt.Sprintf("on-failure:%d", maxAttemptsUint)
					}
				}
			case "any":
				composeService["restart"] = "always"
			case "none":
				composeService["restart"] = "no"
			}
		}
	}

	// Handle placement constraints
	if placement, ok := deploy["placement"].(map[string]interface{}); ok {
		if constraints, ok := placement["constraints"].([]interface{}); ok && len(constraints) > 0 {
			// Convert constraints to environment variables or labels
			// as Docker Compose doesn't support placement constraints directly
			if _, ok := composeService["environment"]; !ok {
				composeService["environment"] = make([]string, 0)
			}
			env := composeService["environment"].([]string)

			for _, constraint := range constraints {
				if constraintStr, ok := constraint.(string); ok {
					env = append(env, "CONSTRAINT_"+constraintStr)
				}
			}
			composeService["environment"] = env
		}
	}
}

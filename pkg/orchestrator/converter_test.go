package orchestrator

import (
    "testing"
    "winterflow-agent/pkg/yaml"
)

func TestConvert(t *testing.T) {
    tests := []struct {
        name       string
        input      string
        targetType string
        wantErr    bool
    }{
        {
            name: "Convert docker_swarm to docker_compose",
            input: `
version: "3"
services:
  web:
    image: nginx:latest
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: "0.5"
          memory: "512M"
      restart_policy:
        condition: on-failure
        max_attempts: 3
    ports:
      - "80:80"
networks:
  frontend:
    driver: overlay
volumes:
  data:
`,
            targetType: "docker_compose",
            wantErr:    false,
        },
        {
            name: "Identity transformation (docker_swarm to docker_swarm)",
            input: `
version: "3"
services:
  web:
    image: nginx:latest
`,
            targetType: "docker_swarm",
            wantErr:    false,
        },
        {
            name:       "Invalid target type",
            input:      "version: '3'",
            targetType: "invalid_type",
            wantErr:    true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Convert([]byte(tt.input), tt.targetType)
            if (err != nil) != tt.wantErr {
                t.Errorf("Convert() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if err != nil {
                return
            }

            // For identity transformation, the result should be the same as the input
            if tt.targetType == "docker_swarm" {
                if string(result) != tt.input {
                    t.Errorf("Convert() result = %v, want %v", string(result), tt.input)
                }
                return
            }

            // For conversion to docker_compose, verify that the result is valid YAML
            var resultMap map[string]interface{}
            if err := yaml.UnmarshalYAML(result, &resultMap); err != nil {
                t.Errorf("Convert() result is not valid YAML: %v", err)
                return
            }

            // Verify that the result contains the expected keys
            if _, ok := resultMap["version"]; !ok {
                t.Errorf("Convert() result does not contain 'version' key")
            }

            if services, ok := resultMap["services"].(map[string]interface{}); ok {
                // Verify that the services were converted correctly
                if web, ok := services["web"].(map[string]interface{}); ok {
                    // Verify that deploy section was removed
                    if _, ok := web["deploy"]; ok {
                        t.Errorf("Convert() result still contains 'deploy' key")
                    }

                    // Verify that replicas were converted to scale
                    t.Logf("web keys: %v", web)
                    if scale, ok := web["scale"]; ok {
                        t.Logf("scale type: %T, value: %v", scale, scale)
                        // Check if it's a float64 (common in JSON/YAML parsing)
                        if scaleFloat, ok := scale.(float64); ok {
                            if scaleFloat != 3 {
                                t.Errorf("Convert() scale = %v, want %v", scaleFloat, 3)
                            }
                        } else if scaleInt, ok := scale.(int); ok {
                            if scaleInt != 3 {
                                t.Errorf("Convert() scale = %v, want %v", scaleInt, 3)
                            }
                        } else {
                            t.Errorf("Convert() scale has unexpected type: %T", scale)
                        }
                    } else {
                        t.Errorf("Convert() result does not contain 'scale' key")
                    }

                    // Verify that resource limits were converted
                    if cpus, ok := web["cpus"].(string); ok {
                        if cpus != "0.5" {
                            t.Errorf("Convert() cpus = %v, want %v", cpus, "0.5")
                        }
                    } else {
                        t.Errorf("Convert() result does not contain 'cpus' key")
                    }

                    if memLimit, ok := web["mem_limit"].(string); ok {
                        if memLimit != "512M" {
                            t.Errorf("Convert() mem_limit = %v, want %v", memLimit, "512M")
                        }
                    } else {
                        t.Errorf("Convert() result does not contain 'mem_limit' key")
                    }

                    // Verify that restart policy was converted
                    if restart, ok := web["restart"]; ok {
                        t.Logf("restart type: %T, value: %v", restart, restart)
                        if restartStr, ok := restart.(string); ok {
                            if restartStr != "on-failure:3" {
                                t.Errorf("Convert() restart = %v, want %v", restartStr, "on-failure:3")
                            }
                        } else {
                            t.Errorf("Convert() restart has unexpected type: %T", restart)
                        }
                    } else {
                        t.Errorf("Convert() result does not contain 'restart' key")
                    }
                } else {
                    t.Errorf("Convert() result does not contain 'web' service")
                }
            } else {
                t.Errorf("Convert() result does not contain 'services' key")
            }
        })
    }
}

func TestIsValidTargetType(t *testing.T) {
    tests := []struct {
        name       string
        targetType string
        want       bool
    }{
        {
            name:       "Valid target type: docker_compose",
            targetType: "docker_compose",
            want:       true,
        },
        {
            name:       "Valid target type: docker_swarm",
            targetType: "docker_swarm",
            want:       true,
        },
        {
            name:       "Invalid target type",
            targetType: "invalid_type",
            want:       false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := isValidTargetType(tt.targetType); got != tt.want {
                t.Errorf("isValidTargetType() = %v, want %v", got, tt.want)
            }
        })
    }
}

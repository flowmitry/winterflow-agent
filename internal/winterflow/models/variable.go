package models

import (
	"encoding/json"
	"winterflow-agent/internal/winterflow/grpc/pb"
)

// VariableMap represents a map of variable UUIDs to values
type VariableMap map[string]string

// ParseVariableMap parses a variable map from JSON bytes
// This is kept for backward compatibility
func ParseVariableMap(bytes []byte) (VariableMap, error) {
	var variableMap VariableMap
	err := json.Unmarshal(bytes, &variableMap)
	if err != nil {
		return nil, err
	}
	return variableMap, nil
}

// ParseVariableMapFromProto converts a repeated AppVarV1 to a VariableMap
func ParseVariableMapFromProto(vars []*pb.AppVarV1) VariableMap {
	variableMap := make(VariableMap)
	for _, v := range vars {
		variableMap[v.Id] = string(v.Content)
	}
	return variableMap
}

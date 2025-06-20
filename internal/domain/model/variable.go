package model

import (
	"winterflow-agent/internal/infra/winterflow/grpc/pb"
)

// VariableMap represents a map of variable UUIDs to values
type VariableMap map[string]string

// ParseVariableMapFromProto converts a repeated AppVarV1 to a VariableMap
func ParseVariableMapFromProto(vars []*pb.AppVarV1) VariableMap {
	variableMap := make(VariableMap)
	for _, v := range vars {
		variableMap[v.Id] = string(v.Content)
	}
	return variableMap
}

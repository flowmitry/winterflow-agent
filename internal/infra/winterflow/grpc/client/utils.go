package client

import (
	"errors"
	"github.com/google/uuid"
	"time"
	"winterflow-agent/internal/infra/winterflow/grpc/pb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Default reconnection parameters
	DefaultReconnectInterval        = 5 * time.Second
	DefaultMaximumReconnectInterval = 320 * time.Second
	DefaultConnectionTimeout        = 30 * time.Second
	HeartbeatInterval               = 10 * time.Second // unified heartbeat cadence
)

// ErrUnrecoverable is returned by RegisterAgent when the server indicates that
// the agent must not retry the registration (e.g. wrong server-token pairing
// or duplicate agent).
var ErrUnrecoverable = errors.New("unrecoverable error. check your server ID and token")
var ErrUnrecoverableAgentAlreadyConnected = errors.New("unrecoverable error: agent already connected")

// GenerateUUID generates a random UUID v4
func GenerateUUID() string {
	return uuid.New().String()
}

// TimestampNow returns the current time as a protobuf Timestamp
func TimestampNow() *timestamppb.Timestamp {
	return timestamppb.Now()
}

func createBaseResponse(messageID string, agentID string, code pb.ResponseCode, message string) pb.BaseResponse {
	return pb.BaseResponse{
		MessageId:    messageID,
		Timestamp:    TimestampNow(),
		ResponseCode: code,
		Message:      message,
		AgentId:      agentID,
	}
}

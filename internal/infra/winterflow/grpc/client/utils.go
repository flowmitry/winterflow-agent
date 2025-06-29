package client

import (
	"errors"
	"fmt"
	"time"
	"winterflow-agent/internal/infra/winterflow/grpc/pb"
	"winterflow-agent/pkg/log"

	"github.com/google/uuid"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Default reconnection parameters
	DefaultReconnectInterval        = 5 * time.Second
	DefaultMaximumReconnectInterval = 320 * time.Second
	DefaultConnectionTimeout        = 30 * time.Second
	HeartbeatInterval               = 10 * time.Second // unified heartbeat cadence
	MetricsInterval                 = 60 * time.Second // interval for sending metrics
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

// extractBaseMessageFromCommand attempts to extract the *pb.BaseMessage from any of the known
// ServerCommand oneof wrappers. It returns nil if the command type does not contain a BaseMessage.
func extractBaseMessageFromCommand(command interface{}) *pb.BaseMessage {
	switch cmd := command.(type) {
	case *pb.ServerCommand_UpdateAgentRequestV1:
		return cmd.UpdateAgentRequestV1.GetBase()
	case *pb.ServerCommand_GetAppRequestV1:
		return cmd.GetAppRequestV1.GetBase()
	case *pb.ServerCommand_SaveAppRequestV1:
		return cmd.SaveAppRequestV1.GetBase()
	case *pb.ServerCommand_RenameAppRequestV1:
		return cmd.RenameAppRequestV1.GetBase()
	case *pb.ServerCommand_DeleteAppRequestV1:
		return cmd.DeleteAppRequestV1.GetBase()
	case *pb.ServerCommand_ControlAppRequestV1:
		return cmd.ControlAppRequestV1.GetBase()
	case *pb.ServerCommand_GetAppsStatusRequestV1:
		return cmd.GetAppsStatusRequestV1.GetBase()
	case *pb.ServerCommand_GetRegistriesRequestV1:
		return cmd.GetRegistriesRequestV1.GetBase()
	case *pb.ServerCommand_CreateRegistryRequestV1:
		return cmd.CreateRegistryRequestV1.GetBase()
	case *pb.ServerCommand_DeleteRegistryRequestV1:
		return cmd.DeleteRegistryRequestV1.GetBase()
	case *pb.ServerCommand_GetNetworksRequestV1:
		return cmd.GetNetworksRequestV1.GetBase()
	case *pb.ServerCommand_CreateNetworkRequestV1:
		return cmd.CreateNetworkRequestV1.GetBase()
	case *pb.ServerCommand_DeleteNetworkRequestV1:
		return cmd.DeleteNetworkRequestV1.GetBase()
	default:
		return nil
	}
}

// buildUnauthorizedAgentMessage constructs an AgentMessage with RESPONSE_CODE_UNAUTHORIZED for the
// provided command. Returns nil if the command type is not supported.
func buildUnauthorizedAgentMessage(command interface{}, messageID, agentID string) *pb.AgentMessage {
	baseResp := createBaseResponse(messageID, agentID, pb.ResponseCode_RESPONSE_CODE_UNAUTHORIZED, "Agent ID mismatch")

	switch cmd := command.(type) {
	case *pb.ServerCommand_UpdateAgentRequestV1:
		resp := &pb.UpdateAgentResponseV1{Base: &baseResp}
		return &pb.AgentMessage{Message: &pb.AgentMessage_UpdateAgentResponseV1{UpdateAgentResponseV1: resp}}
	case *pb.ServerCommand_GetAppRequestV1:
		resp := &pb.GetAppResponseV1{Base: &baseResp}
		return &pb.AgentMessage{Message: &pb.AgentMessage_GetAppResponseV1{GetAppResponseV1: resp}}
	case *pb.ServerCommand_SaveAppRequestV1:
		resp := &pb.SaveAppResponseV1{Base: &baseResp}
		return &pb.AgentMessage{Message: &pb.AgentMessage_SaveAppResponseV1{SaveAppResponseV1: resp}}
	case *pb.ServerCommand_RenameAppRequestV1:
		resp := &pb.RenameAppResponseV1{Base: &baseResp}
		return &pb.AgentMessage{Message: &pb.AgentMessage_RenameAppResponseV1{RenameAppResponseV1: resp}}
	case *pb.ServerCommand_DeleteAppRequestV1:
		resp := &pb.DeleteAppResponseV1{Base: &baseResp}
		return &pb.AgentMessage{Message: &pb.AgentMessage_DeleteAppResponseV1{DeleteAppResponseV1: resp}}
	case *pb.ServerCommand_ControlAppRequestV1:
		resp := &pb.ControlAppResponseV1{Base: &baseResp}
		return &pb.AgentMessage{Message: &pb.AgentMessage_ControlAppResponseV1{ControlAppResponseV1: resp}}
	case *pb.ServerCommand_GetAppsStatusRequestV1:
		resp := &pb.GetAppsStatusResponseV1{Base: &baseResp}
		return &pb.AgentMessage{Message: &pb.AgentMessage_GetAppsStatusResponseV1{GetAppsStatusResponseV1: resp}}
	case *pb.ServerCommand_GetRegistriesRequestV1:
		resp := &pb.GetRegistriesResponseV1{Base: &baseResp}
		return &pb.AgentMessage{Message: &pb.AgentMessage_GetRegistriesResponseV1{GetRegistriesResponseV1: resp}}
	case *pb.ServerCommand_CreateRegistryRequestV1:
		resp := &pb.CreateRegistryResponseV1{Base: &baseResp}
		return &pb.AgentMessage{Message: &pb.AgentMessage_CreateRegistryResponseV1{CreateRegistryResponseV1: resp}}
	case *pb.ServerCommand_DeleteRegistryRequestV1:
		resp := &pb.DeleteRegistryResponseV1{Base: &baseResp}
		return &pb.AgentMessage{Message: &pb.AgentMessage_DeleteRegistryResponseV1{DeleteRegistryResponseV1: resp}}
	case *pb.ServerCommand_GetNetworksRequestV1:
		resp := &pb.GetNetworksResponseV1{Base: &baseResp}
		return &pb.AgentMessage{Message: &pb.AgentMessage_GetNetworksResponseV1{GetNetworksResponseV1: resp}}
	case *pb.ServerCommand_CreateNetworkRequestV1:
		resp := &pb.CreateNetworkResponseV1{Base: &baseResp}
		return &pb.AgentMessage{Message: &pb.AgentMessage_CreateNetworkResponseV1{CreateNetworkResponseV1: resp}}
	case *pb.ServerCommand_DeleteNetworkRequestV1:
		resp := &pb.DeleteNetworkResponseV1{Base: &baseResp}
		return &pb.AgentMessage{Message: &pb.AgentMessage_DeleteNetworkResponseV1{DeleteNetworkResponseV1: resp}}
	default:
		log.Debug("Unsupported command type for unauthorized response", "type", fmt.Sprintf("%T", cmd))
		return nil
	}
}

// ValidateAndRespondAgentID validates that the command targets this agent. If the agent IDs do not match,
// it sends an unauthorized response back through the provided stream and returns false to indicate that the
// caller should ignore the command. When validation succeeds it returns true.
func ValidateAndRespondAgentID(stream pb.AgentService_AgentStreamClient, command interface{}, expectedAgentID string) bool {
	base := extractBaseMessageFromCommand(command)
	if base == nil {
		return true // nothing to validate
	}

	if base.GetAgentId() == expectedAgentID {
		return true
	}

	log.Warn("Received command for different agent ID, ignoring", "expectedAgentID", expectedAgentID, "commandAgentID", base.GetAgentId())

	if agentMsg := buildUnauthorizedAgentMessage(command, base.GetMessageId(), expectedAgentID); agentMsg != nil {
		if err := stream.Send(agentMsg); err != nil {
			log.Warn("Error sending unauthorized response", "error", err)
		} else {
			log.Info("Unauthorized response sent successfully")
		}
	}

	return false
}

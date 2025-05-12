package client

import (
	"fmt"
	"winterflow-agent/internal/winterflow/grpc/pb"
	"winterflow-agent/internal/winterflow/handlers/create_app"
	"winterflow-agent/pkg/cqrs"
	log "winterflow-agent/pkg/log"
)

// HandleCreateAppRequest handles the command dispatch and creates the appropriate response message
func HandleCreateAppRequest(commandBus cqrs.CommandBus, createAppRequest *pb.CreateAppRequestV1, serverID string) (*pb.AgentMessage, error) {
	log.Debug("Processing create app request for app ID: %s", createAppRequest.App.AppId)

	// Create and dispatch the command
	cmd := create_app.CreateAppCommand{Request: createAppRequest}

	var responseCode pb.ResponseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage string = "App created successfully"

	// Dispatch the command to the handler
	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error creating app: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error creating app: %v", err)
	}

	// Create response
	baseResp := &pb.BaseResponse{
		MessageId:    createAppRequest.Base.MessageId,
		Timestamp:    TimestampNow(),
		ResponseCode: responseCode,
		Message:      responseMessage,
		ServerId:     serverID,
	}

	createAppResp := &pb.CreateAppResponseV1{
		Base: baseResp,
		App:  createAppRequest.App,
	}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_CreateAppResponseV1{
			CreateAppResponseV1: createAppResp,
		},
	}

	return agentMsg, nil
}

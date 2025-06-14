package client

import (
	"fmt"
	"winterflow-agent/internal/winterflow/grpc/pb"
	"winterflow-agent/internal/winterflow/handlers/control_app"
	"winterflow-agent/internal/winterflow/handlers/delete_app"
	"winterflow-agent/internal/winterflow/handlers/save_app"
	"winterflow-agent/internal/winterflow/handlers/update_agent"
	"winterflow-agent/pkg/cqrs"
	log "winterflow-agent/pkg/log"
)

// HandleSaveAppRequest handles the command dispatch and creates the appropriate response message
func HandleSaveAppRequest(commandBus cqrs.CommandBus, saveAppRequest *pb.SaveAppRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing save app request for app ID: %s", saveAppRequest.App.AppId)

	// Create and dispatch the command
	cmd := save_app.SaveAppCommand{Request: saveAppRequest}

	var responseCode pb.ResponseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage string = "App saved successfully"

	// Dispatch the command to the handler
	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error saving app: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error saving app: %v", err)
	}

	baseResp := createBaseResponse(saveAppRequest.Base.MessageId, agentID, responseCode, responseMessage)
	saveAppResp := &pb.SaveAppResponseV1{
		Base: &baseResp,
		App:  saveAppRequest.App,
	}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_SaveAppResponseV1{
			SaveAppResponseV1: saveAppResp,
		},
	}

	return agentMsg, nil
}

// HandleDeleteAppRequest handles the command dispatch and creates the appropriate response message
func HandleDeleteAppRequest(commandBus cqrs.CommandBus, deleteAppRequest *pb.DeleteAppRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing delete app request for app ID: %s", deleteAppRequest.AppId)

	// Create and dispatch the command
	cmd := delete_app.DeleteAppCommand{Request: deleteAppRequest}

	var responseCode pb.ResponseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage string = "App deleted successfully"

	// Dispatch the command to the handler
	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error deleting app: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error deleting app: %v", err)
	}

	baseResp := createBaseResponse(deleteAppRequest.Base.MessageId, agentID, responseCode, responseMessage)
	deleteAppResp := &pb.DeleteAppResponseV1{
		Base: &baseResp,
	}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_DeleteAppResponseV1{
			DeleteAppResponseV1: deleteAppResp,
		},
	}

	return agentMsg, nil
}

// HandleControlAppRequest handles the command dispatch and creates the appropriate response message
func HandleControlAppRequest(commandBus cqrs.CommandBus, controlAppRequest *pb.ControlAppRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing control app request for app ID: %s, action: %v", controlAppRequest.AppId, controlAppRequest.Action)

	// Create and dispatch the command
	cmd := control_app.ControlAppCommand{Request: controlAppRequest}

	var responseCode pb.ResponseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage string = "App control action executed successfully"
	var statusCode pb.AppStatusCode = pb.AppStatusCode_STATUS_CODE_ACTIVE

	// Dispatch the command to the handler
	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error controlling app: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error controlling app: %v", err)
	}

	baseResp := createBaseResponse(controlAppRequest.Base.MessageId, agentID, responseCode, responseMessage)
	controlAppResp := &pb.ControlAppResponseV1{
		Base:       &baseResp,
		AppId:      controlAppRequest.AppId,
		AppVersion: controlAppRequest.AppVersion,
		StatusCode: statusCode,
	}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_ControlAppResponseV1{
			ControlAppResponseV1: controlAppResp,
		},
	}

	return agentMsg, nil
}

// HandleUpdateAgentRequest handles the command dispatch and creates the appropriate response message
func HandleUpdateAgentRequest(commandBus cqrs.CommandBus, updateAgentRequest *pb.UpdateAgentRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing update agent request for version: %s", updateAgentRequest.Version)

	// Create and dispatch the command
	cmd := update_agent.UpdateAgentCommand{Request: updateAgentRequest}

	var responseCode pb.ResponseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage string = "Agent update initiated successfully"

	// Dispatch the command to the handler
	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error updating agent: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error updating agent: %v", err)
	}

	baseResp := createBaseResponse(updateAgentRequest.Base.MessageId, agentID, responseCode, responseMessage)
	updateAgentResp := &pb.UpdateAgentResponseV1{
		Base: &baseResp,
	}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_UpdateAgentResponseV1{
			UpdateAgentResponseV1: updateAgentResp,
		},
	}

	return agentMsg, nil
}

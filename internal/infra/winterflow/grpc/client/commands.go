package client

import (
	"fmt"
	"winterflow-agent/internal/application/command/delete_app"
	"winterflow-agent/internal/application/command/save_app"
	"winterflow-agent/internal/application/command/update_agent"
	"winterflow-agent/internal/infra/winterflow/grpc/pb"
	"winterflow-agent/pkg/cqrs"
	log "winterflow-agent/pkg/log"
)

// HandleSaveAppRequest handles the command dispatch and creates the appropriate response message
func HandleSaveAppRequest(commandBus cqrs.CommandBus, saveAppRequest *pb.SaveAppRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing save app request", "app_id", saveAppRequest.App.AppId)
	app := ProtoAppV1ToApp(saveAppRequest.App)
	// Create and dispatch the command
	cmd := save_app.SaveAppCommand{
		App: app,
	}

	var responseCode pb.ResponseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage string = "App saved successfully"

	// Dispatch the command to the handler
	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error saving app", "error", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error saving app: %v", err)
	}

	baseResp := createBaseResponse(saveAppRequest.Base.MessageId, agentID, responseCode, responseMessage)
	saveAppResp := &pb.SaveAppResponseV1{
		Base: &baseResp,
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
	log.Debug("Processing delete app request", "app_id", deleteAppRequest.AppId)

	// Create and dispatch the command
	cmd := delete_app.DeleteAppCommand{
		AppID: deleteAppRequest.AppId,
	}

	var responseCode pb.ResponseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage string = "App deleted successfully"

	// Dispatch the command to the handler
	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error deleting app", "error", err)
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
	log.Debug("Processing control app request", "app_id", controlAppRequest.AppId, "action", controlAppRequest.Action)

	// Create and dispatch the command
	cmd := ProtoControlAppRequestV1ToControlAppCommand(controlAppRequest)

	var responseCode pb.ResponseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage string = "App control action executed successfully"

	// Dispatch the command to the handler
	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error controlling app", "error", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error controlling app: %v", err)
	}

	baseResp := createBaseResponse(controlAppRequest.Base.MessageId, agentID, responseCode, responseMessage)
	controlAppResp := &pb.ControlAppResponseV1{
		Base: &baseResp,
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
	log.Debug("Processing update agent request", "version", updateAgentRequest.Version)

	// Create and dispatch the command
	cmd := update_agent.UpdateAgentCommand{
		Version: updateAgentRequest.Version,
	}

	var responseCode pb.ResponseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage string = "Agent update initiated successfully"

	// Dispatch the command to the handler
	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error updating agent", "error", err)
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

package client

import (
	"fmt"
	"winterflow-agent/internal/application/command/create_network"
	"winterflow-agent/internal/application/command/create_registry"
	"winterflow-agent/internal/application/command/delete_app"
	"winterflow-agent/internal/application/command/delete_network"
	"winterflow-agent/internal/application/command/delete_registry"
	"winterflow-agent/internal/application/command/save_app"
	"winterflow-agent/internal/application/command/update_agent"
	"winterflow-agent/internal/infra/winterflow/grpc/pb"
	"winterflow-agent/pkg/cqrs"
	"winterflow-agent/pkg/log"
)

// HandleSaveAppRequest handles the command dispatch and creates the appropriate response message
func HandleSaveAppRequest(commandBus cqrs.CommandBus, saveAppRequest *pb.SaveAppRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing save app request", "app_id", saveAppRequest.App.AppId)
	app := ProtoAppV1ToApp(saveAppRequest.App)
	// Create and dispatch the command
	cmd := save_app.SaveAppCommand{
		App: app,
	}

	var responseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage = "App saved successfully"

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

	var responseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage = "App deleted successfully"

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

	var responseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage = "App control action executed successfully"

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

	var responseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage = "Agent update initiated successfully"

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

// HandleRenameAppRequest handles the command dispatch and creates the appropriate response message
func HandleRenameAppRequest(commandBus cqrs.CommandBus, renameAppRequest *pb.RenameAppRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing rename app request", "app_id", renameAppRequest.AppId, "app_name", renameAppRequest.AppName)

	// Create and dispatch the command
	cmd := ProtoRenameAppRequestV1ToRenameAppCommand(renameAppRequest)

	var responseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage = "App renamed successfully"

	// Dispatch the command to the handler
	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error renaming app", "error", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error renaming app: %v", err)
	}

	baseResp := createBaseResponse(renameAppRequest.Base.MessageId, agentID, responseCode, responseMessage)
	renameAppResp := &pb.RenameAppResponseV1{
		Base: &baseResp,
	}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_RenameAppResponseV1{
			RenameAppResponseV1: renameAppResp,
		},
	}

	return agentMsg, nil
}

// HandleCreateRegistryRequest handles the command dispatch and creates the appropriate response message
func HandleCreateRegistryRequest(commandBus cqrs.CommandBus, createRegistryRequest *pb.CreateRegistryRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing create registry request", "name", createRegistryRequest.Address)

	cmd := create_registry.CreateRegistryCommand{
		Address:  createRegistryRequest.Address,
		Username: createRegistryRequest.Username,
		Password: createRegistryRequest.Password,
	}

	responseCode := pb.ResponseCode_RESPONSE_CODE_SUCCESS
	responseMessage := "Registry created successfully"

	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error creating registry", "error", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error creating registry: %v", err)
	}

	baseResp := createBaseResponse(createRegistryRequest.Base.MessageId, agentID, responseCode, responseMessage)
	resp := &pb.CreateRegistryResponseV1{Base: &baseResp}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_CreateRegistryResponseV1{CreateRegistryResponseV1: resp},
	}

	return agentMsg, nil
}

// HandleDeleteRegistryRequest handles the command dispatch and creates the appropriate response message
func HandleDeleteRegistryRequest(commandBus cqrs.CommandBus, deleteRegistryRequest *pb.DeleteRegistryRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing delete registry request", "name", deleteRegistryRequest.Address)

	cmd := delete_registry.DeleteRegistryCommand{Address: deleteRegistryRequest.Address}

	responseCode := pb.ResponseCode_RESPONSE_CODE_SUCCESS
	responseMessage := "Registry deleted successfully"

	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error deleting registry", "error", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error deleting registry: %v", err)
	}

	baseResp := createBaseResponse(deleteRegistryRequest.Base.MessageId, agentID, responseCode, responseMessage)
	resp := &pb.DeleteRegistryResponseV1{Base: &baseResp}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_DeleteRegistryResponseV1{DeleteRegistryResponseV1: resp},
	}

	return agentMsg, nil
}

// HandleCreateNetworkRequest handles the create network command and sends back a response message
func HandleCreateNetworkRequest(commandBus cqrs.CommandBus, createNetworkRequest *pb.CreateNetworkRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing create network request", "name", createNetworkRequest.Name)

	cmd := create_network.CreateNetworkCommand{NetworkName: createNetworkRequest.Name}

	responseCode := pb.ResponseCode_RESPONSE_CODE_SUCCESS
	responseMessage := "Network created successfully"

	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error creating network", "error", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error creating network: %v", err)
	}

	baseResp := createBaseResponse(createNetworkRequest.Base.MessageId, agentID, responseCode, responseMessage)
	resp := &pb.CreateNetworkResponseV1{Base: &baseResp}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_CreateNetworkResponseV1{CreateNetworkResponseV1: resp},
	}

	return agentMsg, nil
}

// HandleDeleteNetworkRequest handles the delete network command and sends back a response message
func HandleDeleteNetworkRequest(commandBus cqrs.CommandBus, deleteNetworkRequest *pb.DeleteNetworkRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing delete network request", "name", deleteNetworkRequest.Name)

	cmd := delete_network.DeleteNetworkCommand{NetworkName: deleteNetworkRequest.Name}

	responseCode := pb.ResponseCode_RESPONSE_CODE_SUCCESS
	responseMessage := "Network deleted successfully"

	if err := commandBus.Dispatch(cmd); err != nil {
		log.Error("Error deleting network", "error", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error deleting network: %v", err)
	}

	baseResp := createBaseResponse(deleteNetworkRequest.Base.MessageId, agentID, responseCode, responseMessage)
	resp := &pb.DeleteNetworkResponseV1{Base: &baseResp}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_DeleteNetworkResponseV1{DeleteNetworkResponseV1: resp},
	}

	return agentMsg, nil
}

package client

import (
	"fmt"
	"winterflow-agent/internal/winterflow/grpc/pb"
	"winterflow-agent/internal/winterflow/handlers/get_app"
	"winterflow-agent/internal/winterflow/handlers/get_apps_status"
	"winterflow-agent/pkg/cqrs"
	log "winterflow-agent/pkg/log"
)

// HandleGetAppQuery handles the query dispatch and creates the appropriate response message
func HandleGetAppQuery(queryBus cqrs.QueryBus, getAppRequest *pb.GetAppRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing get app request for app ID: %s", getAppRequest.AppId)

	// Create the query
	query := get_app.GetAppQuery{Request: getAppRequest}

	var responseCode pb.ResponseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage string = "App retrieved successfully"
	var app *pb.AppV1

	// Dispatch the query to the handler
	result, err := queryBus.Dispatch(query)
	if err != nil {
		log.Error("Error retrieving app: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error retrieving app: %v", err)
	} else {
		// Type assertion to get the app data
		var ok bool
		app, ok = result.(*pb.AppV1)
		if !ok {
			log.Error("Error retrieving app: unexpected result type")
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = "Error retrieving app: unexpected result type"
		}
	}

	baseResp := createBaseResponse(getAppRequest.Base.MessageId, agentID, responseCode, responseMessage)
	getAppResp := &pb.GetAppResponseV1{
		Base:       &baseResp,
		App:        app,
		AppVersion: getAppRequest.AppVersion,
	}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_GetAppResponseV1{
			GetAppResponseV1: getAppResp,
		},
	}

	return agentMsg, nil
}

// HandleGetAppsStatusQuery handles the query dispatch and creates the appropriate response message
func HandleGetAppsStatusQuery(queryBus cqrs.QueryBus, getAppsStatusRequest *pb.GetAppsStatusRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing get apps status request")

	// Create the query
	query := get_apps_status.GetAppsStatusQuery{Request: getAppsStatusRequest}

	var responseCode pb.ResponseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage string = "Apps statuses retrieved successfully"
	var appStatuses []*pb.AppStatusV1

	// Dispatch the query to the handler
	result, err := queryBus.Dispatch(query)
	if err != nil {
		log.Error("Error retrieving apps statuses: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error retrieving apps statuses: %v", err)
	} else {
		// Type assertion to get the app statuses
		var ok bool
		appStatuses, ok = result.([]*pb.AppStatusV1)
		if !ok {
			log.Error("Error retrieving apps statuses: unexpected result type")
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = "Error retrieving apps statuses: unexpected result type"
		}
	}

	baseResp := createBaseResponse(getAppsStatusRequest.Base.MessageId, agentID, responseCode, responseMessage)
	getAppsStatusResp := &pb.GetAppsStatusResponseV1{
		Base: &baseResp,
		Apps: appStatuses,
	}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_GetAppsStatusResponseV1{
			GetAppsStatusResponseV1: getAppsStatusResp,
		},
	}

	return agentMsg, nil
}

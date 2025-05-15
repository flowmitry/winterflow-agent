package client

import (
	"fmt"
	"winterflow-agent/internal/winterflow/grpc/pb"
	"winterflow-agent/internal/winterflow/handlers/get_app"
	"winterflow-agent/pkg/cqrs"
	log "winterflow-agent/pkg/log"
)

// HandleGetAppQuery handles the query dispatch and creates the appropriate response message
func HandleGetAppQuery(queryBus cqrs.QueryBus, getAppRequest *pb.GetAppRequestV1, serverID string) (*pb.AgentMessage, error) {
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

	baseResp := createBaseResponse(getAppRequest.Base.MessageId, serverID, responseCode, responseMessage)
	getAppResp := &pb.GetAppResponseV1{
		Base: &baseResp,
		App:  app,
	}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_GetAppResponseV1{
			GetAppResponseV1: getAppResp,
		},
	}

	return agentMsg, nil
}

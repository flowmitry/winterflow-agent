package client

import (
	"fmt"
	"winterflow-agent/internal/application/query/get_app"
	"winterflow-agent/internal/application/query/get_apps_status"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/infra/winterflow/grpc/pb"
	"winterflow-agent/pkg/cqrs"
	"winterflow-agent/pkg/log"
)

// HandleGetAppQuery handles the query dispatch and creates the appropriate response message
func HandleGetAppQuery(queryBus cqrs.QueryBus, getAppRequest *pb.GetAppRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing get app request for app ID: %s", getAppRequest.AppId)

	// Create the query with properties directly
	query := get_app.GetAppQuery{
		AppID:      getAppRequest.AppId,
		AppVersion: getAppRequest.AppVersion,
	}

	var responseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage = "App retrieved successfully"
	var app *pb.AppV1
	var versions []uint32
	var version = getAppRequest.AppVersion

	// Dispatch the query to the handler
	result, err := queryBus.Dispatch(query)
	if err != nil {
		log.Error("Error retrieving app: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error retrieving app: %v", err)
	} else {
		// Type assertion to get the app data along with versions
		appDetails, ok := result.(*model.AppDetails)
		if !ok {
			log.Error("Error retrieving app: unexpected result type")
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = "Error retrieving app: unexpected result type"
		} else {
			// Convert domain model to protobuf
			if appDetails.App != nil {
				app = AppToProtoAppV1(appDetails.App)
			}
			version = appDetails.Version
			versions = appDetails.Versions
		}
	}

	baseResp := createBaseResponse(getAppRequest.Base.MessageId, agentID, responseCode, responseMessage)
	getAppResp := &pb.GetAppResponseV1{
		Base:              &baseResp,
		App:               app,
		AppVersion:        version,
		AvailableVersions: versions,
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

	// Create the query (no properties needed)
	query := get_apps_status.GetAppsStatusQuery{}

	var responseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage = "Apps statuses retrieved successfully"
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
		domainResult, ok := result.(*model.GetAppsStatusResult)
		if !ok {
			log.Error("Error retrieving apps statuses: unexpected result type")
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = "Error retrieving apps statuses: unexpected result type"
		} else {
			appStatuses = ContainerAppsToProtoAppStatusesV1(domainResult.Apps)
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

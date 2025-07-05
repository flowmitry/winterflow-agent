package client

import (
	"fmt"
	"winterflow-agent/internal/application/query/get_app"
	"winterflow-agent/internal/application/query/get_app_logs"
	"winterflow-agent/internal/application/query/get_apps_status"
	"winterflow-agent/internal/application/query/get_networks"
	"winterflow-agent/internal/application/query/get_registries"
	"winterflow-agent/internal/domain/dto"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/infra/winterflow/grpc/pb"
	"winterflow-agent/pkg/cqrs"
	"winterflow-agent/pkg/log"
)

// HandleGetAppQuery handles the query dispatch and creates the appropriate response message
func HandleGetAppQuery(queryBus cqrs.QueryBus, getAppRequest *pb.GetAppRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing get app request", "app_id", getAppRequest.AppId)

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
		log.Error("Error retrieving app", "error", err)
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
		log.Error("Error retrieving apps statuses", "error", err)
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

// HandleGetRegistriesQuery handles the query dispatch and creates the appropriate response message
func HandleGetRegistriesQuery(queryBus cqrs.QueryBus, getRegistriesRequest *pb.GetRegistriesRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing get registries request")

	query := get_registries.GetRegistriesQuery{}

	responseCode := pb.ResponseCode_RESPONSE_CODE_SUCCESS
	responseMessage := "Registries retrieved successfully"
	var registryAddresses []string

	result, err := queryBus.Dispatch(query)
	if err != nil {
		log.Error("Error retrieving registries", "error", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error retrieving registries: %v", err)
	} else {
		domainResult, ok := result.(*dto.GetRegistriesResult)
		if !ok {
			log.Error("Error retrieving registries: unexpected result type")
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = "Error retrieving registries: unexpected result type"
		} else {
			registryAddresses = RegistriesToProtoNames(domainResult.Registries)
		}
	}

	baseResp := createBaseResponse(getRegistriesRequest.Base.MessageId, agentID, responseCode, responseMessage)
	resp := &pb.GetRegistriesResponseV1{
		Base:    &baseResp,
		Address: registryAddresses,
	}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_GetRegistriesResponseV1{GetRegistriesResponseV1: resp},
	}

	return agentMsg, nil
}

// HandleGetNetworksQuery handles the query dispatch and creates the appropriate response message
func HandleGetNetworksQuery(queryBus cqrs.QueryBus, getNetworksRequest *pb.GetNetworksRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing get networks request")

	query := get_networks.GetNetworksQuery{}

	responseCode := pb.ResponseCode_RESPONSE_CODE_SUCCESS
	responseMessage := "Networks retrieved successfully"
	var networkNames []string

	result, err := queryBus.Dispatch(query)
	if err != nil {
		log.Error("Error retrieving networks", "error", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error retrieving networks: %v", err)
	} else {
		domainResult, ok := result.(*dto.GetNetworksResult)
		if !ok {
			log.Error("Error retrieving networks: unexpected result type")
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = "Error retrieving networks: unexpected result type"
		} else {
			networkNames = NetworksToProtoNames(domainResult.Networks)
		}
	}

	baseResp := createBaseResponse(getNetworksRequest.Base.MessageId, agentID, responseCode, responseMessage)
	resp := &pb.GetNetworksResponseV1{
		Base: &baseResp,
		Name: networkNames,
	}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_GetNetworksResponseV1{GetNetworksResponseV1: resp},
	}

	return agentMsg, nil
}

// HandleGetAppLogsQuery handles the query dispatch and creates the appropriate response message
func HandleGetAppLogsQuery(queryBus cqrs.QueryBus, getAppLogsRequest *pb.GetAppLogsRequestV1, agentID string) (*pb.AgentMessage, error) {
	log.Debug("Processing get app logs request", "app_id", getAppLogsRequest.AppId)

	sinceUnix := int64(0)
	untilUnix := int64(0)
	if getAppLogsRequest.Since != nil {
		sinceUnix = getAppLogsRequest.Since.AsTime().Unix()
	}
	if getAppLogsRequest.Until != nil {
		untilUnix = getAppLogsRequest.Until.AsTime().Unix()
	}

	// Build query
	query := get_app_logs.GetAppLogsQuery{
		AppID: getAppLogsRequest.AppId,
		Since: sinceUnix,
		Until: untilUnix,
		Tail:  getAppLogsRequest.Tail,
	}

	responseCode := pb.ResponseCode_RESPONSE_CODE_SUCCESS
	responseMessage := "Logs retrieved successfully"
	var appLogs *pb.AppLogsV1

	result, err := queryBus.Dispatch(query)
	if err != nil {
		log.Error("Error retrieving app logs", "error", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error retrieving app logs: %v", err)
	} else {
		domainLogs, ok := result.(*model.Logs)
		if !ok {
			log.Error("Error retrieving app logs: unexpected result type")
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = "Error retrieving app logs: unexpected result type"
		} else {
			appLogs = LogsToProtoAppLogsV1(domainLogs)
		}
	}

	baseResp := createBaseResponse(getAppLogsRequest.Base.MessageId, agentID, responseCode, responseMessage)
	resp := &pb.GetAppLogsResponseV1{
		Base: &baseResp,
		Logs: appLogs,
	}

	agentMsg := &pb.AgentMessage{
		Message: &pb.AgentMessage_GetAppLogsResponseV1{GetAppLogsResponseV1: resp},
	}

	return agentMsg, nil
}

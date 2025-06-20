package client

import (
	"encoding/json"
	"winterflow-agent/internal/application/command/control_app"
	"winterflow-agent/internal/application/command/delete_app"
	"winterflow-agent/internal/application/command/save_app"
	"winterflow-agent/internal/application/command/update_agent"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/infra/winterflow/grpc/pb"
	log "winterflow-agent/pkg/log"
)

// Domain to Infrastructure transformations

// AppToProtoAppV1 converts a domain app model to a protobuf app model
func AppToProtoAppV1(app *model.App) *pb.AppV1 {
	if app == nil {
		return nil
	}

	// Convert variables
	var variables []*pb.AppVarV1
	for id, content := range app.Variables {
		variables = append(variables, &pb.AppVarV1{
			Id:      id,
			Content: []byte(content),
		})
	}

	// Convert files
	var files []*pb.AppFileV1
	for id, content := range app.Files {
		files = append(files, &pb.AppFileV1{
			Id:      id,
			Content: []byte(content),
		})
	}

	// Convert AppConfig to JSON bytes
	configBytes, err := json.Marshal(app.Config)
	if err != nil {
		log.Error("Error marshaling app config: %v", err)
		configBytes = []byte{}
	}

	return &pb.AppV1{
		AppId:     app.ID,
		Config:    configBytes,
		Variables: variables,
		Files:     files,
	}
}

// ContainerAppsToProtoAppStatusesV1 converts domain container apps to protobuf app statuses
func ContainerAppsToProtoAppStatusesV1(apps []*model.ContainerApp) []*pb.AppStatusV1 {
	var appStatuses []*pb.AppStatusV1

	for _, app := range apps {
		if app == nil {
			continue
		}

		appStatus := &pb.AppStatusV1{
			AppId:      app.ID,
			Containers: ContainersToProtoContainerStatusesV1(app.Containers),
		}

		appStatuses = append(appStatuses, appStatus)
	}

	return appStatuses
}

// ContainersToProtoContainerStatusesV1 converts domain containers to protobuf container statuses
func ContainersToProtoContainerStatusesV1(containers []model.Container) []*pb.ContainerStatusV1 {
	var result []*pb.ContainerStatusV1

	for _, container := range containers {
		containerStatus := &pb.ContainerStatusV1{
			ContainerId: container.ID,
			Name:        container.Name,
			StatusCode:  ContainerStatusCodeToProtoContainerStatusCode(container.StatusCode),
			ExitCode:    int32(container.ExitCode),
			Error:       container.Error,
		}
		result = append(result, containerStatus)
	}

	return result
}

// ContainerStatusCodeToProtoContainerStatusCode converts a domain container status code to a protobuf container status code
func ContainerStatusCodeToProtoContainerStatusCode(statusCode model.ContainerStatusCode) pb.ContainerStatusCode {
	switch statusCode {
	case model.ContainerStatusActive:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_ACTIVE
	case model.ContainerStatusIdle:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_IDLE
	case model.ContainerStatusRestarting:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_RESTARTING
	case model.ContainerStatusProblematic:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_PROBLEMATIC
	case model.ContainerStatusStopped:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_STOPPED
	default:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_UNKNOWN
	}
}

// Infrastructure to Domain transformations

// ProtoAppVarsV1ToVariableMap converts protobuf app variables to a domain variable map
func ProtoAppVarsV1ToVariableMap(vars []*pb.AppVarV1) model.VariableMap {
	variableMap := make(model.VariableMap)
	for _, v := range vars {
		variableMap[v.Id] = string(v.Content)
	}
	return variableMap
}

// ProtoAppFilesV1ToFilesMap converts protobuf app files to a domain files map
func ProtoAppFilesV1ToFilesMap(files []*pb.AppFileV1) model.FilesMap {
	filesMap := make(model.FilesMap)
	for _, v := range files {
		filesMap[v.Id] = string(v.Content)
	}
	return filesMap
}

// ProtoAppV1ToApp converts a protobuf app model to a domain app model
func ProtoAppV1ToApp(app *pb.AppV1) *model.App {
	if app == nil {
		return nil
	}

	// Convert variables
	variables := ProtoAppVarsV1ToVariableMap(app.Variables)

	// Convert files
	files := make(map[string]string)
	for _, file := range app.Files {
		files[file.Id] = string(file.Content)
	}

	// Parse config bytes into AppConfig
	appConfig, err := model.ParseAppConfig(app.Config)
	if err != nil {
		log.Error("Error parsing app config: %v", err)
		appConfig = &model.AppConfig{ID: app.AppId}
	}

	return &model.App{
		ID:        app.AppId,
		Config:    appConfig,
		Variables: variables,
		Files:     files,
	}
}

// ProtoAppStatusesV1ToContainerApps converts protobuf app statuses to domain container apps
func ProtoAppStatusesV1ToContainerApps(appStatuses []*pb.AppStatusV1) []*model.ContainerApp {
	var apps []*model.ContainerApp

	for _, appStatus := range appStatuses {
		if appStatus == nil {
			continue
		}

		app := &model.ContainerApp{
			ID:         appStatus.AppId,
			Containers: ProtoContainerStatusesV1ToContainers(appStatus.Containers),
		}

		apps = append(apps, app)
	}

	return apps
}

// ProtoContainerStatusesV1ToContainers converts protobuf container statuses to domain containers
func ProtoContainerStatusesV1ToContainers(containerStatuses []*pb.ContainerStatusV1) []model.Container {
	var containers []model.Container

	for _, containerStatus := range containerStatuses {
		if containerStatus == nil {
			continue
		}

		container := model.Container{
			ID:         containerStatus.ContainerId,
			Name:       containerStatus.Name,
			StatusCode: ProtoContainerStatusCodeToContainerStatusCode(containerStatus.StatusCode),
			ExitCode:   int(containerStatus.ExitCode),
			Error:      containerStatus.Error,
		}

		containers = append(containers, container)
	}

	return containers
}

// ProtoContainerStatusCodeToContainerStatusCode converts a protobuf container status code to a domain container status code
func ProtoContainerStatusCodeToContainerStatusCode(statusCode pb.ContainerStatusCode) model.ContainerStatusCode {
	switch statusCode {
	case pb.ContainerStatusCode_CONTAINER_STATUS_CODE_ACTIVE:
		return model.ContainerStatusActive
	case pb.ContainerStatusCode_CONTAINER_STATUS_CODE_IDLE:
		return model.ContainerStatusIdle
	case pb.ContainerStatusCode_CONTAINER_STATUS_CODE_RESTARTING:
		return model.ContainerStatusRestarting
	case pb.ContainerStatusCode_CONTAINER_STATUS_CODE_PROBLEMATIC:
		return model.ContainerStatusProblematic
	case pb.ContainerStatusCode_CONTAINER_STATUS_CODE_STOPPED:
		return model.ContainerStatusStopped
	default:
		return model.ContainerStatusUnknown
	}
}

// Command assemblers

// ProtoSaveAppRequestV1ToSaveAppCommand converts a protobuf SaveAppRequestV1 to a domain SaveAppCommand
func ProtoSaveAppRequestV1ToSaveAppCommand(request *pb.SaveAppRequestV1) save_app.SaveAppCommand {
	if request == nil || request.App == nil {
		return save_app.SaveAppCommand{}
	}

	// Convert variables to VariableMap
	variables := ProtoAppVarsV1ToVariableMap(request.App.Variables)
	files := ProtoAppFilesV1ToFilesMap(request.App.Files)

	// Parse config bytes into AppConfig
	appConfig, err := model.ParseAppConfig(request.App.Config)
	if err != nil {
		log.Error("Error parsing app config: %v", err)
		appConfig = &model.AppConfig{ID: request.App.AppId}
	}

	return save_app.SaveAppCommand{
		App: &model.App{
			ID:        request.App.AppId,
			Config:    appConfig,
			Variables: variables,
			Files:     files,
		},
	}
}

// ProtoDeleteAppRequestV1ToDeleteAppCommand converts a protobuf DeleteAppRequestV1 to a domain DeleteAppCommand
func ProtoDeleteAppRequestV1ToDeleteAppCommand(request *pb.DeleteAppRequestV1) delete_app.DeleteAppCommand {
	if request == nil {
		return delete_app.DeleteAppCommand{}
	}

	return delete_app.DeleteAppCommand{
		AppID: request.AppId,
	}
}

// ProtoControlAppRequestV1ToControlAppCommand converts a protobuf ControlAppRequestV1 to a domain ControlAppCommand
func ProtoControlAppRequestV1ToControlAppCommand(request *pb.ControlAppRequestV1) control_app.ControlAppCommand {
	if request == nil {
		return control_app.ControlAppCommand{}
	}

	// Convert AppAction
	var action control_app.AppAction
	switch request.Action {
	case pb.AppAction_START:
		action = control_app.AppActionStart
	case pb.AppAction_STOP:
		action = control_app.AppActionStop
	case pb.AppAction_RESTART:
		action = control_app.AppActionRestart
	case pb.AppAction_UPDATE:
		action = control_app.AppActionUpdate
	default:
		action = control_app.AppActionStop
	}

	return control_app.ControlAppCommand{
		AppID:      request.AppId,
		AppVersion: request.AppVersion,
		Action:     action,
	}
}

// ProtoUpdateAgentRequestV1ToUpdateAgentCommand converts a protobuf UpdateAgentRequestV1 to a domain UpdateAgentCommand
func ProtoUpdateAgentRequestV1ToUpdateAgentCommand(request *pb.UpdateAgentRequestV1) update_agent.UpdateAgentCommand {
	if request == nil {
		return update_agent.UpdateAgentCommand{}
	}

	return update_agent.UpdateAgentCommand{
		Version: request.Version,
	}
}

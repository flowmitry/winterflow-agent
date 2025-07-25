package client

import (
	"encoding/json"
	"time"
	"winterflow-agent/internal/application/command/control_app"
	"winterflow-agent/internal/application/command/create_registry"
	"winterflow-agent/internal/application/command/delete_app"
	"winterflow-agent/internal/application/command/delete_registry"
	"winterflow-agent/internal/application/command/rename_app"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/infra/winterflow/grpc/pb"
	"winterflow-agent/pkg/log"

	"google.golang.org/protobuf/types/known/timestamppb"
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
		log.Error("Error marshaling app config", "error", err)
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
			StatusCode: ContainerStatusCodeToProtoContainerStatusCode(app.StatusCode),
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

// ProtoAppV1ToApp converts a protobuf app model to a domain app model
func ProtoAppV1ToApp(app *pb.AppV1) *model.App {
	if app == nil {
		return nil
	}

	// Convert variables
	variables := ProtoAppVarsV1ToVariableMap(app.Variables)

	// Convert files
	files := make(map[string][]byte)
	for _, file := range app.Files {
		files[file.Id] = file.Content
	}

	// Parse config bytes into AppConfig
	appConfig, err := model.ParseAppConfig(app.Config)
	if err != nil {
		log.Error("Error parsing app config", "error", err)
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
	case pb.AppAction_REDEPLOY:
		action = control_app.AppActionRedeploy
	default:
		action = control_app.AppActionStop
	}

	return control_app.ControlAppCommand{
		AppID:  request.AppId,
		Action: action,
	}
}

// ProtoRenameAppRequestV1ToRenameAppCommand converts a protobuf RenameAppRequestV1 to a domain RenameAppCommand
func ProtoRenameAppRequestV1ToRenameAppCommand(request *pb.RenameAppRequestV1) rename_app.RenameAppCommand {
	if request == nil {
		return rename_app.RenameAppCommand{}
	}
	return rename_app.RenameAppCommand{
		AppID:   request.AppId,
		AppName: request.AppName,
	}
}

// ---------------------------------------------------------------------------
// Registry helpers
// ---------------------------------------------------------------------------

// RegistriesToProtoNames converts a slice of domain Registry models to a slice of strings
// expected by GetRegistriesResponseV1.
func RegistriesToProtoNames(registries []model.Registry) []string {
	names := make([]string, 0, len(registries))
	for _, r := range registries {
		names = append(names, r.Address)
	}
	return names
}

// ProtoCreateRegistryRequestV1ToCreateRegistryCommand converts protobuf CreateRegistryRequestV1
// into a CreateRegistryCommand.
func ProtoCreateRegistryRequestV1ToCreateRegistryCommand(request *pb.CreateRegistryRequestV1) create_registry.CreateRegistryCommand {
	if request == nil {
		return create_registry.CreateRegistryCommand{}
	}
	return create_registry.CreateRegistryCommand{
		Address:  request.Address,
		Username: request.Username,
		Password: request.Password,
	}
}

// ProtoDeleteRegistryRequestV1ToDeleteRegistryCommand converts protobuf DeleteRegistryRequestV1
// into a DeleteRegistryCommand.
func ProtoDeleteRegistryRequestV1ToDeleteRegistryCommand(request *pb.DeleteRegistryRequestV1) delete_registry.DeleteRegistryCommand {
	if request == nil {
		return delete_registry.DeleteRegistryCommand{}
	}
	return delete_registry.DeleteRegistryCommand{
		Address: request.Address,
	}
}

// NetworksToProtoNames converts a slice of domain Network models to a slice of strings
// expected by GetNetworksResponseV1.
func NetworksToProtoNames(networks []model.Network) []string {
	names := make([]string, 0, len(networks))
	for _, n := range networks {
		names = append(names, n.Name)
	}
	return names
}

// LogsToProtoAppLogsV1 converts domain logs model to a protobuf AppLogsV1 message.
func LogsToProtoAppLogsV1(l *model.Logs) *pb.AppLogsV1 {
	if l == nil {
		return nil
	}

	// Map containers to a map[containerID]name
	containersMap := make(map[string]string)
	for _, c := range l.Containers {
		containersMap[c.ID] = c.Name
	}

	// Convert log entries
	var protoEntries []*pb.LogEntryV1
	for _, le := range l.Logs {
		protoEntries = append(protoEntries, LogEntryToProtoLogEntryV1(le))
	}

	return &pb.AppLogsV1{
		Containers: containersMap,
		Logs:       protoEntries,
	}
}

// LogEntryToProtoLogEntryV1 converts a domain LogEntry to protobuf LogEntryV1.
func LogEntryToProtoLogEntryV1(e model.LogEntry) *pb.LogEntryV1 {
	ts := timestamppb.New(time.Unix(e.Timestamp, 0))

	// Marshal Data map to string if present
	var dataStr string
	if e.Data != nil {
		if b, err := json.Marshal(e.Data); err == nil {
			dataStr = string(b)
		}
	}

	return &pb.LogEntryV1{
		Timestamp:   ts,
		Channel:     LogChannelToProtoLogChannel(e.Channel),
		Level:       LogLevelToProtoLogLevel(e.Level),
		Message:     e.Message,
		Data:        dataStr,
		ContainerId: e.ContainerID,
	}
}

// LogChannelToProtoLogChannel converts domain LogChannel to protobuf LogChannel.
func LogChannelToProtoLogChannel(ch model.LogChannel) pb.LogChannel {
	switch ch {
	case model.LogChannelStdout:
		return pb.LogChannel_LOG_CHANNEL_STDOUT
	case model.LogChannelStderr:
		return pb.LogChannel_LOG_CHANNEL_STDERR
	default:
		return pb.LogChannel_LOG_CHANNEL_UNKNOWN
	}
}

// LogLevelToProtoLogLevel converts domain LogLevel to protobuf LogLevel.
func LogLevelToProtoLogLevel(lvl model.LogLevel) pb.LogLevel {
	switch lvl {
	case model.LogLevelTrace:
		return pb.LogLevel_LOG_LEVEL_TRACE
	case model.LogLevelDebug:
		return pb.LogLevel_LOG_LEVEL_DEBUG
	case model.LogLevelInfo:
		return pb.LogLevel_LOG_LEVEL_INFO
	case model.LogLevelWarn:
		return pb.LogLevel_LOG_LEVEL_WARN
	case model.LogLevelError:
		return pb.LogLevel_LOG_LEVEL_ERROR
	case model.LogLevelFatal:
		return pb.LogLevel_LOG_LEVEL_FATAL
	default:
		return pb.LogLevel_LOG_LEVEL_UNKNOWN
	}
}

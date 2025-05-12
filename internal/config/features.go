package config

const (
	FeatureFileOperations   = "file_operations"
	FeatureNetworkScanning  = "network_scanning"
	FeatureProcessControl   = "process_control"
	FeatureSystemMonitoring = "system_monitoring"
	FeatureLogCollection    = "log_collection"
	FeatureRemoteExecution  = "remote_execution"
	FeatureDataCollection   = "data_collection"
	FeatureSecurityScanning = "security_scanning"
)

// DefaultFeatureValues defines the default values for each feature
var DefaultFeatureValues = map[string]bool{
	FeatureFileOperations:   true,
	FeatureNetworkScanning:  true,
	FeatureProcessControl:   true,
	FeatureSystemMonitoring: true,
	FeatureLogCollection:    true,
	FeatureRemoteExecution:  false,
	FeatureDataCollection:   true,
	FeatureSecurityScanning: false,
}

package model

type LogLevel int8

const (
	LogLevelUnknown LogLevel = 0
	LogLevelTrace   LogLevel = 1
	LogLevelDebug   LogLevel = 2
	LogLevelInfo    LogLevel = 3
	LogLevelWarn    LogLevel = 4
	LogLevelError   LogLevel = 5
	LogLevelFatal   LogLevel = 6
)

type LogChannel int8

const (
	LogChannelUnknown LogChannel = 0
	LogChannelStdout  LogChannel = 1
	LogChannelStderr  LogChannel = 2
)

type Logs struct {
	Logs       []LogEntry  `json:"logs"`
	Containers []Container `json:"containers"`
}

type LogEntry struct {
	Timestamp   int64                  `json:"timestamp"`
	Channel     LogChannel             `json:"channel"`
	Level       LogLevel               `json:"level"`
	Message     string                 `json:"message,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	ContainerID string                 `json:"container_id,omitempty"`
}

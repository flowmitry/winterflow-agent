package docker_compose

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/infra/orchestrator"
	"winterflow-agent/pkg/log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

func (r *composeRepository) GetLogs(appID string, since int64, until int64) (model.Logs, error) {
	// Prepare the result struct so we can populate it incrementally.
	res := model.Logs{
		Logs:       make([]model.LogEntry, 0),
		Containers: make([]model.Container, 0),
	}

	// Resolve the compose project (human-friendly) name of the application.
	appName, err := r.getAppNameById(appID)
	if err != nil {
		return res, fmt.Errorf("cannot get logs: %w", err)
	}

	ctx := context.Background()

	// Locate containers that belong to the compose project by label.
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", appName))

	containers, err := r.client.ContainerList(ctx, container.ListOptions{All: true, Filters: filterArgs})
	if err != nil {
		return res, fmt.Errorf("failed to list containers for app %s: %w", appID, err)
	}

	// Convert unix timestamps (in seconds) to strings understood by the Docker API.
	sinceStr := ""
	untilStr := ""
	if since > 0 {
		sinceStr = strconv.FormatInt(since, 10)
	}
	if until > 0 {
		untilStr = strconv.FormatInt(until, 10)
	}

	// Iterate over each container and fetch its logs.
	for _, c := range containers {
		containerModel := model.Container{
			ID:         c.ID,
			Name:       strings.TrimPrefix(c.Names[0], "/"),
			StatusCode: orchestrator.MapDockerStateToContainerStatus(c.State),
		}
		res.Containers = append(res.Containers, containerModel)

		// Fetch both stdout and stderr separately so that we always know the channel.
		for _, ch := range []struct {
			stdout  bool
			stderr  bool
			channel model.LogChannel
		}{
			{stdout: true, stderr: false, channel: model.LogChannelStdout},
			{stdout: false, stderr: true, channel: model.LogChannelStderr},
		} {
			logsReader, err := r.client.ContainerLogs(ctx, c.ID, container.LogsOptions{
				ShowStdout: ch.stdout,
				ShowStderr: ch.stderr,
				Timestamps: true,
				Since:      sinceStr,
				Until:      untilStr,
				Follow:     false,
				Details:    false,
			})
			if err != nil {
				log.Warn("failed to fetch container logs", "container_id", c.ID, "error", err)
				continue
			}

			scanner := bufio.NewScanner(logsReader)
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					continue
				}

				// Expected format when Timestamps=true: "<timestamp> <message>"
				ts := time.Now()
				rawMsg := line
				msg := rawMsg
				if sp := strings.SplitN(line, " ", 2); len(sp) == 2 {
					if parsed, err := time.Parse(time.RFC3339Nano, sp[0]); err == nil {
						ts = parsed
						rawMsg = sp[1]
					}
				}

				// Attempt to parse the message part as JSON. When successful we
				// populate the Data field and extract the textual message.
				var dataMap map[string]interface{}
				if err := json.Unmarshal([]byte(rawMsg), &dataMap); err == nil {
					// Extract message string if present, otherwise keep raw JSON string.
					if m, ok := dataMap["message"].(string); ok {
						msg = m
					} else if m, ok := dataMap["msg"].(string); ok {
						msg = m
					} else {
						msg = rawMsg
					}
				} else {
					dataMap = nil
					msg = rawMsg
				}

				level := detectLogLevel(msg)

				res.Logs = append(res.Logs, model.LogEntry{
					Timestamp:   ts.Unix(),
					Channel:     ch.channel,
					Level:       level,
					Message:     msg,
					Data:        dataMap,
					ContainerID: c.ID,
				})
			}
			// Intentionally ignore scanner error – in most cases incomplete logs are acceptable.
			_ = logsReader.Close()
		}
	}

	return res, nil
}

// detectLogLevel performs a best-effort detection of the log level based on
// common textual prefixes. If no known prefix is found it returns
// LogLevelUnknown.
func detectLogLevel(msg string) model.LogLevel {
	upper := strings.ToUpper(msg)
	switch {
	case strings.HasPrefix(upper, "TRACE"):
		return model.LogLevelTrace
	case strings.HasPrefix(upper, "DEBUG"):
		return model.LogLevelDebug
	case strings.HasPrefix(upper, "INFO"):
		return model.LogLevelInfo
	case strings.HasPrefix(upper, "WARN") || strings.HasPrefix(upper, "WARNING"):
		return model.LogLevelWarn
	case strings.HasPrefix(upper, "ERROR"):
		return model.LogLevelError
	case strings.HasPrefix(upper, "FATAL"):
		return model.LogLevelFatal
	default:
		return model.LogLevelUnknown
	}
}

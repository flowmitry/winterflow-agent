package metrics

import (
	"strconv"
	"time"
)

// AgentUptimeMetric reports the process uptime since the agent was started.
// The value is expressed in seconds to avoid issues with different duration
// string formats across platforms.
type AgentUptimeMetric struct {
	startTime time.Time
}

// NewAgentUptimeMetric returns a new AgentUptimeMetric initialised with the agent start
// time.
func NewAgentUptimeMetric(startTime time.Time) *AgentUptimeMetric {
	return &AgentUptimeMetric{startTime: startTime}
}

// Name implements the Metric interface.
func (m *AgentUptimeMetric) Name() string { return "agent_uptime_seconds" }

// Value implements the Metric interface and returns the uptime in seconds
// encoded as a string.
func (m *AgentUptimeMetric) Value() string {
	seconds := int64(time.Since(m.startTime).Seconds())
	return strconv.FormatInt(seconds, 10)
}

package metrics

import (
	"runtime"
	"strconv"
)

// GoroutinesMetric reports the current number of goroutines.
// This can be useful to spot leaks or unusual load.
type AgentGoroutinesMetric struct{}

// NewAgentGoroutinesMetric returns a new AgentGoroutinesMetric.
func NewAgentGoroutinesMetric() *AgentGoroutinesMetric { return &AgentGoroutinesMetric{} }

// Name implements Metric interface.
func (m *AgentGoroutinesMetric) Name() string { return "agent_goroutines_count" }

// Value implements Metric interface.
func (m *AgentGoroutinesMetric) Value() string {
	n := runtime.NumGoroutine()
	return strconv.Itoa(n)
}

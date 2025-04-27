package metrics

import (
	"runtime"
	"strconv"
)

// AgentMemoryMetric reports the current heap allocation in bytes.
// The metric is intentionally kept simple, focusing on the heap allocation
// which is often the most relevant figure for an agent.
// If you need more detailed memory statistics you can extend this metric or
// add separate ones without breaking the existing API.
type AgentMemoryMetric struct{}

// NewAgentMemoryMetric returns a new AgentMemoryMetric.
func NewAgentMemoryMetric() *AgentMemoryMetric { return &AgentMemoryMetric{} }

// Name implements Metric.
func (m *AgentMemoryMetric) Name() string { return "agent_memory_heap_bytes" }

// Value implements Metric.
func (m *AgentMemoryMetric) Value() string {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	// Return the number as a decimal string to avoid JSON number precision
	// issues on the receiver side for large values.
	return strconv.FormatUint(stats.HeapAlloc, 10)
}

package metrics

import "time"

// Metric represents a single system metric that can be collected at runtime.
// Each metric should have a human-readable name and return its current value
// as a string so that the caller can decide how to serialise and transmit the
// data (e.g. gRPC, JSON, etc.). Keeping the value as a string avoids
// additional conversions on the caller side and makes it trivial to add new
// metrics that might not naturally fit into a numerical type (e.g. uptime,
// version strings).
type Metric interface {
	// Name returns the canonical metric name that will be used as the key in
	// the metrics map sent to the server.
	Name() string
	// Value returns the current metric value encoded as a string.
	Value() string
}

// MetricFactory is responsible for instantiating all the Metric
// implementations that are available on the current platform and aggregating
// their values into a single map[string]string ready to be shipped inside a
// heartbeat message.  This mirrors the design used for the capabilities
// package to keep the overall architecture consistent and easy to reason
// about.
//
// The factory is initialised with the agent start time so that metrics that
// rely on it (e.g. uptime) can be implemented without introducing global
// variables.
//
// NOTE: If you need to add a new metric just create a new file in this package
// implementing the Metric interface and register it inside NewMetricFactory.
type MetricFactory struct {
	metrics []Metric
}

// NewMetricFactory returns a factory filled with agent-specific runtime metrics.
// These metrics focus on the agent's internal state and performance.
// The list can be extended later without touching the calling code.
func NewMetricsFactory(startTime time.Time) *MetricFactory {
	return &MetricFactory{
		metrics: []Metric{
			NewAgentUptimeMetric(startTime),
			NewAgentMemoryMetric(),
			NewAgentGoroutinesMetric(),
			NewSystemLoadavgMetric(),
			NewSystemMemoryAvailableMetric(),
			NewSystemDiskAvailableMetric("/"),
		},
	}
}

// NewSystemInfoFactory returns a factory filled with system-wide metrics.
// These metrics focus on the overall system state and resources.
// The list can be extended later without touching the calling code.
//
// NOTE: The following metrics have been moved to capabilities:
// - NewSystemUptimeMetric()
// - NewSystemMemoryTotalMetric()
// - NewSystemDiskTotalMetric("/")
func NewSystemInfoFactory(startTime time.Time) *MetricFactory {
	return &MetricFactory{
		metrics: []Metric{},
	}
}

// Collect walks through all registered metrics and returns their current
// values.  The function is intentionally lightweight so that it can be called
// on every heartbeat tick without noticeable overhead.
func (f *MetricFactory) Collect() map[string]string {
	results := make(map[string]string, len(f.metrics))
	for _, m := range f.metrics {
		results[m.Name()] = m.Value()
	}
	return results
}

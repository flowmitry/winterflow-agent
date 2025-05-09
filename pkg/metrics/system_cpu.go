package metrics

import (
	"os"
	"runtime"
	"strconv"
	"strings"
)

// SystemCpuUsageMetric reports the percentage of CPU utilized since the last reading.
// Only supported on Linux; returns empty string on other platforms.
type SystemCpuUsageMetric struct {
	prevIdle    uint64
	prevTotal   uint64
	initialized bool
}

// NewSystemCpuUsageMetric returns a new SystemCpuUsageMetric.
func NewSystemCpuUsageMetric() *SystemCpuUsageMetric { return &SystemCpuUsageMetric{} }

// Name implements Metric.
func (m *SystemCpuUsageMetric) Name() string { return "system_cpu_usage_percent" }

// Value implements Metric: reads /proc/stat, parses total and idle jiffies,
// computes the delta since last call, and returns the percentage of active time.
func (m *SystemCpuUsageMetric) Value() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			var total uint64
			for _, f := range fields[1:] {
				v, err := strconv.ParseUint(f, 10, 64)
				if err != nil {
					continue
				}
				total += v
			}
			idle, err := strconv.ParseUint(fields[4], 10, 64)
			if err != nil {
				return ""
			}
			if !m.initialized {
				m.prevIdle = idle
				m.prevTotal = total
				m.initialized = true
				return ""
			}
			idleDelta := idle - m.prevIdle
			totalDelta := total - m.prevTotal
			m.prevIdle = idle
			m.prevTotal = total
			if totalDelta == 0 {
				return "0"
			}
			usage := 100 * (float64(totalDelta-idleDelta) / float64(totalDelta))
			return strconv.FormatFloat(usage, 'f', 2, 64)
		}
	}
	return ""
}

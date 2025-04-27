package metrics

import (
	"os"
	"runtime"
	"strings"
)

// SystemLoadavgMetric reports the 1-minute system load average on Unix
// platforms. On unsupported OSes the metric returns an empty string.
// This keeps the interface consistent while signalling the absence of data.
//
// NOTE: Windows currently returns an empty value because the Go standard
// library does not expose load averages on that platform.
//
// The implementation reads /proc/loadavg which is available on Linux. For
// other Unix systems you may extend the implementation using sysctl or
// getloadavg via cgo (not allowed per project constraints).
type SystemLoadavgMetric struct{}

// NewSystemLoadavgMetric returns a new SystemLoadavgMetric.
func NewSystemLoadavgMetric() *SystemLoadavgMetric { return &SystemLoadavgMetric{} }

func (m *SystemLoadavgMetric) Name() string { return "system_load_average_1m" }

func (m *SystemLoadavgMetric) Value() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return ""
	}
	return fields[0] // first field is 1-minute load average
}

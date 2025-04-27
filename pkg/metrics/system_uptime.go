package metrics

import (
	"os"
	"runtime"
	"strconv"
	"strings"
)

// SystemUptimeMetric reports the OS boot uptime in seconds.
// Only supported on Linux (reads /proc/uptime); returns empty on other OSes.
type SystemUptimeMetric struct{}

// NewSystemUptimeMetric returns a new SystemUptimeMetric.
func NewSystemUptimeMetric() *SystemUptimeMetric { return &SystemUptimeMetric{} }

// Name implements Metric.
func (m *SystemUptimeMetric) Name() string { return "system_uptime_seconds" }

// Value implements Metric: reads seconds since boot from /proc/uptime.
func (m *SystemUptimeMetric) Value() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return ""
	}
	f, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return ""
	}
	secs := int64(f)
	return strconv.FormatInt(secs, 10)
}

package capabilities

import (
	"os"
	"runtime"
	"strconv"
	"strings"
)

// SystemUptimeCapability reports the OS boot uptime in seconds.
// Only supported on Linux (reads /proc/uptime); returns empty on other OSes.
type SystemUptimeCapability struct{}

// NewSystemUptimeCapability returns a new SystemUptimeCapability.
func NewSystemUptimeCapability() *SystemUptimeCapability {
	return &SystemUptimeCapability{}
}

// Name implements Capability.
func (c *SystemUptimeCapability) Name() string {
	return CapabilitySystemUptime
}

// Version implements Capability: reads seconds since boot from /proc/uptime.
func (c *SystemUptimeCapability) Version() string {
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

// IsAvailable implements Capability.
func (c *SystemUptimeCapability) IsAvailable() bool {
	return runtime.GOOS == "linux"
}

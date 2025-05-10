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
// Returns nil if the OS is not Linux, as the implementation reads from /proc/uptime which is only available on Linux.
func NewSystemUptimeCapability() *SystemUptimeCapability {
	if runtime.GOOS != "linux" {
		return nil
	}
	return &SystemUptimeCapability{}
}

// Name implements Capability.
func (c *SystemUptimeCapability) Name() string {
	return CapabilitySystemUptime
}

// Value implements Capability: reads seconds since boot from /proc/uptime.
func (c *SystemUptimeCapability) Value() string {
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

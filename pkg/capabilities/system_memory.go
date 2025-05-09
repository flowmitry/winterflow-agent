package capabilities

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// SystemMemoryTotalCapability reports the total physical memory (kB) on Linux.
// Returns empty value on unsupported OSes.
type SystemMemoryTotalCapability struct{}

// NewSystemMemoryTotalCapability returns a new SystemMemoryTotalCapability.
func NewSystemMemoryTotalCapability() *SystemMemoryTotalCapability {
	return &SystemMemoryTotalCapability{}
}

// Name implements Capability.
func (c *SystemMemoryTotalCapability) Name() string {
	return CapabilitySystemMemoryTotal
}

// Version implements Capability: reads total memory from /proc/meminfo.
func (c *SystemMemoryTotalCapability) Version() string {
	total, _ := readMemInfo("MemTotal")
	return total
}

// IsAvailable implements Capability.
func (c *SystemMemoryTotalCapability) IsAvailable() bool {
	return runtime.GOOS == "linux"
}

// readMemInfo helper to parse /proc/meminfo
func readMemInfo(key string) (string, bool) {
	if runtime.GOOS != "linux" {
		return "", false
	}
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return "", false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, key+":") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// return number without unit
				if _, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
					return parts[1], true
				}
			}
		}
	}
	return "", false
}

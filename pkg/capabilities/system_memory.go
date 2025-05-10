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
// Returns nil if the OS is not Linux, as the implementation reads from /proc/meminfo which is only available on Linux.
func NewSystemMemoryTotalCapability() *SystemMemoryTotalCapability {
	if runtime.GOOS != "linux" {
		return nil
	}
	return &SystemMemoryTotalCapability{}
}

// Name implements Capability.
func (c *SystemMemoryTotalCapability) Name() string {
	return CapabilitySystemMemoryTotal
}

// Value implements Capability: reads total memory from /proc/meminfo.
func (c *SystemMemoryTotalCapability) Value() string {
	total, _ := readMemInfo("MemTotal")
	return total
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

	// Check for errors that occurred during scanning
	if err := scanner.Err(); err != nil {
		return "", false
	}

	return "", false
}

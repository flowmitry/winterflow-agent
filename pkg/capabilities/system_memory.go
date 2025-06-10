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

// readMemInfo helper to parse /proc/meminfo and return value in bytes
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
				// Parse the number
				value, err := strconv.ParseUint(parts[1], 10, 64)
				if err != nil {
					return "", false
				}

				// Convert to bytes based on unit (default is kB)
				unit := "kB"
				if len(parts) >= 3 {
					unit = parts[2]
				}

				var bytes uint64
				switch unit {
				case "kB":
					bytes = value * 1024
				case "MB":
					bytes = value * 1024 * 1024
				case "GB":
					bytes = value * 1024 * 1024 * 1024
				default:
					bytes = value // If no unit or unrecognized unit, assume it's already in bytes
				}

				return strconv.FormatUint(bytes, 10), true
			}
		}
	}

	// Check for errors that occurred during scanning
	if err := scanner.Err(); err != nil {
		return "", false
	}

	return "", false
}

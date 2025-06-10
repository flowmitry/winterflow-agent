package metrics

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// SystemMemoryTotalMetric reports the total physical memory (kB) on Linux.
// Returns empty value on unsupported OSes.

// SystemMemoryAvailableMetric reports the available memory (kB) on Linux.

type SystemMemoryTotalMetric struct{}

type SystemMemoryAvailableMetric struct{}

func NewSystemMemoryTotalMetric() *SystemMemoryTotalMetric { return &SystemMemoryTotalMetric{} }
func NewSystemMemoryAvailableMetric() *SystemMemoryAvailableMetric {
	return &SystemMemoryAvailableMetric{}
}

func (m *SystemMemoryTotalMetric) Name() string     { return "system_memory_total_kb" }
func (m *SystemMemoryAvailableMetric) Name() string { return "system_memory_available_kb" }

func (m *SystemMemoryTotalMetric) Value() string { total, _ := readMemInfo("MemTotal"); return total }
func (m *SystemMemoryAvailableMetric) Value() string {
	avail, _ := readMemInfo("MemAvailable")
	return avail
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

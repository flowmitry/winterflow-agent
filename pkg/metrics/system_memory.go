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
				// return number without unit
				if _, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
					return parts[1], true
				}
			}
		}
	}
	return "", false
}

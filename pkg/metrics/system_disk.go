package metrics

import (
	"runtime"
	"strconv"
	"syscall"
)

// SystemDiskTotalMetric reports total bytes in the given mount point.
// SystemDiskAvailableMetric reports available bytes.

// Implementation uses syscall.Statfs which is available in stdlib on Unix.
// On unsupported OSes (e.g., Windows) returns empty value.

type SystemDiskTotalMetric struct{ path string }

type SystemDiskAvailableMetric struct{ path string }

func NewSystemDiskTotalMetric(path string) *SystemDiskTotalMetric {
	return &SystemDiskTotalMetric{path: path}
}
func NewSystemDiskAvailableMetric(path string) *SystemDiskAvailableMetric {
	return &SystemDiskAvailableMetric{path: path}
}

func (m *SystemDiskTotalMetric) Name() string     { return "system_disk_total_bytes" }
func (m *SystemDiskAvailableMetric) Name() string { return "system_disk_available_bytes" }

func (m *SystemDiskTotalMetric) Value() string {
	total, _ := statfsBytes(m.path, func(s *syscall.Statfs_t) uint64 { return s.Blocks * uint64(s.Bsize) })
	return strconv.FormatUint(total, 10)
}

func (m *SystemDiskAvailableMetric) Value() string {
	avail, _ := statfsBytes(m.path, func(s *syscall.Statfs_t) uint64 { return s.Bavail * uint64(s.Bsize) })
	return strconv.FormatUint(avail, 10)
}

// helper to compute bytes using statfs, returns 0 on failure
func statfsBytes(path string, getter func(*syscall.Statfs_t) uint64) (uint64, bool) {
	if runtime.GOOS == "windows" {
		return 0, false
	}
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return 0, false
	}
	return getter(&fs), true
}

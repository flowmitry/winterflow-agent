package capabilities

import (
	"runtime"
	"strconv"
	"syscall"
)

// SystemDiskTotalCapability reports total bytes in the given mount point.
// Implementation uses syscall.Statfs which is available in stdlib on Unix.
// On unsupported OSes (e.g., Windows) returns empty value.
type SystemDiskTotalCapability struct {
	path string
}

// NewSystemDiskTotalCapability returns a new SystemDiskTotalCapability.
// Returns nil if the OS is Windows, as the implementation uses syscall.Statfs which is not available on Windows.
func NewSystemDiskTotalCapability(path string) *SystemDiskTotalCapability {
	if runtime.GOOS == "windows" {
		return nil
	}
	return &SystemDiskTotalCapability{path: path}
}

// Name implements Capability.
func (c *SystemDiskTotalCapability) Name() string {
	return CapabilitySystemDiskTotal
}

// Value implements Capability: returns total disk space in bytes.
func (c *SystemDiskTotalCapability) Value() string {
	total, _ := statfsBytes(c.path, func(s *syscall.Statfs_t) uint64 { return s.Blocks * uint64(s.Bsize) })
	return strconv.FormatUint(total, 10)
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

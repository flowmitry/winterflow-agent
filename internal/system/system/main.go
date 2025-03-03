package system

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

const (
	// ErrUnsupportedOS is returned when the operating system is not Linux
	ErrUnsupportedOS = "this agent only supports Linux operating systems"
)

// checkOS verifies that the current operating system is Linux
func checkOS() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf(ErrUnsupportedOS)
	}
	return nil
}

// runCommand executes a command and returns any error
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

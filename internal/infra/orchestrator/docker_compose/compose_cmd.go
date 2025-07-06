package docker_compose

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"winterflow-agent/pkg/log"
)

// composeUp performs `docker compose up -d` in the provided directory.
func (r *composeRepository) composeUp(appDir string) error {
	files, err := r.detectComposeFiles(appDir)
	if err != nil {
		return err
	}

	args := make([]string, 0)
	if fileExists(filepath.Join(appDir, ".winterflow.env")) {
		args = append(args, "--env-file", ".winterflow.env")
	}
	args = append(args, r.buildComposeFileArgs(files)...)
	args = append(args, "up", "-d")

	return r.runDockerCompose(appDir, args...)
}

func (r *composeRepository) composeDown(appDir string) error {
	files, err := r.detectComposeFiles(appDir)
	if err != nil {
		return err
	}

	args := make([]string, 0)
	if fileExists(filepath.Join(appDir, ".winterflow.env")) {
		args = append(args, "--env-file", ".winterflow.env")
	}
	args = append(args, r.buildComposeFileArgs(files)...)
	args = append(args, "down", "--remove-orphans")

	return r.runDockerCompose(appDir, args...)
}

func (r *composeRepository) composeRestart(appDir string) error {
	files, err := r.detectComposeFiles(appDir)
	if err != nil {
		return err
	}

	args := make([]string, 0)
	if fileExists(filepath.Join(appDir, ".winterflow.env")) {
		args = append(args, "--env-file", ".winterflow.env")
	}
	args = append(args, r.buildComposeFileArgs(files)...)
	args = append(args, "restart")

	return r.runDockerCompose(appDir, args...)
}

func (r *composeRepository) composePull(appDir string) error {
	files, err := r.detectComposeFiles(appDir)
	if err != nil {
		return err
	}
	args := append(r.buildComposeFileArgs(files), "pull")
	return r.runDockerCompose(appDir, args...)
}

// detectComposeFiles mimics the original playbook logic to decide which compose files to use.
func (r *composeRepository) detectComposeFiles(appDir string) ([]string, error) {
	custom := filepath.Join(appDir, "compose.custom.yml")
	override := filepath.Join(appDir, "compose.override.yml")
	compose := filepath.Join(appDir, "compose.yml")

	customExists := fileExists(custom)
	overrideExists := fileExists(override)
	composeExists := fileExists(compose)

	switch {
	case customExists && overrideExists:
		return []string{custom, override}, nil
	case customExists:
		return []string{custom}, nil
	case composeExists && overrideExists:
		return []string{compose, override}, nil
	case composeExists:
		// With only compose.yml docker automatically detects it – return nil slice.
		return nil, nil
	default:
		return nil, fmt.Errorf("no compose file found in %s", appDir)
	}
}

// buildComposeFileArgs converts file list into `-f file` CLI arguments.
func (r *composeRepository) buildComposeFileArgs(files []string) []string {
	if len(files) == 0 {
		return nil
	}
	var args []string
	for _, f := range files {
		args = append(args, "-f", filepath.Base(f))
	}
	return args
}

// runDockerCompose executes `docker compose` with given args in dir.
func (r *composeRepository) runDockerCompose(dir string, args ...string) error {
	fullCmd := append([]string{"compose"}, args...)
	cmd := exec.Command("docker", fullCmd...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("docker compose command failed", "dir", dir, "args", fullCmd, "output", string(output), "error", err)
		return fmt.Errorf("docker compose %v failed: %w", args, err)
	}
	log.Debug("docker compose executed", "dir", dir, "args", fullCmd, "output", string(output))
	return nil
}

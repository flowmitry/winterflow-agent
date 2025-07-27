package docker_compose

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"winterflow-agent/pkg/log"
)

// composeExtensionFiles holds all allowed compose file extensions such as
// "compose.<extension>.yml". The slice can be extended as new compose
// variants are introduced. The order is preserved when constructing the
// final -f arguments.
var composeExtensionFiles = []string{
	"expose", // compose.expose.yml
}

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
	// Base compose files recognised by Docker by default
	dockerCompose := filepath.Join(appDir, "docker-compose.yml")
	compose := filepath.Join(appDir, "compose.yml")

	dockerExists := fileExists(dockerCompose)
	composeExists := fileExists(compose)

	if !dockerExists && !composeExists {
		return nil, fmt.Errorf("neither docker-compose.yml nor compose.yml found in %s", appDir)
	}

	// Collect extension compose files if present
	var extraFiles []string
	for _, ext := range composeExtensionFiles {
		candidate := filepath.Join(appDir, fmt.Sprintf("compose.%s.yml", ext))
		if fileExists(candidate) {
			extraFiles = append(extraFiles, candidate)
		}
	}

	// Optional override file should be last so it can supersede previous ones
	override := filepath.Join(appDir, "compose.override.yml")
	if fileExists(override) {
		extraFiles = append(extraFiles, override)
	}

	// If no additional files are found, rely on Docker's implicit file detection.
	if len(extraFiles) == 0 {
		return nil, nil
	}

	// When additional files are present we must explicitly specify the base file first.
	var files []string
	if dockerExists {
		files = append(files, dockerCompose)
	} else {
		files = append(files, compose)
	}
	files = append(files, extraFiles...)

	return files, nil
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

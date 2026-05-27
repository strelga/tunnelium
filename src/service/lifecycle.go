package service

import (
	"fmt"
	"os"
	"os/exec"

	"tunnelium/src/paths"
)

// Start runs `docker compose up -d` for the given service name.
// The service name must match a key in docker-compose.yaml (e.g. "gost-incoming").
func Start(serviceName string) error {
	if err := validateServiceExists(serviceName); err != nil {
		return err
	}

	composeFile := paths.ComposeFile()
	cmd := exec.Command("docker", "compose", "-f", composeFile, "up", "-d", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("starting service %q: %w", serviceName, err)
	}

	return nil
}

// Stop runs `docker compose stop` for the given service name.
// The service name must match a key in docker-compose.yaml (e.g. "gost-incoming").
func Stop(serviceName string) error {
	if err := validateServiceExists(serviceName); err != nil {
		return err
	}

	composeFile := paths.ComposeFile()
	cmd := exec.Command("docker", "compose", "-f", composeFile, "stop", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("stopping service %q: %w", serviceName, err)
	}

	return nil
}

// Restart runs `docker compose restart` for the given service name.
func Restart(serviceName string) error {
	if err := validateServiceExists(serviceName); err != nil {
		return err
	}

	composeFile := paths.ComposeFile()
	cmd := exec.Command("docker", "compose", "-f", composeFile, "restart", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restarting service %q: %w", serviceName, err)
	}

	return nil
}

// validateServiceExists checks that the service name exists in docker-compose.yaml.
func validateServiceExists(serviceName string) error {
	services, err := GetExistingServices()
	if err != nil {
		return fmt.Errorf("reading existing services: %w", err)
	}

	for _, s := range services {
		if s == serviceName {
			return nil
		}
	}

	return fmt.Errorf("service %q not found in docker-compose.yaml", serviceName)
}

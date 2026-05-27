package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tunnelium/src/paths"
)

// ServiceInfo holds information about a service for listing.
type ServiceInfo struct {
	Name      string // e.g. "gost-incoming"
	Type      string // e.g. "gost"
	Instance  string // e.g. "incoming"
	ConfigDir string // path to config directory
}

// ListServices returns all services defined in docker-compose.yaml.
func ListServices() ([]ServiceInfo, error) {
	services, err := GetExistingServices()
	if err != nil {
		return nil, fmt.Errorf("reading existing services: %w", err)
	}

	var result []ServiceInfo
	for _, name := range services {
		info := ServiceInfo{
			Name:      name,
			ConfigDir: paths.ServiceDir(name),
		}
		// Parse type and instance from name (e.g. "gost-incoming" → "gost", "incoming")
		if idx := strings.Index(name, "-"); idx >= 0 {
			info.Type = name[:idx]
			info.Instance = name[idx+1:]
		}
		result = append(result, info)
	}

	return result, nil
}

// ServiceNames returns the names of all services (for shell completion).
func ServiceNames() ([]string, error) {
	return GetExistingServices()
}

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

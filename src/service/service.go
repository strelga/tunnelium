package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"tunnelium/src/paths"
)

// ServiceType defines the type of a service.
type ServiceType string

const (
	ServiceTypeGost ServiceType = "gost"
)

// AllServiceTypes returns all supported service types.
func AllServiceTypes() []ServiceType {
	return []ServiceType{ServiceTypeGost}
}

// GostRole defines the role of a gost service.
type GostRole string

const (
	GostRoleClient GostRole = "client"
	GostRoleServer GostRole = "server"
)

// ServiceParams holds all parameters needed to add a new service.
type ServiceParams struct {
	ServiceType    ServiceType
	InstanceName   string
	HostSystemPort int

	// Gost-specific
	GostRole        GostRole // "client" or "server"
	GostNextHopHost string   // for client: next hop host
	GostNextHopPort int      // for client: next hop port (default 443)
	GostSocksPort   int      // for client: SOCKS5+auth port on host (0 = disabled)
	GostHTTPPort    int      // for client: HTTP proxy port on host (0 = disabled)
	GostTLSCertPath string   // for server: path to existing combined PEM file (cert+key), empty = auto-generate
}

// validInstanceName matches alphanumeric characters and hyphens.
var validInstanceName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)

// Add orchestrates the full service add flow: validate, generate config, update compose.
func Add(params ServiceParams) error {
	// Validate
	if err := ValidateServiceType(params.ServiceType); err != nil {
		return err
	}
	if err := ValidateInstanceName(params.InstanceName); err != nil {
		return err
	}

	serviceName := fmt.Sprintf("%s-%s", params.ServiceType, params.InstanceName)

	// Check service doesn't already exist
	services, err := GetExistingServices()
	if err != nil {
		return fmt.Errorf("reading existing services: %w", err)
	}
	for _, s := range services {
		if s == serviceName {
			return fmt.Errorf("service %q already exists in docker-compose.yaml", serviceName)
		}
	}

	// Check directory doesn't exist
	configDir := paths.ServiceDir(serviceName)
	if _, err := os.Stat(configDir); err == nil {
		return fmt.Errorf("directory %s already exists", configDir)
	}

	// Type-specific validation
	switch params.ServiceType {
	case ServiceTypeGost:
		if err := validateGostParams(params); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported service type: %s", params.ServiceType)
	}

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", configDir, err)
	}

	// Generate and write config
	if err := generateAndWriteConfig(params); err != nil {
		return fmt.Errorf("generating config: %w", err)
	}

	// Add service to docker-compose.yaml
	if err := AddServiceToCompose(params); err != nil {
		return fmt.Errorf("adding service to compose: %w", err)
	}

	return nil
}

// ValidateServiceType checks that the service type is one of the known types.
func ValidateServiceType(serviceType ServiceType) error {
	for _, t := range AllServiceTypes() {
		if t == serviceType {
			return nil
		}
	}
	return fmt.Errorf("unsupported service type: %q", serviceType)
}

// ValidateInstanceName checks that the instance name contains only valid characters.
func ValidateInstanceName(name string) error {
	if name == "" {
		return fmt.Errorf("instance name cannot be empty")
	}
	if !validInstanceName.MatchString(name) {
		return fmt.Errorf("instance name %q is invalid: must start with alphanumeric, contain only alphanumeric and hyphens", name)
	}
	return nil
}

// ValidatePort checks that the port is in valid range and not already used in docker-compose.
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}

	usedPorts, err := GetUsedPorts()
	if err != nil {
		return fmt.Errorf("reading used ports: %w", err)
	}

	for _, p := range usedPorts {
		if p == port {
			return fmt.Errorf("port %d is already in use by another service", port)
		}
	}

	return nil
}

// validateGostParams validates gost-specific parameters.
func validateGostParams(params ServiceParams) error {
	if params.GostRole == "" {
		return fmt.Errorf("role is required for gost service (client or server)")
	}
	if params.GostRole != GostRoleClient && params.GostRole != GostRoleServer {
		return fmt.Errorf("role must be %q or %q, got %q", GostRoleClient, GostRoleServer, params.GostRole)
	}

	switch params.GostRole {
	case GostRoleClient:
		if params.GostSocksPort == 0 && params.GostHTTPPort == 0 {
			return fmt.Errorf("at least one entry point is required for gost client (socks_port or http_port)")
		}
		if params.GostNextHopHost == "" {
			return fmt.Errorf("next_hop_host is required for gost client")
		}
		if params.GostNextHopPort < 1 || params.GostNextHopPort > 65535 {
			return fmt.Errorf("next_hop_port must be between 1 and 65535, got %d", params.GostNextHopPort)
		}
		if params.GostSocksPort < 0 || params.GostSocksPort > 65535 {
			return fmt.Errorf("socks_port must be between 0 and 65535, got %d", params.GostSocksPort)
		}
		if params.GostHTTPPort < 0 || params.GostHTTPPort > 65535 {
			return fmt.Errorf("http_port must be between 0 and 65535, got %d", params.GostHTTPPort)
		}
	case GostRoleServer:
		if err := ValidatePort(params.HostSystemPort); err != nil {
			return err
		}
	}
	return nil
}

// generateAndWriteConfig generates the service config and writes it to the config directory.
func generateAndWriteConfig(params ServiceParams) error {
	serviceDir := paths.ServiceDir(fmt.Sprintf("%s-%s", params.ServiceType, params.InstanceName))

	switch params.ServiceType {
	case ServiceTypeGost:
		if params.GostRole == GostRoleServer {
			tlsPath := filepath.Join(serviceDir, "tls.pem")
			if params.GostTLSCertPath != "" {
				// Copy user-provided TLS PEM file (cert + key)
				data, err := os.ReadFile(params.GostTLSCertPath)
				if err != nil {
					return fmt.Errorf("reading TLS cert file %s: %w", params.GostTLSCertPath, err)
				}
				if err := os.WriteFile(tlsPath, data, 0600); err != nil {
					return fmt.Errorf("writing TLS PEM: %w", err)
				}
			} else {
				// Auto-generate self-signed TLS PEM (cert + key)
				if err := GenerateTLSCert(tlsPath); err != nil {
					return fmt.Errorf("generating TLS certificate: %w", err)
				}
			}
		}
		// Write empty auths.yaml if SOCKS port is configured
		if params.GostSocksPort > 0 {
			authsPath := filepath.Join(serviceDir, "auths.yaml")
			if err := os.WriteFile(authsPath, []byte{}, 0644); err != nil {
				return fmt.Errorf("writing %s: %w", authsPath, err)
			}
		}
	default:
		return fmt.Errorf("config generation not implemented for service type: %s", params.ServiceType)
	}

	return nil
}

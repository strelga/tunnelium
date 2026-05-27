package service

import (
	"fmt"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
)

// RunInteractive collects all service parameters interactively via survey prompts.
func RunInteractive() (*ServiceParams, error) {
	params := &ServiceParams{}

	// Step 1: Service type
	serviceType, err := promptServiceType()
	if err != nil {
		return nil, err
	}
	params.ServiceType = serviceType

	// Step 2: Instance name
	instanceName, err := promptInstanceName()
	if err != nil {
		return nil, err
	}
	params.InstanceName = instanceName

	// Step 3+: Type-specific parameters
	switch params.ServiceType {
	case ServiceTypeGost:
		// Role
		role, err := promptGostRole()
		if err != nil {
			return nil, err
		}
		params.GostRole = role

		if role == GostRoleClient {
			// Entry points
			socksPort, httpPort, err := promptGostEntryPoints()
			if err != nil {
				return nil, err
			}
			params.GostSocksPort = socksPort
			params.GostHTTPPort = httpPort

			// Next hop host
			nextHopHost, err := promptNextHopHost()
			if err != nil {
				return nil, err
			}
			params.GostNextHopHost = nextHopHost

			// Next hop port (default 443)
			nextHopPort, err := promptNextHopPort()
			if err != nil {
				return nil, err
			}
			params.GostNextHopPort = nextHopPort
		} else {
			// Server: host port
			hostPort, err := promptHostPort()
			if err != nil {
				return nil, err
			}
			params.HostSystemPort = hostPort

			// TLS cert path (optional)
			certPath, err := promptGostTLSCertPath()
			if err != nil {
				return nil, err
			}
			params.GostTLSCertPath = certPath
		}
	}

	return params, nil
}

// promptServiceType shows a Select prompt with available service types.
func promptServiceType() (ServiceType, error) {
	types := AllServiceTypes()
	options := make([]string, len(types))
	for i, t := range types {
		options[i] = string(t)
	}

	var selected string
	prompt := &survey.Select{
		Message: "Service type:",
		Options: options,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		return "", fmt.Errorf("selecting service type: %w", err)
	}

	return ServiceType(selected), nil
}

// promptInstanceName shows an Input prompt for the instance name.
func promptInstanceName() (string, error) {
	var name string
	prompt := &survey.Input{
		Message: "Instance name:",
		Help:    "Letters, digits, and hyphens. Example: myproxy, eu-west",
	}
	if err := survey.AskOne(prompt, &name, survey.WithValidator(func(val interface{}) error {
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}
		return ValidateInstanceName(s)
	})); err != nil {
		return "", fmt.Errorf("entering instance name: %w", err)
	}

	return name, nil
}

// promptHostPort shows an Input prompt for the host system port.
func promptHostPort() (int, error) {
	var portStr string
	prompt := &survey.Input{
		Message: "Host port:",
		Help:    "Port 1-65535, must not be used by another service",
	}
	if err := survey.AskOne(prompt, &portStr, survey.WithValidator(func(val interface{}) error {
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}
		port, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("must be a number")
		}
		return ValidatePort(port)
	})); err != nil {
		return 0, fmt.Errorf("entering host port: %w", err)
	}

	port, _ := strconv.Atoi(portStr)
	return port, nil
}

// promptGostRole shows a Select prompt for the gost role.
func promptGostRole() (GostRole, error) {
	options := []string{
		"client — entry points with relay+tls forwarding",
		"server — listens on relay+tls external port, accepts incoming traffic",
	}

	var selected string
	prompt := &survey.Select{
		Message: "Gost role:",
		Options: options,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		return "", fmt.Errorf("selecting gost role: %w", err)
	}

	if selected == options[0] {
		return GostRoleClient, nil
	}
	return GostRoleServer, nil
}

// promptGostEntryPoints asks which entry points to enable and their ports.
// Returns socksPort and httpPort (0 if disabled).
func promptGostEntryPoints() (socksPort int, httpPort int, err error) {
	// Ask which entry points to enable
	var entryPoints []string
	prompt := &survey.MultiSelect{
		Message: "Entry points (select one or both):",
		Options: []string{
			"SOCKS5 with auth",
			"HTTP proxy",
		},
	}
	if err := survey.AskOne(prompt, &entryPoints); err != nil {
		return 0, 0, fmt.Errorf("selecting entry points: %w", err)
	}

	if len(entryPoints) == 0 {
		return 0, 0, fmt.Errorf("at least one entry point must be selected")
	}

	for _, ep := range entryPoints {
		switch ep {
		case "SOCKS5 with auth":
			socksPort, err = promptEntryPointPort("SOCKS5")
			if err != nil {
				return 0, 0, err
			}
		case "HTTP proxy":
			httpPort, err = promptEntryPointPort("HTTP")
			if err != nil {
				return 0, 0, err
			}
		}
	}

	return socksPort, httpPort, nil
}

// promptEntryPointPort shows an Input prompt for an entry point port.
func promptEntryPointPort(name string) (int, error) {
	var portStr string
	prompt := &survey.Input{
		Message: fmt.Sprintf("%s port on host:", name),
		Help:    "Port 1-65535, must not be used by another service",
	}
	if err := survey.AskOne(prompt, &portStr, survey.WithValidator(func(val interface{}) error {
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}
		port, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("must be a number")
		}
		return ValidatePort(port)
	})); err != nil {
		return 0, fmt.Errorf("entering %s port: %w", name, err)
	}

	port, _ := strconv.Atoi(portStr)
	return port, nil
}

// promptNextHopHost shows an Input prompt for the gost client next hop host.
func promptNextHopHost() (string, error) {
	var host string
	prompt := &survey.Input{
		Message: "Next hop host:",
		Help:    "IP address or hostname of the server running gost-server",
	}
	if err := survey.AskOne(prompt, &host, survey.WithValidator(func(val interface{}) error {
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}
		if s == "" {
			return fmt.Errorf("host cannot be empty")
		}
		return nil
	})); err != nil {
		return "", fmt.Errorf("entering next hop host: %w", err)
	}

	return host, nil
}

// promptNextHopPort shows an Input prompt for the gost client next hop port (default 443).
func promptNextHopPort() (int, error) {
	var portStr string
	prompt := &survey.Input{
		Message: "Next hop port:",
		Default: "443",
		Help:    "relay+tls port on the server, default 443",
	}
	if err := survey.AskOne(prompt, &portStr, survey.WithValidator(func(val interface{}) error {
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}
		port, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("must be a number")
		}
		if port < 1 || port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535")
		}
		return nil
	})); err != nil {
		return 0, fmt.Errorf("entering next hop port: %w", err)
	}

	port, _ := strconv.Atoi(portStr)
	return port, nil
}

// promptGostTLSCertPath shows an Input prompt for an existing TLS cert+key PEM file (optional).
func promptGostTLSCertPath() (string, error) {
	var path string
	prompt := &survey.Input{
		Message: "TLS certificate path (combined PEM, leave empty to auto-generate):",
		Help:    "File with certificate and private key. If empty, a self-signed cert will be generated",
	}
	if err := survey.AskOne(prompt, &path); err != nil {
		return "", fmt.Errorf("entering TLS cert path: %w", err)
	}

	return path, nil
}

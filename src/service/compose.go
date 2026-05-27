package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"tunnelium/src/paths"

	goccy "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/token"
)

// GetUsedPorts reads docker-compose.yaml and returns all host ports used in services.
func GetUsedPorts() ([]int, error) {
	file, err := parseComposeFile()
	if err != nil {
		return nil, err
	}

	var ports []int
	for _, doc := range file.Docs {
		body, ok := doc.Body.(*ast.MappingNode)
		if !ok {
			continue
		}
		services := findMappingValue(body, "services")
		if services == nil {
			continue
		}
		servicesMap, ok := services.(*ast.MappingNode)
		if !ok {
			continue
		}
		for _, svcKey := range servicesMap.Values {
			svcMap, ok := svcKey.Value.(*ast.MappingNode)
			if !ok {
				continue
			}
			portsNode := findMappingValue(svcMap, "ports")
			if portsNode == nil {
				continue
			}
			seq, ok := portsNode.(*ast.SequenceNode)
			if !ok {
				continue
			}
			for _, item := range seq.Values {
				hostPort, err := parseHostPort(item.String())
				if err != nil {
					continue
				}
				ports = append(ports, hostPort)
			}
		}
	}

	return ports, nil
}

// GetExistingServices reads docker-compose.yaml and returns all service names.
func GetExistingServices() ([]string, error) {
	file, err := parseComposeFile()
	if err != nil {
		return nil, err
	}

	var names []string
	for _, doc := range file.Docs {
		body, ok := doc.Body.(*ast.MappingNode)
		if !ok {
			continue
		}
		services := findMappingValue(body, "services")
		if services == nil {
			continue
		}
		servicesMap, ok := services.(*ast.MappingNode)
		if !ok {
			continue
		}
		for _, svcKey := range servicesMap.Values {
			names = append(names, svcKey.Key.String())
		}
	}

	return names, nil
}

// AddServiceToCompose appends a new service to docker-compose.yaml using AST.
func AddServiceToCompose(params ServiceParams) error {
	file, err := parseComposeFile()
	if err != nil {
		return err
	}

	if len(file.Docs) == 0 {
		return fmt.Errorf("docker-compose.yaml: empty file")
	}

	doc := file.Docs[0]
	body, ok := doc.Body.(*ast.MappingNode)
	if !ok {
		return fmt.Errorf("docker-compose.yaml: root is not a mapping")
	}

	services := findMappingValue(body, "services")
	if services == nil {
		return fmt.Errorf("docker-compose.yaml: 'services' key not found")
	}

	servicesMap, ok := services.(*ast.MappingNode)
	if !ok {
		return fmt.Errorf("docker-compose.yaml: 'services' is not a mapping")
	}

	serviceName := fmt.Sprintf("%s-%s", params.ServiceType, params.InstanceName)

	// Check service doesn't exist
	for _, svcKey := range servicesMap.Values {
		if svcKey.Key.String() == serviceName {
			return fmt.Errorf("service %q already exists in docker-compose.yaml", serviceName)
		}
	}

	// Generate the full compose YAML with just the new service, then parse it.
	// This gives us AST nodes with correct indentation (2-space) because
	// goccy/go-yaml MarshalWithOptions produces the output with proper indent.
	serviceData, err := buildServiceData(params)
	if err != nil {
		return fmt.Errorf("building service data: %w", err)
	}

	fullCompose := map[string]interface{}{
		"services": map[string]interface{}{
			serviceName: serviceData,
		},
	}

	out, err := goccy.MarshalWithOptions(fullCompose, goccy.Indent(2), goccy.IndentSequence(true))
	if err != nil {
		return fmt.Errorf("marshaling service: %w", err)
	}

	newFile, err := parser.ParseBytes(out, 0)
	if err != nil {
		return fmt.Errorf("parsing generated YAML: %w", err)
	}

	if len(newFile.Docs) == 0 {
		return fmt.Errorf("generated YAML is empty")
	}

	newBody, ok := newFile.Docs[0].Body.(*ast.MappingNode)
	if !ok {
		return fmt.Errorf("generated YAML root is not a mapping")
	}

	newServices := findMappingValue(newBody, "services")
	if newServices == nil {
		return fmt.Errorf("generated YAML has no 'services' key")
	}

	newServicesMap, ok := newServices.(*ast.MappingNode)
	if !ok || len(newServicesMap.Values) == 0 {
		return fmt.Errorf("generated YAML 'services' has no entries")
	}

	svcEntry := newServicesMap.Values[0]

	// Add a blank comment line before the new service for readability.
	commentTk := token.New("", "", nil)
	svcEntry.SetComment(ast.CommentGroup([]*token.Token{commentTk}))

	// Add the service to the existing services mapping
	servicesMap.Values = append(servicesMap.Values, svcEntry)

	// Write back
	composePath := paths.ComposeFile()
	if err := os.WriteFile(composePath, []byte(file.String()), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", composePath, err)
	}

	return nil
}

// parseComposeFile parses docker-compose.yaml into an AST file.
// Automatically creates the base directory and compose file if they don't exist.
func parseComposeFile() (*ast.File, error) {
	if err := paths.EnsureBaseDir(); err != nil {
		return nil, err
	}

	composePath := paths.ComposeFile()
	data, err := os.ReadFile(composePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", composePath, err)
	}

	file, err := parser.ParseBytes(data, 0)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", composePath, err)
	}

	return file, nil
}

// findMappingValue finds a value by key in a MappingNode.
func findMappingValue(node *ast.MappingNode, key string) ast.Node {
	for _, v := range node.Values {
		if v.Key.String() == key {
			return v.Value
		}
	}
	return nil
}

// buildServiceData builds the Go map for a service, used for marshaling.
func buildServiceData(params ServiceParams) (interface{}, error) {
	switch params.ServiceType {
	case ServiceTypeGost:
		return buildGostServiceData(params), nil
	default:
		return nil, fmt.Errorf("unsupported service type: %s", params.ServiceType)
	}
}

// buildGostServiceData builds the Go map structure for a gost service.
func buildGostServiceData(params ServiceParams) map[string]interface{} {
	serviceName := fmt.Sprintf("%s-%s", params.ServiceType, params.InstanceName)
	containerName := fmt.Sprintf("tunnelium-%s-%s", params.ServiceType, params.InstanceName)
	command := GenerateGostCommand(params)

	svc := map[string]interface{}{
		"image":          "ginuerzh/gost",
		"container_name": containerName,
		"command":        command,
		"restart":        "always",
	}

	// Server: add volumes for TLS cert and host port mapping
	if params.GostRole == GostRoleServer {
		volumes := GenerateGostVolumes(params)
		if len(volumes) > 0 {
			svc["volumes"] = volumes
		}
		svc["ports"] = []string{fmt.Sprintf("%d:%d", params.HostSystemPort, params.HostSystemPort)}
	}

	// Client: configure entry points and port mappings
	if params.GostRole == GostRoleClient {
		var ports []string
		var volumes []string

		if params.GostSocksPort > 0 {
			ports = append(ports, fmt.Sprintf("%d:%d", params.GostSocksPort, params.GostSocksPort))
			authsPath := filepath.Join(paths.ServiceDir(serviceName), "auths.yaml")
			volumes = append(volumes, authsPath+":/etc/gost/auths.yaml:ro")
		}

		if params.GostHTTPPort > 0 {
			ports = append(ports, fmt.Sprintf("%d:%d", params.GostHTTPPort, params.GostHTTPPort))
		}

		if len(ports) > 0 {
			svc["ports"] = ports
		}
		if len(volumes) > 0 {
			svc["volumes"] = volumes
		}
	}

	return svc
}

// parseHostPort extracts the host port from a port mapping string.
func parseHostPort(portStr string) (int, error) {
	base := portStr
	for i, c := range portStr {
		if c == '/' {
			base = portStr[:i]
			break
		}
	}

	for i, c := range base {
		if c == ':' {
			hostPort, err := strconv.Atoi(base[:i])
			if err != nil {
				return 0, err
			}
			return hostPort, nil
		}
	}

	port, err := strconv.Atoi(base)
	if err != nil {
		return 0, fmt.Errorf("invalid port format: %s", portStr)
	}
	return port, nil
}

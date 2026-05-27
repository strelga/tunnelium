package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tunnelium/src/paths"
)

func TestValidateInstanceName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "incoming", false},
		{"valid with hyphen", "my-proxy", false},
		{"valid alphanumeric", "proxy123", false},
		{"valid starts with letter", "a", false},
		{"empty", "", true},
		{"starts with hyphen", "-bad", true},
		{"has underscore", "bad_name", true},
		{"has spaces", "bad name", true},
		{"has dots", "bad.name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInstanceName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstanceName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateServiceType(t *testing.T) {
	tests := []struct {
		name    string
		input   ServiceType
		wantErr bool
	}{
		{"gost is valid", ServiceTypeGost, false},
		{"nonexistent is invalid", ServiceType("nonexistent"), true},
		{"empty is invalid", ServiceType(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServiceType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServiceType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestParseHostPort(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"1081:1080", 1081, false},
		{"1081:1080/tcp", 1081, false},
		{"8080:80/udp", 8080, false},
		{"3000", 3000, false},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseHostPort(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHostPort(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseHostPort(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// setupTempCompose creates a temp docker-compose.yaml, overrides base dir, and returns a cleanup func.
func setupTempCompose(t *testing.T, content string) {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "etc"), 0755); err != nil {
		t.Fatal(err)
	}
	composePath := filepath.Join(tmpDir, "docker-compose.yaml")
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	paths.SetOverride(tmpDir)
	t.Cleanup(func() { paths.SetOverride("") })
}

func TestGetExistingServices(t *testing.T) {
	setupTempCompose(t, `services:
  gost-incoming:
    image: test
  gost-cross-dc:
    image: test
`)

	services, err := GetExistingServices()
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}
	if services[0] != "gost-incoming" || services[1] != "gost-cross-dc" {
		t.Errorf("unexpected services: %v", services)
	}
}

func TestGetUsedPorts(t *testing.T) {
	setupTempCompose(t, `services:
  gost-incoming:
    ports:
      - 1081:1080
  gost-cross-dc:
    ports:
      - 8080:80/tcp
`)

	ports, err := GetUsedPorts()
	if err != nil {
		t.Fatal(err)
	}
	if len(ports) != 2 {
		t.Fatalf("expected 2 ports, got %d: %v", len(ports), ports)
	}
	if ports[0] != 1081 || ports[1] != 8080 {
		t.Errorf("unexpected ports: %v", ports)
	}
}

func TestGetUsedPorts_EmptyCompose(t *testing.T) {
	setupTempCompose(t, `services:
  gost-incoming:
    image: test
`)

	ports, err := GetUsedPorts()
	if err != nil {
		t.Fatal(err)
	}
	if len(ports) != 0 {
		t.Fatalf("expected 0 ports, got %d: %v", len(ports), ports)
	}
}

func TestAddServiceToCompose_DuplicateService(t *testing.T) {
	setupTempCompose(t, `services:
  gost-incoming:
    image: test
`)

	params := ServiceParams{
		ServiceType:     ServiceTypeGost,
		InstanceName:    "incoming",
		GostRole:        GostRoleClient,
		GostSocksPort:   1081,
		GostNextHopHost: "192.0.2.10",
		GostNextHopPort: 443,
	}

	err := AddServiceToCompose(params)
	if err == nil {
		t.Error("expected error for duplicate service")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %v", err)
	}
}

// --- Gost command tests ---

func TestGenerateGostCommand_Client_SocksOnly(t *testing.T) {
	params := ServiceParams{
		ServiceType:     ServiceTypeGost,
		InstanceName:    "incoming",
		GostRole:        GostRoleClient,
		GostSocksPort:   1081,
		GostNextHopHost: "192.0.2.10",
		GostNextHopPort: 443,
	}

	cmd := GenerateGostCommand(params)

	if len(cmd) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(cmd), cmd)
	}
	if cmd[0] != "-L" || cmd[1] != "socks5://:1081?auths=/etc/gost/auths.yaml" {
		t.Errorf("unexpected listen arg: %s %s", cmd[0], cmd[1])
	}
	if cmd[2] != "-F" || cmd[3] != "relay+tls://192.0.2.10:443" {
		t.Errorf("unexpected forward arg: %s %s", cmd[2], cmd[3])
	}
}

func TestGenerateGostCommand_Client_HTTPOnly(t *testing.T) {
	params := ServiceParams{
		ServiceType:     ServiceTypeGost,
		InstanceName:    "http-proxy",
		GostRole:        GostRoleClient,
		GostHTTPPort:    8080,
		GostNextHopHost: "192.0.2.10",
		GostNextHopPort: 443,
	}

	cmd := GenerateGostCommand(params)

	if len(cmd) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(cmd), cmd)
	}
	if cmd[0] != "-L" || cmd[1] != "http://:8080" {
		t.Errorf("unexpected listen arg: %s %s", cmd[0], cmd[1])
	}
	if cmd[2] != "-F" || cmd[3] != "relay+tls://192.0.2.10:443" {
		t.Errorf("unexpected forward arg: %s %s", cmd[2], cmd[3])
	}
}

func TestGenerateGostCommand_Client_BothEntryPoints(t *testing.T) {
	params := ServiceParams{
		ServiceType:     ServiceTypeGost,
		InstanceName:    "incoming",
		GostRole:        GostRoleClient,
		GostSocksPort:   1081,
		GostHTTPPort:    8080,
		GostNextHopHost: "192.0.2.10",
		GostNextHopPort: 443,
	}

	cmd := GenerateGostCommand(params)

	if len(cmd) != 6 {
		t.Fatalf("expected 6 args, got %d: %v", len(cmd), cmd)
	}
	if cmd[0] != "-L" || cmd[1] != "socks5://:1081?auths=/etc/gost/auths.yaml" {
		t.Errorf("unexpected socks arg: %s %s", cmd[0], cmd[1])
	}
	if cmd[2] != "-L" || cmd[3] != "http://:8080" {
		t.Errorf("unexpected http arg: %s %s", cmd[2], cmd[3])
	}
	if cmd[4] != "-F" || cmd[5] != "relay+tls://192.0.2.10:443" {
		t.Errorf("unexpected forward arg: %s %s", cmd[4], cmd[5])
	}
}

func TestGenerateGostCommand_Client_DefaultNextHopPort(t *testing.T) {
	params := ServiceParams{
		ServiceType:     ServiceTypeGost,
		InstanceName:    "test",
		GostRole:        GostRoleClient,
		GostSocksPort:   1081,
		GostNextHopHost: "192.0.2.20",
	}

	cmd := GenerateGostCommand(params)

	if len(cmd) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(cmd), cmd)
	}
	if cmd[3] != "relay+tls://192.0.2.20:443" {
		t.Errorf("expected default next hop port 443, got: %s", cmd[3])
	}
}

func TestGenerateGostCommand_Server(t *testing.T) {
	params := ServiceParams{
		ServiceType:    ServiceTypeGost,
		InstanceName:   "relay-eu",
		HostSystemPort: 443,
		GostRole:       GostRoleServer,
	}

	cmd := GenerateGostCommand(params)

	if len(cmd) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(cmd), cmd)
	}
	if cmd[0] != "-L" {
		t.Errorf("expected -L flag, got: %s", cmd[0])
	}
	expected := "relay+tls://:443?cert=/cert.pem&key=/key.pem"
	if cmd[1] != expected {
		t.Errorf("expected %q, got %q", expected, cmd[1])
	}
}

// --- Gost volumes tests ---

func TestGenerateGostVolumes_Client(t *testing.T) {
	params := ServiceParams{
		ServiceType:  ServiceTypeGost,
		InstanceName: "cross-dc",
		GostRole:     GostRoleClient,
	}

	volumes := GenerateGostVolumes(params)
	if volumes != nil {
		t.Errorf("expected nil volumes for client, got: %v", volumes)
	}
}

func TestGenerateGostVolumes_Server(t *testing.T) {
	tmpDir := t.TempDir()
	paths.SetOverride(tmpDir)
	t.Cleanup(func() { paths.SetOverride("") })

	params := ServiceParams{
		ServiceType:  ServiceTypeGost,
		InstanceName: "relay-eu",
		GostRole:     GostRoleServer,
	}

	expectedTLS := filepath.Join(paths.ServiceDir("gost-relay-eu"), "tls.pem")

	volumes := GenerateGostVolumes(params)
	if len(volumes) != 2 {
		t.Fatalf("expected 2 volumes, got %d: %v", len(volumes), volumes)
	}
	if volumes[0] != expectedTLS+":/cert.pem:ro" {
		t.Errorf("unexpected cert volume: %s", volumes[0])
	}
	if volumes[1] != expectedTLS+":/key.pem:ro" {
		t.Errorf("unexpected key volume: %s", volumes[1])
	}
}

// --- TLS cert tests ---

func TestGenerateTLSCert(t *testing.T) {
	tmpDir := t.TempDir()
	tlsPath := tmpDir + "/tls.pem"

	if err := GenerateTLSCert(tlsPath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(tlsPath)
	if err != nil {
		t.Fatalf("reading TLS PEM: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("TLS PEM is empty")
	}

	content := string(data)
	if !strings.Contains(content, "BEGIN CERTIFICATE") {
		t.Error("TLS PEM should contain certificate")
	}
	if !strings.Contains(content, "BEGIN EC PRIVATE KEY") {
		t.Error("TLS PEM should contain EC private key")
	}
}

// --- Gost validation tests ---

func TestValidateGostParams_Client(t *testing.T) {
	// Valid client with socks
	params := ServiceParams{
		GostRole:        GostRoleClient,
		GostSocksPort:   1081,
		GostNextHopHost: "192.0.2.30",
		GostNextHopPort: 443,
	}
	if err := validateGostParams(params); err != nil {
		t.Errorf("valid client with socks should pass: %v", err)
	}

	// Valid client with http
	params = ServiceParams{
		GostRole:        GostRoleClient,
		GostHTTPPort:    8080,
		GostNextHopHost: "192.0.2.30",
		GostNextHopPort: 443,
	}
	if err := validateGostParams(params); err != nil {
		t.Errorf("valid client with http should pass: %v", err)
	}

	// Missing role
	params = ServiceParams{}
	if err := validateGostParams(params); err == nil {
		t.Error("missing role should fail")
	}

	// Missing entry points
	params = ServiceParams{GostRole: GostRoleClient, GostNextHopHost: "192.0.2.30", GostNextHopPort: 443}
	if err := validateGostParams(params); err == nil {
		t.Error("missing entry points should fail")
	}

	// Missing next hop host
	params = ServiceParams{GostRole: GostRoleClient, GostSocksPort: 1081, GostNextHopPort: 443}
	if err := validateGostParams(params); err == nil {
		t.Error("missing next_hop_host should fail")
	}

	// Invalid next hop port
	params = ServiceParams{GostRole: GostRoleClient, GostSocksPort: 1081, GostNextHopHost: "192.0.2.30", GostNextHopPort: 0}
	if err := validateGostParams(params); err == nil {
		t.Error("invalid next_hop_port should fail")
	}
}

func TestValidateGostParams_InvalidRole(t *testing.T) {
	params := ServiceParams{GostRole: "invalid"}
	if err := validateGostParams(params); err == nil {
		t.Error("invalid role should fail")
	}
}

// --- Compose tests ---

func TestAddServiceToCompose_GostClient_Socks(t *testing.T) {
	setupTempCompose(t, "services:\n  existing:\n    image: test\n")

	params := ServiceParams{
		ServiceType:     ServiceTypeGost,
		InstanceName:    "incoming",
		GostRole:        GostRoleClient,
		GostSocksPort:   1081,
		GostNextHopHost: "192.0.2.10",
		GostNextHopPort: 443,
	}

	if err := AddServiceToCompose(params); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(paths.ComposeFile())
	if err != nil {
		t.Fatal(err)
	}

	result := string(data)

	if !strings.Contains(result, "gost-incoming:") {
		t.Error("should contain gost-incoming service")
	}
	if !strings.Contains(result, "image: ginuerzh/gost") {
		t.Error("should contain gost image")
	}
	if !strings.Contains(result, "tunnelium-gost-incoming") {
		t.Error("should contain correct container name")
	}
	if !strings.Contains(result, "socks5://:1081?auths=/etc/gost/auths.yaml") {
		t.Error("should contain SOCKS5 listen command")
	}
	if !strings.Contains(result, "relay+tls://192.0.2.10:443") {
		t.Error("should contain relay+tls forward command")
	}
	if !strings.Contains(result, "1081:1081") {
		t.Error("should contain port mapping 1081:1081")
	}
	if !strings.Contains(result, "auths.yaml") {
		t.Error("should contain auths.yaml volume")
	}
}

func TestAddServiceToCompose_GostClient_BothEntryPoints(t *testing.T) {
	setupTempCompose(t, "services:\n  existing:\n    image: test\n")

	params := ServiceParams{
		ServiceType:     ServiceTypeGost,
		InstanceName:    "incoming",
		GostRole:        GostRoleClient,
		GostSocksPort:   1081,
		GostHTTPPort:    8080,
		GostNextHopHost: "192.0.2.10",
		GostNextHopPort: 443,
	}

	if err := AddServiceToCompose(params); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(paths.ComposeFile())
	if err != nil {
		t.Fatal(err)
	}

	result := string(data)

	if !strings.Contains(result, "1081:1081") {
		t.Error("should contain SOCKS port mapping")
	}
	if !strings.Contains(result, "8080:8080") {
		t.Error("should contain HTTP port mapping")
	}
	if !strings.Contains(result, "auths.yaml") {
		t.Error("should contain auths.yaml volume")
	}
}

func TestAddServiceToCompose_GostClient_HTTPOnly(t *testing.T) {
	setupTempCompose(t, "services:\n  existing:\n    image: test\n")

	params := ServiceParams{
		ServiceType:     ServiceTypeGost,
		InstanceName:    "http-proxy",
		GostRole:        GostRoleClient,
		GostHTTPPort:    8080,
		GostNextHopHost: "192.0.2.10",
		GostNextHopPort: 443,
	}

	if err := AddServiceToCompose(params); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(paths.ComposeFile())
	if err != nil {
		t.Fatal(err)
	}

	result := string(data)

	if !strings.Contains(result, "http://:8080") {
		t.Error("should contain HTTP listen command")
	}
	if !strings.Contains(result, "8080:8080") {
		t.Error("should contain HTTP port mapping")
	}
	// HTTP-only client should NOT have auths volume
	if strings.Contains(result, "auths.yaml") {
		t.Error("HTTP-only client should not have auths.yaml volume")
	}
}

func TestAddServiceToCompose_GostServer(t *testing.T) {
	setupTempCompose(t, "services:\n  existing:\n    image: test\n")

	params := ServiceParams{
		ServiceType:    ServiceTypeGost,
		InstanceName:   "relay-eu",
		HostSystemPort: 443,
		GostRole:       GostRoleServer,
	}

	if err := AddServiceToCompose(params); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(paths.ComposeFile())
	if err != nil {
		t.Fatal(err)
	}

	result := string(data)

	if !strings.Contains(result, "gost-relay-eu:") {
		t.Error("should contain gost-relay-eu service")
	}
	if !strings.Contains(result, "443:443") {
		t.Error("should contain port mapping 443:443")
	}
	if !strings.Contains(result, "tls.pem") {
		t.Error("should contain tls.pem volume mount")
	}
	if !strings.Contains(result, "relay+tls://:443?cert=/cert.pem&key=/key.pem") {
		t.Error("should contain relay+tls listen command")
	}
}

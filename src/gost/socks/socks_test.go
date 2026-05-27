package socks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tunnelium/src/paths"
)

func TestNewInstance(t *testing.T) {
	tmpDir := t.TempDir()
	paths.SetOverride(tmpDir)
	t.Cleanup(func() { paths.SetOverride("") })

	c := NewInstance("incoming")
	if c.Name != "incoming" {
		t.Fatalf("expected name 'incoming', got %q", c.Name)
	}
	if c.ContainerName != "tunnelium-gost-incoming" {
		t.Fatalf("expected container name 'tunnelium-gost-incoming', got %q", c.ContainerName)
	}
	expectedSuffix := filepath.Join("etc", "gost-incoming", "auths.yaml")
	if !strings.HasSuffix(c.AuthsFile, expectedSuffix) {
		t.Fatalf("expected auths file to end with %q, got %q", expectedSuffix, c.AuthsFile)
	}
	if !filepath.IsAbs(c.AuthsFile) {
		t.Fatalf("expected absolute auths file path, got %q", c.AuthsFile)
	}
}

func TestCreateUser_MissingUsername(t *testing.T) {
	c := &InstanceConfig{
		Name:          "test",
		ContainerName: "test-container",
		AuthsFile:     filepath.Join(t.TempDir(), "auths.yaml"),
	}
	_, err := c.CreateUser("", "pass")
	if err == nil {
		t.Fatal("expected error for empty username")
	}
}

func TestRemoveUser_MissingUsername(t *testing.T) {
	c := &InstanceConfig{
		Name:          "test",
		ContainerName: "test-container",
		AuthsFile:     filepath.Join(t.TempDir(), "auths.yaml"),
	}
	err := c.RemoveUser("")
	if err == nil {
		t.Fatal("expected error for empty username")
	}
}

func TestGenerateRandomPassword(t *testing.T) {
	pass, err := generateRandomPassword(12)
	if err != nil {
		t.Fatal(err)
	}
	if len(pass) != 12 {
		t.Fatalf("expected password length 12, got %d", len(pass))
	}
	for _, c := range pass {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			t.Fatalf("unexpected character in password: %c", c)
		}
	}
}

func TestGenerateRandomPassword_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for range 100 {
		pass, err := generateRandomPassword(16)
		if err != nil {
			t.Fatal(err)
		}
		if seen[pass] {
			t.Fatalf("duplicate password generated: %s", pass)
		}
		seen[pass] = true
	}
}

func TestReadAuthsFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "auths.yaml")

	content := `auths:
  - username: alice
    password: secret123
  - username: bob
    password: pass456
`
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	c := &InstanceConfig{
		Name:          "test",
		ContainerName: "test-container",
		AuthsFile:     file,
	}

	users, err := c.readAuthsFile()
	if err != nil {
		t.Fatal(err)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Username != "alice" || users[0].Password != "secret123" {
		t.Fatalf("unexpected first user: %+v", users[0])
	}
	if users[1].Username != "bob" || users[1].Password != "pass456" {
		t.Fatalf("unexpected second user: %+v", users[1])
	}
}

func TestReadAuthsFile_NonExistent(t *testing.T) {
	c := &InstanceConfig{
		Name:          "test",
		ContainerName: "test-container",
		AuthsFile:     "/nonexistent/path/auths.yaml",
	}

	users, err := c.readAuthsFile()
	if err != nil {
		t.Fatal(err)
	}
	if users != nil {
		t.Fatalf("expected nil users, got %v", users)
	}
}

func TestReadAuthsFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "auths.yaml")

	if err := os.WriteFile(file, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	c := &InstanceConfig{
		Name:          "test",
		ContainerName: "test-container",
		AuthsFile:     file,
	}

	users, err := c.readAuthsFile()
	if err != nil {
		t.Fatal(err)
	}
	if users != nil {
		t.Fatalf("expected nil users for empty file, got %v", users)
	}
}

func TestWriteAndReadAuthsFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "auths.yaml")

	c := &InstanceConfig{
		Name:          "test",
		ContainerName: "test-container",
		AuthsFile:     file,
	}

	auths := []AuthEntry{
		{Username: "alice", Password: "secret123"},
		{Username: "bob", Password: "pass456"},
	}

	if err := c.writeAuthsFile(auths); err != nil {
		t.Fatal(err)
	}

	read, err := c.readAuthsFile()
	if err != nil {
		t.Fatal(err)
	}

	if len(read) != 2 {
		t.Fatalf("expected 2 users, got %d", len(read))
	}
	if read[0].Username != "alice" || read[0].Password != "secret123" {
		t.Fatalf("unexpected first user: %+v", read[0])
	}
	if read[1].Username != "bob" || read[1].Password != "pass456" {
		t.Fatalf("unexpected second user: %+v", read[1])
	}
}

func TestListUsers(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "auths.yaml")

	content := `auths:
  - username: alice
    password: secret123
  - username: bob
    password: pass456
`
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	c := &InstanceConfig{
		Name:          "test",
		ContainerName: "test-container",
		AuthsFile:     file,
	}

	users, err := c.ListUsers()
	if err != nil {
		t.Fatal(err)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
}

func TestListUsers_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "auths.yaml")

	if err := os.WriteFile(file, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	c := &InstanceConfig{
		Name:          "test",
		ContainerName: "test-container",
		AuthsFile:     file,
	}

	users, err := c.ListUsers()
	if err != nil {
		t.Fatal(err)
	}

	if len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}
}

func TestCreateUser_WritesToFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "auths.yaml")

	c := &InstanceConfig{
		Name:          "test",
		ContainerName: "test-container",
		AuthsFile:     file,
	}

	result, err := c.CreateUser("alice", "secret123")
	if err != nil {
		t.Fatal(err)
	}

	if result.Username != "alice" {
		t.Fatalf("expected username 'alice', got %q", result.Username)
	}
	if result.Password != "secret123" {
		t.Fatalf("expected password 'secret123', got %q", result.Password)
	}

	// Verify file content
	users, err := c.readAuthsFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].Username != "alice" || users[0].Password != "secret123" {
		t.Fatalf("unexpected user: %+v", users[0])
	}
}

func TestCreateUser_GeneratesPassword(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "auths.yaml")

	c := &InstanceConfig{
		Name:          "test",
		ContainerName: "test-container",
		AuthsFile:     file,
	}

	result, err := c.CreateUser("alice", "")
	if err != nil {
		t.Fatal(err)
	}

	if result.Password == "" {
		t.Fatal("expected non-empty generated password")
	}
	if len(result.Password) != randomPasswordLength {
		t.Fatalf("expected password length %d, got %d", randomPasswordLength, len(result.Password))
	}
}

func TestCreateUser_DuplicateError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "auths.yaml")

	c := &InstanceConfig{
		Name:          "test",
		ContainerName: "test-container",
		AuthsFile:     file,
	}

	_, err := c.CreateUser("alice", "pass1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.CreateUser("alice", "pass2")
	if err == nil {
		t.Fatal("expected error for duplicate user")
	}
}

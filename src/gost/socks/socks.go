// Package socks provides functionality for managing SOCKS5 proxy users
// in a gost container via auths.yaml file and docker exec.
package socks

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"tunnelium/src/paths"

	"gopkg.in/yaml.v3"
)

const (
	randomPasswordLength = 12
	randomCharset        = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

// AuthEntry represents a username/password pair in auths.yaml.
type AuthEntry struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// AuthsFile represents the structure of auths.yaml.
type AuthsFile struct {
	Auths []AuthEntry `yaml:"auths"`
}

// InstanceConfig holds configuration for a gost SOCKS instance.
type InstanceConfig struct {
	// Name is the instance identifier (e.g. "incoming").
	Name string
	// ContainerName is the Docker container name (e.g. "tunnelium-gost-incoming").
	ContainerName string
	// AuthsFile is the path to the auths file on the host (e.g. "etc/gost-incoming/auths.yaml").
	AuthsFile string
}

// NewInstance creates an InstanceConfig for the given gost instance name.
// It derives container name and file paths from the name using the convention:
//
//	name "incoming" → container "tunnelium-gost-incoming", file "etc/gost-incoming/auths.yaml"
func NewInstance(name string) *InstanceConfig {
	return &InstanceConfig{
		Name:          name,
		ContainerName: fmt.Sprintf("tunnelium-gost-%s", name),
		AuthsFile:     filepath.Join(paths.ServiceDir(fmt.Sprintf("gost-%s", name)), "auths.yaml"),
	}
}

// CreateResult contains information about a created user.
type CreateResult struct {
	Username string
	Password string
}

// CreateUser adds a user to the auths file and reloads gost config via SIGHUP.
// If password is empty, a random 12-character alphanumeric password is generated.
func (c *InstanceConfig) CreateUser(username, password string) (*CreateResult, error) {
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}

	// Generate random password if not provided
	if password == "" {
		var err error
		password, err = generateRandomPassword(randomPasswordLength)
		if err != nil {
			return nil, fmt.Errorf("generating random password: %w", err)
		}
	}

	auths, err := c.readAuthsFile()
	if err != nil {
		return nil, fmt.Errorf("reading auths file: %w", err)
	}

	// Check if user already exists
	for _, a := range auths {
		if a.Username == username {
			return nil, fmt.Errorf("user %q already exists", username)
		}
	}

	auths = append(auths, AuthEntry{Username: username, Password: password})

	if err := c.writeAuthsFile(auths); err != nil {
		return nil, fmt.Errorf("writing auths file: %w", err)
	}

	// Reload gost config
	if err := c.reloadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to reload gost config: %v\n", err)
	}

	return &CreateResult{
		Username: username,
		Password: password,
	}, nil
}

// RemoveUser removes a user from the auths file and reloads gost config via SIGHUP.
func (c *InstanceConfig) RemoveUser(username string) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}

	auths, err := c.readAuthsFile()
	if err != nil {
		return fmt.Errorf("reading auths file: %w", err)
	}

	var kept []AuthEntry
	found := false
	for _, a := range auths {
		if a.Username == username {
			found = true
		} else {
			kept = append(kept, a)
		}
	}

	if !found {
		return fmt.Errorf("user %q not found", username)
	}

	if err := c.writeAuthsFile(kept); err != nil {
		return fmt.Errorf("writing auths file: %w", err)
	}

	// Reload gost config
	if err := c.reloadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to reload gost config: %v\n", err)
	}

	return nil
}

// ListUsers reads and returns all users from the auths file.
func (c *InstanceConfig) ListUsers() ([]AuthEntry, error) {
	return c.readAuthsFile()
}

// ReloadConfig sends SIGHUP to the gost process inside the container.
func (c *InstanceConfig) ReloadConfig() error {
	return c.reloadConfig()
}

// reloadConfig sends SIGHUP to gost inside the container via docker exec.
func (c *InstanceConfig) reloadConfig() error {
	if output, err := exec.Command("docker", "exec", c.ContainerName, "pkill", "-HUP", "-f", "gost").CombinedOutput(); err != nil {
		return fmt.Errorf("pkill -HUP gost in container: %s: %w", string(output), err)
	}
	return nil
}

// readAuthsFile parses the host auths.yaml file and returns all entries.
func (c *InstanceConfig) readAuthsFile() ([]AuthEntry, error) {
	data, err := os.ReadFile(c.AuthsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	var file AuthsFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing auths file: %w", err)
	}

	return file.Auths, nil
}

// writeAuthsFile writes the auths entries to the host auths.yaml file.
func (c *InstanceConfig) writeAuthsFile(auths []AuthEntry) error {
	file := AuthsFile{Auths: auths}
	data, err := yaml.Marshal(&file)
	if err != nil {
		return fmt.Errorf("marshaling auths: %w", err)
	}

	if err := os.WriteFile(c.AuthsFile, data, 0644); err != nil {
		return fmt.Errorf("writing auths file: %w", err)
	}

	return nil
}

// generateRandomPassword generates a cryptographically random alphanumeric password
// of the given length using crypto/rand.Read.
func generateRandomPassword(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = randomCharset[int(b[i])%len(randomCharset)]
	}
	return string(b), nil
}

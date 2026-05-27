// Package paths provides centralized file path resolution for tunnelium.
// All runtime artifacts (docker-compose.yaml, config directories) live under ~/.tunnelium/.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// override allows tests to substitute an alternative base directory.
var override string

// SetOverride sets an alternative base directory for testing.
// Pass empty string to restore default behavior.
func SetOverride(dir string) {
	override = dir
}

// BaseDir returns the tunnelium configuration directory.
// Default: ~/.tunnelium/ (resolved via os.UserHomeDir).
func BaseDir() string {
	if override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback for rare cases; should not happen in practice.
		return ".tunnelium"
	}
	return filepath.Join(home, ".tunnelium")
}

// ComposeFile returns the path to docker-compose.yaml.
func ComposeFile() string {
	return filepath.Join(BaseDir(), "docker-compose.yaml")
}

// EtcDir returns the path to the etc/ configuration directory.
func EtcDir() string {
	return filepath.Join(BaseDir(), "etc")
}

// ServiceDir returns the path to a specific service's configuration directory.
func ServiceDir(name string) string {
	return filepath.Join(EtcDir(), name)
}

// EnsureBaseDir creates the base directory and a minimal docker-compose.yaml
// if they don't exist yet. Called automatically before compose operations.
func EnsureBaseDir() error {
	base := BaseDir()
	if err := os.MkdirAll(base, 0755); err != nil {
		return fmt.Errorf("creating base dir %s: %w", base, err)
	}

	composePath := ComposeFile()
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		if err := os.WriteFile(composePath, []byte("services:\n"), 0644); err != nil {
			return fmt.Errorf("creating %s: %w", composePath, err)
		}
	}

	return nil
}

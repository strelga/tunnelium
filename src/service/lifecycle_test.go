package service

import (
	"strings"
	"testing"
)

func TestValidateServiceExists_Found(t *testing.T) {
	setupTempCompose(t, `services:
  gost-incoming:
    image: test
  gost-cross-dc:
    image: test
`)

	if err := validateServiceExists("gost-incoming"); err != nil {
		t.Errorf("expected no error for existing service, got: %v", err)
	}
}

func TestValidateServiceExists_NotFound(t *testing.T) {
	setupTempCompose(t, `services:
  gost-incoming:
    image: test
`)

	err := validateServiceExists("gost-nonexistent")
	if err == nil {
		t.Error("expected error for non-existing service")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestValidateServiceExists_EmptyCompose(t *testing.T) {
	setupTempCompose(t, "services: {}\n")

	err := validateServiceExists("gost-incoming")
	if err == nil {
		t.Error("expected error for service in empty compose")
	}
}

func TestValidateServiceExists_NullServicesKey(t *testing.T) {
	setupTempCompose(t, "services:\n")

	err := validateServiceExists("gost-incoming")
	if err == nil {
		t.Error("expected error for service in null services compose")
	}
}

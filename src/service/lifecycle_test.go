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

func TestListServices(t *testing.T) {
	setupTempCompose(t, `services:
  gost-incoming:
    image: test
  gost-relay-eu:
    image: test
`)

	services, err := ListServices()
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}

	if services[0].Name != "gost-incoming" {
		t.Errorf("expected first service name gost-incoming, got %s", services[0].Name)
	}
	if services[0].Type != "gost" {
		t.Errorf("expected type gost, got %s", services[0].Type)
	}
	if services[0].Instance != "incoming" {
		t.Errorf("expected instance incoming, got %s", services[0].Instance)
	}

	if services[1].Name != "gost-relay-eu" {
		t.Errorf("expected second service name gost-relay-eu, got %s", services[1].Name)
	}
	if services[1].Instance != "relay-eu" {
		t.Errorf("expected instance relay-eu, got %s", services[1].Instance)
	}
}

func TestListServices_Empty(t *testing.T) {
	setupTempCompose(t, "services: {}\n")

	services, err := ListServices()
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 0 {
		t.Errorf("expected 0 services, got %d", len(services))
	}
}

func TestServiceNames(t *testing.T) {
	setupTempCompose(t, `services:
  gost-incoming:
    image: test
  gost-cross-dc:
    image: test
`)

	names, err := ServiceNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "gost-incoming" || names[1] != "gost-cross-dc" {
		t.Errorf("unexpected names: %v", names)
	}
}

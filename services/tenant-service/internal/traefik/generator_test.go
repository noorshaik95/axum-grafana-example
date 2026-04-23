package traefik

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteConfig_EmitsTraefikAndEndpointsYaml(t *testing.T) {
	traefikDir := t.TempDir()
	tenantsDir := t.TempDir()

	g := NewGeneratorWithTenants(traefikDir, tenantsDir)
	if err := g.WriteConfig("t-123", "eastfield"); err != nil {
		t.Fatal(err)
	}

	// Traefik dynamic yml present.
	traefikPath := filepath.Join(traefikDir, "tenant-t-123.yml")
	if _, err := os.Stat(traefikPath); err != nil {
		t.Fatalf("traefik config missing: %v", err)
	}

	// Endpoints yaml present and contains all 8 services.
	epPath := filepath.Join(tenantsDir, "eastfield.yaml")
	raw, err := os.ReadFile(epPath)
	if err != nil {
		t.Fatalf("endpoints yaml missing: %v", err)
	}
	content := string(raw)
	for _, svc := range []string{"user-auth", "course", "assignment", "content", "video", "discussion", "scheduling", "ai"} {
		if !strings.Contains(content, svc+"-eastfield") {
			t.Fatalf("endpoints yaml missing %s hostname", svc)
		}
	}
}

func TestDeleteConfigAndEndpoints(t *testing.T) {
	traefikDir := t.TempDir()
	tenantsDir := t.TempDir()
	g := NewGeneratorWithTenants(traefikDir, tenantsDir)
	if err := g.WriteConfig("t-1", "eastfield"); err != nil {
		t.Fatal(err)
	}
	if err := g.DeleteConfig("t-1"); err != nil {
		t.Fatal(err)
	}
	if err := g.DeleteTenantEndpoints("eastfield"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(traefikDir, "tenant-t-1.yml")); !os.IsNotExist(err) {
		t.Fatalf("expected traefik config removed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(tenantsDir, "eastfield.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected endpoints yaml removed, got %v", err)
	}
	// Deleting again is a no-op (not an error).
	if err := g.DeleteConfig("t-1"); err != nil {
		t.Fatalf("second delete should be idempotent, got %v", err)
	}
	if err := g.DeleteTenantEndpoints("eastfield"); err != nil {
		t.Fatalf("second delete should be idempotent, got %v", err)
	}
}

package traefik

import (
	"fmt"
	"os"
	"path/filepath"
)

// Generator writes per-tenant Traefik dynamic config files.
// Traefik watches the config directory and hot-reloads on change.
type Generator struct {
	configDir string
}

// NewGenerator creates a Generator that writes YAML to configDir.
func NewGenerator(configDir string) *Generator {
	return &Generator{configDir: configDir}
}

// WriteConfig writes a Traefik dynamic configuration file for a tenant.
// The file routes {slug}.slate.local to the api-gateway, injecting X-Tenant-ID.
func (g *Generator) WriteConfig(tenantID, slug string) error {
	if err := os.MkdirAll(g.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	config := fmt.Sprintf(`http:
  routers:
    tenant-%s:
      rule: "Host(`+"`"+`%s.slate.local`+"`"+`)"
      service: api-gateway-svc
      middlewares:
        - inject-tenant-%s
  middlewares:
    inject-tenant-%s:
      headers:
        customRequestHeaders:
          X-Tenant-ID: "%s"
  services:
    api-gateway-svc:
      loadBalancer:
        servers:
          - url: "http://api-gateway:8080"
`, tenantID, slug, tenantID, tenantID, tenantID)

	path := filepath.Join(g.configDir, fmt.Sprintf("tenant-%s.yml", tenantID))
	if err := os.WriteFile(path, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write traefik config: %w", err)
	}

	return nil
}

// DeleteConfig removes the Traefik dynamic configuration file for a tenant.
func (g *Generator) DeleteConfig(tenantID string) error {
	path := filepath.Join(g.configDir, fmt.Sprintf("tenant-%s.yml", tenantID))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete traefik config: %w", err)
	}
	return nil
}

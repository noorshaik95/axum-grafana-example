# Infrastructure Plan — Traefik + Per-Tenant Subdomain Routing

## Owner Agent: `infra-expert`

## Stack: Traefik v3, Docker Compose, dnsmasq (local)

---

## Objective

Replace direct port exposure with Traefik as the ingress layer. Each tenant gets `{slug}.slate.local`. Admin, API Gateway, and shared services get fixed subdomains.

---

## Deliverables

### 1. `config/traefik/traefik.yml` (static config)

```yaml
api:
  dashboard: true
  insecure: true # dev only

entryPoints:
  web:
    address: ':80'
  websecure:
    address: ':443'

providers:
  docker:
    endpoint: 'unix:///var/run/docker.sock'
    exposedByDefault: false
    network: slate-network
  file:
    directory: /config/traefik/dynamic
    watch: true

log:
  level: INFO
```

### 2. `config/traefik/dynamic/base.yml` (static routes for shared services)

Routes:

- `admin.slate.local` → admin-frontend:3003
- `app.slate.local` → student-frontend:3000
- `teach.slate.local` → provider-frontend:3002
- `api.slate.local` → api-gateway:8080
- `traefik.slate.local` → Traefik dashboard

### 3. Docker Compose Updates

- Add `traefik` service to `docker-compose.yml`
- Remove hardcoded port mappings from frontend/api-gateway services
- Add `traefik.enable=true` labels to services
- Mount `/var/run/docker.sock` into Traefik for Docker provider

### 4. Per-Tenant Dynamic Config Generator

`services/tenant-service/internal/traefik/generator.go` writes YAML files to `config/traefik/dynamic/tenant-{id}.yml` when a tenant is provisioned. Traefik watches this dir and hot-reloads.

Template:

```yaml
http:
  routers:
    tenant-{id}-api:
      rule: 'Host(`{slug}.slate.local`)'
      service: tenant-{id}-api-gateway
      middlewares:
        - tenant-{id}-header
  middlewares:
    tenant-{id}-header:
      headers:
        customRequestHeaders:
          X-Tenant-ID: '{id}'
  services:
    tenant-{id}-api-gateway:
      loadBalancer:
        servers:
          - url: 'http://api-gateway:8080'
```

### 5. Local DNS Setup Script

`scripts/setup-local-dns.sh`:

- macOS: configure dnsmasq to route `*.slate.local → 127.0.0.1`
- Linux: add `/etc/hosts` entries or use systemd-resolved
- Docs: `README.local-dev.md`

---

## docker-compose.yml Changes

```yaml
traefik:
  image: traefik:v3
  ports:
    - '80:80'
    - '8090:8080' # dashboard
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock:ro
    - ./config/traefik:/config/traefik
  networks:
    - slate-network
  command:
    - --configFile=/config/traefik/traefik.yml
```

---

## Testing

- `curl -H "Host: api.slate.local" http://localhost/health` → 200
- `curl -H "Host: admin.slate.local" http://localhost` → admin frontend
- Provision test tenant → verify `test-uni.slate.local` routes correctly
- Traefik dashboard at `http://traefik.slate.local` shows all routes

---

## Files to Create/Modify

- [NEW] `config/traefik/traefik.yml`
- [NEW] `config/traefik/dynamic/base.yml`
- [NEW] `scripts/setup-local-dns.sh`
- [MODIFY] `docker-compose.yml` — add Traefik, update service labels
- [NEW] `services/tenant-service/internal/traefik/generator.go`

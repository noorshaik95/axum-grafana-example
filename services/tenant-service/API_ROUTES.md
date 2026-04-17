# Tenant Service API Routes

This document describes how the Tenant Service gRPC methods are exposed through the API Gateway as REST endpoints.

## API Gateway Integration

The tenant-service is integrated with the API Gateway using:

- **Auto-discovery**: Automatically maps standard CRUD operations to REST endpoints
- **Manual route overrides**: Custom mappings for non-standard operations

## Auto-Discovered Routes (Standard CRUD)

These routes are automatically discovered by the API Gateway based on naming conventions:

### Tenant Management

| HTTP Method | Path               | gRPC Method    | Description                           |
| ----------- | ------------------ | -------------- | ------------------------------------- |
| `POST`      | `/api/tenants`     | `CreateTenant` | Create a new tenant with provisioning |
| `GET`       | `/api/tenants/:id` | `GetTenant`    | Get tenant details by ID              |
| `PUT`       | `/api/tenants/:id` | `UpdateTenant` | Update tenant properties              |
| `DELETE`    | `/api/tenants/:id` | `DeleteTenant` | Delete a tenant                       |
| `GET`       | `/api/tenants`     | `ListTenants`  | List all tenants with filtering       |

**Example: Create Tenant**

```bash
POST /api/tenants
Content-Type: application/json

{
  "name": "Acme Corp",
  "domain": "acme",
  "tier": "PROFESSIONAL",
  "admin_email": "admin@acme.com",
  "admin_first_name": "John",
  "admin_last_name": "Doe",
  "admin_password": "SecurePass123!"
}
```

**Example: List Tenants**

```bash
GET /api/tenants?page=1&page_size=20&tier=PROFESSIONAL&is_active=true
```

## Manual Route Overrides (Custom Operations)

These routes require manual configuration because they don't follow the standard naming conventions:

### Provisioning Status

| HTTP Method | Path                    | gRPC Method             | Description                     |
| ----------- | ----------------------- | ----------------------- | ------------------------------- |
| `GET`       | `/api/provisioning/:id` | `GetProvisioningStatus` | Check async provisioning status |

**Example**

```bash
GET /api/provisioning/prov-12345
```

**Response**

```json
{
  "provisioning_id": "prov-12345",
  "status": "COMPLETED",
  "current_step": "completed",
  "progress_percentage": 100,
  "duration_seconds": 45
}
```

### Storage Management

| HTTP Method | Path                              | gRPC Method          | Description                   |
| ----------- | --------------------------------- | -------------------- | ----------------------------- |
| `GET`       | `/api/tenants/:tenant_id/storage` | `GetStorageQuota`    | Get storage quota and usage   |
| `PUT`       | `/api/tenants/:tenant_id/storage` | `UpdateStorageUsage` | Update storage usage tracking |

**Example: Get Storage Quota**

```bash
GET /api/tenants/tenant-123/storage
```

**Response**

```json
{
  "tenant_id": "tenant-123",
  "total_quota_bytes": 107374182400,
  "used_bytes": 5368709120,
  "available_bytes": 102005473280,
  "usage_percentage": 5.0,
  "file_count": 1250
}
```

**Example: Update Storage Usage**

```bash
PUT /api/tenants/tenant-123/storage
Content-Type: application/json

{
  "tenant_id": "tenant-123",
  "bytes_delta": 1048576,
  "file_count_delta": 1
}
```

### Admin Management

| HTTP Method | Path                                       | gRPC Method         | Description               |
| ----------- | ------------------------------------------ | ------------------- | ------------------------- |
| `POST`      | `/api/tenants/:tenant_id/admins`           | `CreateTenantAdmin` | Add a new admin to tenant |
| `GET`       | `/api/tenants/:tenant_id/admins/:admin_id` | `GetTenantAdmin`    | Get admin details         |

**Example: Create Tenant Admin**

```bash
POST /api/tenants/tenant-123/admins
Content-Type: application/json

{
  "tenant_id": "tenant-123",
  "email": "admin2@acme.com",
  "first_name": "Jane",
  "last_name": "Smith",
  "password": "SecurePass456!"
}
```

## Configuration

### Gateway Config Location

The API Gateway configuration is located at:

- Development: `config/gateway-config.yaml`
- Docker: `config/gateway-config.docker.yaml`

### Tenant Service Configuration

```yaml
services:
  tenant-service:
    name: 'tenant-service'
    endpoint: 'http://tenant-service:50052'
    timeout_ms: 10000 # 10s for provisioning operations
    connection_pool_size: 10
    auto_discover: true
    tls_enabled: false
    circuit_breaker:
      failure_threshold: 5
      success_threshold: 2
      timeout_seconds: 60
```

### Route Overrides

```yaml
route_overrides:
  - grpc_method: 'tenant.TenantService/GetProvisioningStatus'
    http_path: '/api/provisioning/:id'
    http_method: 'GET'
    service: 'tenant-service'

  - grpc_method: 'tenant.TenantService/GetStorageQuota'
    http_path: '/api/tenants/:tenant_id/storage'
    http_method: 'GET'
    service: 'tenant-service'

  - grpc_method: 'tenant.TenantService/UpdateStorageUsage'
    http_path: '/api/tenants/:tenant_id/storage'
    http_method: 'PUT'
    service: 'tenant-service'

  - grpc_method: 'tenant.TenantService/CreateTenantAdmin'
    http_path: '/api/tenants/:tenant_id/admins'
    http_method: 'POST'
    service: 'tenant-service'

  - grpc_method: 'tenant.TenantService/GetTenantAdmin'
    http_path: '/api/tenants/:tenant_id/admins/:admin_id'
    http_method: 'GET'
    service: 'tenant-service'
```

## Authentication & Authorization

All tenant management endpoints require authentication except for:

- Health checks
- Provisioning status (may be accessed via setup token)

Admin-level operations (create, update, delete tenants) require appropriate permissions from the auth service.

## Rate Limiting

The API Gateway applies rate limiting:

- Default: 100 requests per minute per IP
- Tenant creation has additional service-level rate limiting: 5 creations per hour per IP

## Circuit Breaker

The tenant-service has circuit breaker protection:

- **Failure Threshold**: 5 consecutive failures
- **Success Threshold**: 2 consecutive successes to close
- **Timeout**: 60 seconds before attempting recovery

This protects the gateway from cascading failures when the tenant service is unavailable.

## Service Discovery

The API Gateway automatically discovers new routes when:

1. The tenant-service starts up and registers its gRPC reflection endpoints
2. The discovery refresh interval triggers (every 5 minutes by default)
3. The gateway is restarted

No manual route registration is needed for standard CRUD operations following the naming conventions:

- `Get{Resource}` → `GET /api/{resources}/:id`
- `List{Resources}` → `GET /api/{resources}`
- `Create{Resource}` → `POST /api/{resources}`
- `Update{Resource}` → `PUT /api/{resources}/:id`
- `Delete{Resource}` → `DELETE /api/{resources}/:id`

## Acceptance Criteria Coverage

This API design satisfies Epic 9 acceptance criteria:

- **AC1**: `POST /api/tenants` creates tenant with name, domain, tier
- **AC2**: Professional+ tier gets dedicated database during provisioning
- **AC3**: Default admin created via `admin_email`, `admin_first_name`, `admin_last_name` in create request
- **AC4**: Welcome email sent automatically, setup URL returned in response
- **AC5**: Custom subdomain configured via `domain` field
- **AC6**: Storage quota set by tier, queryable via `GET /api/tenants/:tenant_id/storage`
- **AC7**: Provisioning status trackable via `GET /api/provisioning/:id`, must complete <120s

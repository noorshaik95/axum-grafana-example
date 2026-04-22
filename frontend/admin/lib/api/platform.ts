/**
 * Platform API modules — incidents, flags, broadcast, status, audit, admin users, platform metrics.
 *
 * Uses the shared apiClient (fetch-based with traceparent + X-Request-ID propagation per
 * plan/CONTRACTS.md "trace.propagation") so every call is stitched into a single Tempo trace.
 */

import { apiClient } from '../../../shared/lib/api/client'

// ---------------------------------------------------------------------------
// Incidents (plan/CONTRACTS.md → incident.IncidentService W4)
// ---------------------------------------------------------------------------

export type IncidentPriority = 'P0' | 'P1' | 'P2' | 'P3' | 'P4'
export type IncidentStatus = 'open' | 'watch' | 'resolved'
export type IncidentStatusFilter = 'open' | 'watch' | 'resolved'

export interface IncidentEvent {
  id: string
  incident_id: string
  kind: string
  message: string
  author?: string
  created_at_unix_ms: number
}

export interface Incident {
  id: string
  tenant_id?: string
  tenant_slug?: string
  priority: IncidentPriority
  status: IncidentStatus
  title: string
  impact: string
  service?: string
  created_by?: string
  opened_at_unix_ms: number
  resolved_at_unix_ms?: number
  events: IncidentEvent[]
}

export interface ListIncidentsParams {
  status?: IncidentStatusFilter
  tenant?: string
  limit?: number
}

export interface CreateIncidentRequest {
  tenant_id?: string
  priority: IncidentPriority
  title: string
  impact: string
  service?: string
}

export interface UpdateIncidentRequest {
  status?: IncidentStatus
  priority?: IncidentPriority
  title?: string
  impact?: string
}

export interface PostIncidentEventRequest {
  kind: string
  message: string
  author?: string
}

function toQuery(params?: Record<string, unknown>): string {
  if (!params) return ''
  const q = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null || v === '') continue
    q.set(k, String(v))
  }
  const s = q.toString()
  return s ? `?${s}` : ''
}

function asRecord<T extends object>(v?: T): Record<string, unknown> | undefined {
  return v as Record<string, unknown> | undefined
}

export const incidentsApi = {
  list(params?: ListIncidentsParams): Promise<{ incidents: Incident[] }> {
    return apiClient.get<{ incidents: Incident[] }>(`/api/incidents${toQuery(asRecord(params))}`)
  },
  get(id: string): Promise<Incident> {
    return apiClient.get<Incident>(`/api/incidents/${id}`)
  },
  create(data: CreateIncidentRequest): Promise<Incident> {
    return apiClient.post<Incident>('/api/incidents', data)
  },
  update(id: string, data: UpdateIncidentRequest): Promise<Incident> {
    return apiClient.patch<Incident>(`/api/incidents/${id}`, data)
  },
  postEvent(id: string, data: PostIncidentEventRequest): Promise<IncidentEvent> {
    return apiClient.post<IncidentEvent>(`/api/incidents/${id}/events`, data)
  },
}

// ---------------------------------------------------------------------------
// Feature flags (plan/CONTRACTS.md → feature-flag.FlagService W3)
// ---------------------------------------------------------------------------

export type FlagRuleType = 'all' | 'tenant' | 'role' | 'percentage' | 'user'

export interface FlagTargetRule {
  rule_type: FlagRuleType
  rule_value_json: string
}

export interface FeatureFlag {
  key: string
  description: string
  state: boolean
  targets: FlagTargetRule[]
  updated_at_unix_ms: number
}

export interface UpdateFlagRequest {
  state: boolean
  targets: FlagTargetRule[]
}

export const flagsApi = {
  list(): Promise<{ flags: FeatureFlag[] }> {
    return apiClient.get<{ flags: FeatureFlag[] }>('/api/flags')
  },
  update(key: string, data: UpdateFlagRequest): Promise<FeatureFlag> {
    return apiClient.put<FeatureFlag>(`/api/flags/${key}`, data)
  },
  delete(key: string): Promise<void> {
    return apiClient.delete<void>(`/api/flags/${key}`)
  },
}

// ---------------------------------------------------------------------------
// Broadcast (plan W15 email-service /api/broadcast)
// ---------------------------------------------------------------------------

export type BroadcastChannel = 'banner' | 'email' | 'push'
export type BroadcastTarget = 'all' | 'tenant' | 'role'

export interface BroadcastRequest {
  title: string
  message: string
  targets: BroadcastTarget[]
  channels: BroadcastChannel[]
  scheduled_at_unix_ms?: number
}

export interface BroadcastResponse {
  id: string
  status: 'scheduled' | 'sent'
  scheduled_at_unix_ms?: number
}

export const broadcastApi = {
  send(data: BroadcastRequest): Promise<BroadcastResponse> {
    return apiClient.post<BroadcastResponse>('/api/broadcast', data)
  },
}

// ---------------------------------------------------------------------------
// Public status (plan/CONTRACTS.md → incident.IncidentService GetPublicStatus)
// ---------------------------------------------------------------------------

export type OverallStatus = 'operational' | 'degraded' | 'outage'

export interface ComponentStatus {
  service: string
  status: OverallStatus
  highest_priority?: IncidentPriority
}

export interface PublicStatusResponse {
  overall: OverallStatus
  components: ComponentStatus[]
  incidents_last_7d: number
}

export const statusApi = {
  get(): Promise<PublicStatusResponse> {
    return apiClient.get<PublicStatusResponse>('/api/status')
  },
}

// ---------------------------------------------------------------------------
// Admin users + audit (plan/CONTRACTS.md → admin_auth.AdminAuthService W2)
// ---------------------------------------------------------------------------

export interface AdminUser {
  id: string
  email: string
  first_name: string
  last_name: string
  tenant_id?: string
  tenant_slug?: string
  roles: string[]
  is_active: boolean
  last_login_at_unix_ms?: number
  created_at_unix_ms: number
}

export interface ListAdminUsersParams {
  search?: string
  tenant?: string
  page?: number
  pageSize?: number
}

export interface AdminUsersResponse {
  users: AdminUser[]
  total: number
  page: number
  pageSize: number
}

export interface ImpersonateResponse {
  redirect_url: string
  impersonation_id: string
  expires_at_unix_ms: number
}

export const adminUsersApi = {
  list(params?: ListAdminUsersParams): Promise<AdminUsersResponse> {
    return apiClient.get<AdminUsersResponse>(`/api/admin/users${toQuery(asRecord(params))}`)
  },
  get(id: string): Promise<AdminUser> {
    return apiClient.get<AdminUser>(`/api/admin/users/${id}`)
  },
  impersonate(tenantId: string, userId: string): Promise<ImpersonateResponse> {
    return apiClient.post<ImpersonateResponse>(`/api/admin/impersonate/${tenantId}/${userId}`, {})
  },
}

export interface AuditEntry {
  id: string
  actor_id: string
  actor_email: string
  action: string
  target_type: string
  target_id: string
  tenant_id?: string
  request_id?: string
  details?: Record<string, unknown>
  created_at_unix_ms: number
}

export interface ListAuditParams {
  search?: string
  actor?: string
  action?: string
  start?: string
  end?: string
  page?: number
  pageSize?: number
}

export interface AuditResponse {
  entries: AuditEntry[]
  total: number
  page: number
  pageSize: number
}

export const auditApi = {
  list(params?: ListAuditParams): Promise<AuditResponse> {
    return apiClient.get<AuditResponse>(`/api/admin/audit${toQuery(asRecord(params))}`)
  },
}

// ---------------------------------------------------------------------------
// Platform metrics (plan W12 metrics-service /api/metrics/platform)
// ---------------------------------------------------------------------------

export interface PlatformMetrics {
  total_tenants: number
  active_tenants: number
  provisioning_tenants: number
  total_users: number
  monthly_active_users: number
  uptime_30d_pct: number
  open_incidents: number
  updated_at_unix_ms: number
}

export const platformMetricsApi = {
  get(): Promise<PlatformMetrics> {
    return apiClient.get<PlatformMetrics>('/api/metrics/platform')
  },
}

// ---------------------------------------------------------------------------
// Active alerts (plan W12 metrics-service /api/metrics/alerts)
// Proto: metrics.MetricsService/GetActiveAlerts → AlertsResponse
// ---------------------------------------------------------------------------

export type AlertSeverity = 'INFO' | 'WARNING' | 'CRITICAL' | 'SEVERITY_UNSPECIFIED'
export type AlertType =
  | 'ERROR_RATE_HIGH'
  | 'UPTIME_LOW'
  | 'STORAGE_QUOTA_EXCEEDED'
  | 'API_RATE_LIMIT_EXCEEDED'
  | 'DATABASE_CONNECTION_FAILED'
  | 'ALERT_TYPE_UNSPECIFIED'

export interface Alert {
  id: string
  type: AlertType
  severity: AlertSeverity
  title: string
  message: string
  tenant_id: string
  tenant_name: string
  threshold_value: number
  current_value: number
  is_active: boolean
  triggered_at?: string
  resolved_at?: string
  acknowledged_at?: string
  acknowledged_by?: string
}

export interface AlertsResponse {
  alerts: Alert[]
  total_count: number
}

export const alertsApi = {
  getActive(): Promise<AlertsResponse> {
    return apiClient.get<AlertsResponse>('/api/metrics/alerts')
  },
}

// ---------------------------------------------------------------------------
// Tenants (plan W16 tenant-service + status field closure)
// ---------------------------------------------------------------------------

export type TenantLifecycleStatus = 'provisioning' | 'active' | 'failed'

export interface TenantSummary {
  id: string
  slug: string
  name: string
  status: TenantLifecycleStatus
  status_detail?: string
  tier?: string
  seats_used?: number
  seats_cap?: number
  mau?: number
  created_at: string
  updated_at: string
}

export interface TenantDetail extends TenantSummary {
  admin_email?: string
  region?: string
  integrations?: Array<{ name: string; status: 'healthy' | 'degraded' | 'down' }>
  contract?: { value_usd: number; starts_at: string; ends_at: string }
}

export interface ListTenantsParams {
  status?: TenantLifecycleStatus
  search?: string
  page?: number
  pageSize?: number
}

export interface TenantsListResponse {
  tenants: TenantSummary[]
  total: number
}

export const platformTenantsApi = {
  list(params?: ListTenantsParams): Promise<TenantsListResponse> {
    return apiClient.get<TenantsListResponse>(`/api/tenants${toQuery(asRecord(params))}`)
  },
  get(id: string): Promise<TenantDetail> {
    return apiClient.get<TenantDetail>(`/api/tenants/${id}`)
  },
}

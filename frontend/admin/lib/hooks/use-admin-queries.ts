'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { onboardingApi } from '../api/onboarding'
import { tenantsApi, type ListTenantsParams } from '../api/tenants'
import { iamApi } from '../api/iam'
import { billingApi, type ListInvoicesParams } from '../api/billing'
import { healthApi } from '../api/health'
import { impersonationApi } from '../api/impersonation'
import {
  incidentsApi,
  flagsApi,
  broadcastApi,
  statusApi,
  adminUsersApi,
  auditApi,
  platformMetricsApi,
  platformTenantsApi,
  alertsApi,
  type ListIncidentsParams,
  type CreateIncidentRequest,
  type UpdateIncidentRequest,
  type PostIncidentEventRequest,
  type UpdateFlagRequest,
  type BroadcastRequest,
  type ListAdminUsersParams,
  type ListAuditParams as PlatformListAuditParams,
  type ListTenantsParams as PlatformListTenantsParams,
} from '../api/platform'
import type { ListParams, OnboardingPayload, ResourcePlan } from '../api/types'
import type { ListOnboardingParams } from '../api/onboarding'
import type { InviteUserRequest } from '../api/iam'

// Onboarding
export function useOnboardingJobs(params?: ListOnboardingParams) {
  return useQuery({
    queryKey: ['onboarding', 'list', params],
    queryFn: () => onboardingApi.list(params),
  })
}

export function useOnboardingJob(id: string) {
  return useQuery({
    queryKey: ['onboarding', id],
    queryFn: () => onboardingApi.get(id),
    enabled: !!id,
  })
}

export function useOnboardingStatus(id: string) {
  return useQuery({
    queryKey: ['onboarding', id, 'status'],
    queryFn: () => onboardingApi.getStatus(id),
    enabled: !!id,
    refetchInterval: 5000,
  })
}

export function useStartOnboarding() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: OnboardingPayload) => onboardingApi.start(payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['onboarding'] }),
  })
}

export function useApproveOnboarding() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => onboardingApi.approve(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['onboarding'] }),
  })
}

export function useRejectOnboarding() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      onboardingApi.reject(id, reason),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['onboarding'] }),
  })
}

// Tenants
export function useTenants(params?: ListTenantsParams) {
  return useQuery({
    queryKey: ['tenants', 'list', params],
    queryFn: () => tenantsApi.list(params),
  })
}

export function useTenant(id: string) {
  return useQuery({
    queryKey: ['tenants', id],
    queryFn: () => tenantsApi.get(id),
    enabled: !!id,
  })
}

export function useTenantUsage(id: string) {
  return useQuery({
    queryKey: ['tenants', id, 'usage'],
    queryFn: () => tenantsApi.getUsage(id),
    enabled: !!id,
  })
}

export function useTenantUsers(id: string, params?: ListParams) {
  return useQuery({
    queryKey: ['tenants', id, 'users', params],
    queryFn: () => tenantsApi.listUsers(id, params),
    enabled: !!id,
  })
}

export function useTenantBillingHistory(id: string) {
  return useQuery({
    queryKey: ['tenants', id, 'billing'],
    queryFn: () => tenantsApi.getBillingHistory(id),
    enabled: !!id,
  })
}

export function useToggleTenantAccess() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, enabled, reason }: { id: string; enabled: boolean; reason: string }) =>
      tenantsApi.toggleAccess(id, enabled, reason),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tenants'] }),
  })
}

export function useUpdateTenantPlan() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, plan }: { id: string; plan: ResourcePlan }) =>
      tenantsApi.updatePlan(id, plan),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tenants'] }),
  })
}

// IAM
export function useIAMUsers(params?: ListParams) {
  return useQuery({
    queryKey: ['iam', 'users', params],
    queryFn: () => iamApi.listUsers(params),
  })
}

export function useInviteUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: InviteUserRequest) => iamApi.inviteUser(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['iam', 'users'] }),
  })
}

export function useAdminRoles() {
  return useQuery({
    queryKey: ['iam', 'roles'],
    queryFn: () => iamApi.listRoles(),
  })
}

export function useUpdateRolePermissions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ roleId, permissions }: { roleId: string; permissions: string[] }) =>
      iamApi.updateRolePermissions(roleId, permissions),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['iam', 'roles'] }),
  })
}

export function useAuditLogs(params?: PlatformListAuditParams) {
  return useQuery({
    queryKey: ['admin', 'audit', params],
    queryFn: () => auditApi.list(params),
  })
}

// Billing
export function useBillingOverview() {
  return useQuery({
    queryKey: ['billing', 'overview'],
    queryFn: () => billingApi.getOverview(),
  })
}

export function useInvoices(params?: ListInvoicesParams) {
  return useQuery({
    queryKey: ['billing', 'invoices', params],
    queryFn: () => billingApi.listInvoices(params),
  })
}

export function useIssueCredit() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      tenantId,
      amount,
      reason,
    }: {
      tenantId: string
      amount: number
      reason: string
    }) => billingApi.issueCredit(tenantId, amount, reason),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['billing'] }),
  })
}

// Health
export function useServiceHealth() {
  return useQuery({
    queryKey: ['system', 'health'],
    queryFn: () => healthApi.getServiceHealth(),
    refetchInterval: 15000,
  })
}

export function usePlatformStats() {
  return useQuery({
    queryKey: ['system', 'stats'],
    queryFn: () => healthApi.getPlatformStats(),
    refetchInterval: 30000,
  })
}

export function useKafkaLag() {
  return useQuery({
    queryKey: ['system', 'kafka', 'lag'],
    queryFn: () => healthApi.getKafkaLag(),
    refetchInterval: 10000,
  })
}

// Impersonation
export function useImpersonationSessions(params?: ListParams) {
  return useQuery({
    queryKey: ['impersonation', 'sessions', params],
    queryFn: () => impersonationApi.listSessions(params),
  })
}

export function useStartImpersonation() {
  return useMutation({
    mutationFn: ({ targetUserId, reason }: { targetUserId: string; reason: string }) =>
      impersonationApi.start(targetUserId, reason),
  })
}

// Platform tenants (W16)
export function usePlatformTenants(params?: PlatformListTenantsParams) {
  return useQuery({
    queryKey: ['platform', 'tenants', params],
    queryFn: () => platformTenantsApi.list(params),
  })
}

export function usePlatformTenant(id: string) {
  return useQuery({
    queryKey: ['platform', 'tenants', id],
    queryFn: () => platformTenantsApi.get(id),
    enabled: !!id,
  })
}

// Incidents (W4)
export function useIncidents(params?: ListIncidentsParams) {
  return useQuery({
    queryKey: ['incidents', 'list', params],
    queryFn: () => incidentsApi.list(params),
    refetchInterval: 30000,
  })
}

export function useIncident(id: string) {
  return useQuery({
    queryKey: ['incidents', id],
    queryFn: () => incidentsApi.get(id),
    enabled: !!id,
  })
}

export function useCreateIncident() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateIncidentRequest) => incidentsApi.create(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['incidents'] }),
  })
}

export function useUpdateIncident() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateIncidentRequest }) =>
      incidentsApi.update(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['incidents'] }),
  })
}

export function usePostIncidentEvent() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: PostIncidentEventRequest }) =>
      incidentsApi.postEvent(id, data),
    onSuccess: (_r, vars) => {
      qc.invalidateQueries({ queryKey: ['incidents', vars.id] })
    },
  })
}

// Feature flags (W3)
export function useFeatureFlags() {
  return useQuery({
    queryKey: ['flags', 'list'],
    queryFn: () => flagsApi.list(),
  })
}

export function useUpdateFlag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ key, data }: { key: string; data: UpdateFlagRequest }) =>
      flagsApi.update(key, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['flags'] }),
  })
}

export function useDeleteFlag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (key: string) => flagsApi.delete(key),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['flags'] }),
  })
}

// Broadcast (W15)
export function useSendBroadcast() {
  return useMutation({
    mutationFn: (data: BroadcastRequest) => broadcastApi.send(data),
  })
}

// Public status (W4)
export function usePublicStatus() {
  return useQuery({
    queryKey: ['status', 'public'],
    queryFn: () => statusApi.get(),
    refetchInterval: 30000,
  })
}

// Admin users (W2)
export function useAdminUsers(params?: ListAdminUsersParams) {
  return useQuery({
    queryKey: ['admin', 'users', params],
    queryFn: () => adminUsersApi.list(params),
  })
}

export function useAdminUser(id: string) {
  return useQuery({
    queryKey: ['admin', 'users', id],
    queryFn: () => adminUsersApi.get(id),
    enabled: !!id,
  })
}

export function useStartAdminImpersonation() {
  return useMutation({
    mutationFn: ({ tenantId, userId }: { tenantId: string; userId: string }) =>
      adminUsersApi.impersonate(tenantId, userId),
  })
}

// Platform metrics (W12)
export function usePlatformMetrics() {
  return useQuery({
    queryKey: ['metrics', 'platform'],
    queryFn: () => platformMetricsApi.get(),
    refetchInterval: 30000,
  })
}

// Active alerts (W12 metrics.MetricsService/GetActiveAlerts)
export function useActiveAlerts() {
  return useQuery({
    queryKey: ['metrics', 'alerts'],
    queryFn: () => alertsApi.getActive(),
    refetchInterval: 30_000,
  })
}

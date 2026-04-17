'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { onboardingApi } from '../api/onboarding'
import { tenantsApi, type ListTenantsParams } from '../api/tenants'
import { iamApi, type ListAuditParams } from '../api/iam'
import { billingApi, type ListInvoicesParams } from '../api/billing'
import { healthApi } from '../api/health'
import { impersonationApi } from '../api/impersonation'
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

export function useAuditLogs(params?: ListAuditParams) {
  return useQuery({
    queryKey: ['iam', 'audit', params],
    queryFn: () => iamApi.listAuditLogs(params),
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

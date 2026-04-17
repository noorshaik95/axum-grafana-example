// Shared types for the admin API layer

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

export interface ListParams {
  page?: number
  pageSize?: number
  search?: string
  sortBy?: string
  sortOrder?: 'asc' | 'desc'
}

// Onboarding types
export type OnboardingStatus =
  | 'draft'
  | 'pending_review'
  | 'approved'
  | 'provisioning'
  | 'active'
  | 'failed'
  | 'rejected'

export interface OnboardingJob {
  id: string
  institutionName: string
  status: OnboardingStatus
  currentStep: number
  totalSteps: number
  adminEmail: string
  createdAt: string
  updatedAt: string
  approvedBy?: string
  approvedAt?: string
  failureReason?: string
}

export interface InstitutionDetails {
  name: string
  address: string
  city: string
  state: string
  country: string
  postalCode: string
  phone: string
  website: string
  logoUrl?: string
  timezone: string
  locale: string
}

export interface AdminUserDetails {
  email: string
  firstName: string
  lastName: string
  phone?: string
}

export interface ResourcePlan {
  planTier: 'starter' | 'standard' | 'enterprise'
  maxUsers: number
  maxCourses: number
  storageQuotaGb: number
  features: string[]
}

export interface OnboardingPayload {
  institutionDetails: InstitutionDetails
  adminUser: AdminUserDetails
  resourcePlan: ResourcePlan
}

// Tenant types
export type TenantStatus = 'active' | 'suspended' | 'pending' | 'in_review'

export interface Tenant {
  id: string
  name: string
  slug: string
  status: TenantStatus
  domain: string
  adminEmail: string
  plan: string
  maxUsers: number
  currentUsers: number
  storageUsedGb: number
  storageQuotaGb: number
  createdAt: string
  updatedAt: string
}

export interface TenantUsage {
  tenantId: string
  activeUsers: number
  totalCourses: number
  storageUsedGb: number
  bandwidthUsedGb: number
  apiCallsToday: number
}

// IAM types
export interface AdminRole {
  id: string
  name: string
  description: string
  permissions: string[]
  userCount: number
  isSystem: boolean
  createdAt: string
}

export interface AuditLogEntry {
  id: string
  userId: string
  userName: string
  action: string
  resource: string
  resourceId: string
  details: string
  ipAddress: string
  timestamp: string
}

// Billing types
export interface BillingOverview {
  totalRevenue: number
  monthlyRecurring: number
  activeTenants: number
  averageRevenuePerTenant: number
  outstandingInvoices: number
  revenueGrowth: number
}

export interface Invoice {
  id: string
  tenantId: string
  tenantName: string
  amount: number
  status: 'paid' | 'pending' | 'overdue' | 'cancelled'
  issuedAt: string
  dueAt: string
  paidAt?: string
}

export interface TenantBillingHistory {
  tenantId: string
  invoices: Invoice[]
  credits: CreditAdjustment[]
  currentBalance: number
}

export interface CreditAdjustment {
  id: string
  amount: number
  reason: string
  createdBy: string
  createdAt: string
}

// System health types
export type ServiceStatus = 'healthy' | 'degraded' | 'down' | 'unknown'

export interface ServiceHealth {
  name: string
  status: ServiceStatus
  latencyMs: number
  uptime: number
  lastCheck: string
  version: string
  errorRate: number
}

export interface KafkaConsumerLag {
  topic: string
  consumerGroup: string
  partition: number
  currentOffset: number
  endOffset: number
  lag: number
}

export interface PlatformStats {
  activeTenants: number
  totalUsers: number
  dailyActiveUsers: number
  storageUsedTb: number
  errorRate: number
  avgResponseMs: number
}

// Impersonation types
export interface ImpersonationSession {
  id: string
  adminUserId: string
  adminUserName: string
  targetUserId: string
  targetUserName: string
  targetTenantId: string
  targetTenantName: string
  startedAt: string
  endedAt?: string
  reason: string
}

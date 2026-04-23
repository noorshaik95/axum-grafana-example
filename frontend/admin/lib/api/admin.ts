import apiClient from './client'

export interface UsageStats {
  total_users: number
  active_users: number
  total_storage_used: number
  total_storage_limit: number
  bandwidth_used: number
  bandwidth_limit: number
  ai_credits_used: number
  ai_credits_limit: number
  period_start: string
  period_end: string
}

export interface BillingInfo {
  current_plan: {
    name: string
    price: number
    billing_cycle: 'monthly' | 'yearly'
    features: string[]
  }
  next_billing_date: string
  payment_method: {
    type: string
    last4?: string
  }
  billing_history: BillingTransaction[]
}

export interface BillingTransaction {
  id: string
  date: string
  amount: number
  status: 'paid' | 'pending' | 'failed'
  description: string
  invoice_url?: string
}

export interface Plan {
  id: string
  name: string
  description: string
  price: number
  billing_cycle: 'monthly' | 'yearly'
  features: string[]
  limits: {
    users: number
    storage: number
    bandwidth: number
    ai_credits: number
  }
}

export interface ResourcePurchase {
  type: 'storage' | 'bandwidth' | 'ai_credits'
  amount: number
  unit_price: number
  total_price: number
}

export interface CostOptimization {
  current_cost: number
  potential_savings: number
  savings_percentage: number
  recommendations: {
    id: string
    title: string
    description: string
    potential_savings: number
    impact: 'low' | 'medium' | 'high'
    implementation_effort: 'easy' | 'medium' | 'hard'
  }[]
}

export const adminService = {
  async getUsageStats(params?: { start_date?: string; end_date?: string }): Promise<UsageStats> {
    const response = await apiClient.get('/admin/usage/stats', { params })
    return response.data
  },

  async getBillingInfo(): Promise<BillingInfo> {
    const response = await apiClient.get('/admin/billing/info')
    return response.data
  },

  async getPlans(): Promise<Plan[]> {
    const response = await apiClient.get('/admin/billing/plans')
    return response.data
  },

  // Admin billing + cost-optimization write ops have no backend RPCs yet
  // (#52). Reject with a clear message so existing mutation .catch()
  // handlers surface red toasts instead of 404s in devtools.
  async upgradePlan(_planId: string): Promise<{ success: boolean; message: string }> {
    return Promise.reject(new Error('Not available in MVP — backend endpoint not yet implemented'))
  },

  async purchaseResources(
    _purchase: ResourcePurchase
  ): Promise<{ success: boolean; message: string }> {
    return Promise.reject(new Error('Not available in MVP — backend endpoint not yet implemented'))
  },

  async updatePaymentMethod(_paymentMethod: {
    type: string
    token: string
  }): Promise<{ success: boolean; message: string }> {
    return Promise.reject(new Error('Not available in MVP — backend endpoint not yet implemented'))
  },

  async getCostOptimization(): Promise<CostOptimization> {
    const response = await apiClient.get('/admin/cost-optimization')
    return response.data
  },

  async applyOptimization(
    _recommendationId: string
  ): Promise<{ success: boolean; message: string }> {
    return Promise.reject(new Error('Not available in MVP — backend endpoint not yet implemented'))
  },

  async getAnalytics(params: {
    metric: 'users' | 'storage' | 'bandwidth' | 'ai_credits'
    granularity: 'hour' | 'day' | 'week' | 'month'
    start_date: string
    end_date: string
  }): Promise<{ labels: string[]; values: number[] }> {
    const response = await apiClient.get('/admin/analytics', { params })
    return response.data
  },
}

export default adminService

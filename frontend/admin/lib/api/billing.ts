import apiClient from './client'
import type {
  BillingOverview,
  Invoice,
  CreditAdjustment,
  PaginatedResponse,
  ListParams,
} from './types'

export interface ListInvoicesParams extends ListParams {
  status?: Invoice['status']
  tenantId?: string
}

export const billingApi = {
  async getOverview(): Promise<BillingOverview> {
    const response = await apiClient.get('/billing/overview')
    return response.data
  },

  async listInvoices(params?: ListInvoicesParams): Promise<PaginatedResponse<Invoice>> {
    const response = await apiClient.get('/billing/invoices', { params })
    return response.data
  },

  async issueCredit(tenantId: string, amount: number, reason: string): Promise<CreditAdjustment> {
    const response = await apiClient.post(`/billing/tenants/${tenantId}/credits`, {
      amount,
      reason,
    })
    return response.data
  },
}

export default billingApi

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

const NOT_IMPLEMENTED = 'Not available in MVP — backend endpoint not yet implemented'

export const billingApi = {
  async getOverview(): Promise<BillingOverview> {
    const response = await apiClient.get('/billing/overview')
    return response.data
  },

  async listInvoices(params?: ListInvoicesParams): Promise<PaginatedResponse<Invoice>> {
    const response = await apiClient.get('/billing/invoices', { params })
    return response.data
  },

  // No backend RPC for admin credit grants yet (#52). Reject with a clear
  // message so the existing mutation .catch() surfaces a red toast instead
  // of a 404 in devtools.
  async issueCredit(
    _tenantId: string,
    _amount: number,
    _reason: string
  ): Promise<CreditAdjustment> {
    return Promise.reject(new Error(NOT_IMPLEMENTED))
  },
}

export default billingApi

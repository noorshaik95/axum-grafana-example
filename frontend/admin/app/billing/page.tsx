'use client'

import { useState } from 'react'
import { useBillingOverview, useInvoices } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { formatDate, formatCurrency } from '@/lib/utils'
import {
  TrendingUp,
  DollarSign,
  AlertCircle,
  Users,
  Loader2,
  ChevronUp,
  ChevronDown,
} from 'lucide-react'
import type { ListInvoicesParams } from '@/lib/api/billing'

function statusBadge(status: string) {
  const map: Record<string, string> = {
    paid: 'bg-green-100 text-green-800',
    pending: 'bg-yellow-100 text-yellow-800',
    overdue: 'bg-red-100 text-red-800',
    cancelled: 'bg-gray-100 text-gray-500',
  }
  return map[status] ?? 'bg-gray-100 text-gray-600'
}

function KpiCard({
  label,
  value,
  sub,
  icon: Icon,
  trend,
}: {
  label: string
  value: string
  sub?: string
  icon: React.ElementType
  trend?: number
}) {
  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center gap-4">
          <div className="rounded-xl bg-primary/10 p-3">
            <Icon className="h-5 w-5 text-primary" />
          </div>
          <div className="flex-1">
            <p className="text-sm text-muted-foreground">{label}</p>
            <p className="text-2xl font-bold">{value}</p>
            {sub && <p className="text-xs text-muted-foreground">{sub}</p>}
          </div>
          {trend !== undefined && (
            <div
              className={`flex items-center gap-1 text-sm font-medium ${trend >= 0 ? 'text-green-600' : 'text-red-600'}`}
            >
              {trend >= 0 ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
              {Math.abs(trend).toFixed(1)}%
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

export default function BillingPage() {
  const { data: overview, isLoading: loadingOverview } = useBillingOverview()
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)

  const invoiceParams: ListInvoicesParams = {
    page,
    pageSize: 25,
    status: (statusFilter as any) || undefined,
    search: search || undefined,
  }
  const { data: invoices, isLoading: loadingInvoices } = useInvoices(invoiceParams)

  const rows = invoices?.data ?? []
  const totalPages = invoices?.totalPages ?? 1

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold">Billing Overview</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Platform-wide revenue and invoice management.
        </p>
      </div>

      {/* KPI Cards */}
      {loadingOverview ? (
        <div className="grid grid-cols-4 gap-4">
          {[0, 1, 2, 3].map((i) => (
            <Card key={i}>
              <CardContent className="pt-6">
                <div className="h-16 animate-pulse rounded-lg bg-muted" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : overview ? (
        <div className="grid grid-cols-4 gap-4">
          <KpiCard
            label="Monthly Recurring Revenue"
            value={formatCurrency(overview.monthlyRecurring)}
            icon={TrendingUp}
            trend={overview.revenueGrowth}
          />
          <KpiCard
            label="Total Revenue"
            value={formatCurrency(overview.totalRevenue)}
            icon={DollarSign}
            sub="all time"
          />
          <KpiCard
            label="Active Tenants"
            value={overview.activeTenants.toString()}
            icon={Users}
            sub={`avg ${formatCurrency(overview.averageRevenuePerTenant)}/tenant`}
          />
          <KpiCard
            label="Outstanding Invoices"
            value={formatCurrency(overview.outstandingInvoices)}
            icon={AlertCircle}
            sub="awaiting payment"
          />
        </div>
      ) : null}

      {/* Invoice List */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">All Invoices</CardTitle>
            <div className="flex gap-3">
              <Input
                placeholder="Search tenant…"
                className="w-52"
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value)
                  setPage(1)
                }}
              />
              <select
                className="rounded-lg border bg-background px-3 py-2 text-sm"
                value={statusFilter}
                onChange={(e) => {
                  setStatusFilter(e.target.value)
                  setPage(1)
                }}
              >
                <option value="">All statuses</option>
                <option value="paid">Paid</option>
                <option value="pending">Pending</option>
                <option value="overdue">Overdue</option>
                <option value="cancelled">Cancelled</option>
              </select>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {loadingInvoices ? (
            <div className="flex justify-center py-10">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : rows.length === 0 ? (
            <p className="py-10 text-center text-sm text-muted-foreground">No invoices found</p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-muted-foreground">
                  <th className="pb-3 text-left font-medium">ID</th>
                  <th className="pb-3 text-left font-medium">University</th>
                  <th className="pb-3 text-left font-medium">Amount</th>
                  <th className="pb-3 text-left font-medium">Issued</th>
                  <th className="pb-3 text-left font-medium">Due</th>
                  <th className="pb-3 text-left font-medium">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {rows.map((inv) => (
                  <tr key={inv.id} className="hover:bg-muted/30 transition-colors">
                    <td className="py-3 font-mono text-xs text-muted-foreground">
                      {inv.id.slice(0, 8)}…
                    </td>
                    <td className="py-3 font-medium">{inv.tenantName}</td>
                    <td className="py-3 font-semibold">{formatCurrency(inv.amount)}</td>
                    <td className="py-3 text-muted-foreground">{formatDate(inv.issuedAt)}</td>
                    <td
                      className={`py-3 ${inv.status === 'overdue' ? 'text-red-600 font-medium' : 'text-muted-foreground'}`}
                    >
                      {formatDate(inv.dueAt)}
                    </td>
                    <td className="py-3">
                      <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusBadge(inv.status)}`}
                      >
                        {inv.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {totalPages > 1 && (
            <div className="flex items-center justify-between border-t pt-4 mt-4">
              <p className="text-sm text-muted-foreground">
                Page {page} of {totalPages}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page === 1}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page === totalPages}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

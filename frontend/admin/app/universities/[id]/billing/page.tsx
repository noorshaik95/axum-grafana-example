'use client'

import { useParams } from 'next/navigation'
import Link from 'next/link'
import { useState } from 'react'
import { useTenant, useTenantBillingHistory, useIssueCredit } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { formatDate, formatCurrency } from '@/lib/utils'
import { ChevronLeft, Loader2, Plus, CreditCard, Receipt, TrendingUp, X } from 'lucide-react'

function statusBadge(status: string) {
  const map: Record<string, string> = {
    paid: 'bg-green-100 text-green-800',
    pending: 'bg-yellow-100 text-yellow-800',
    overdue: 'bg-red-100 text-red-800',
    cancelled: 'bg-gray-100 text-gray-600',
  }
  return map[status] ?? 'bg-gray-100 text-gray-600'
}

export default function UniversityBillingPage() {
  const params = useParams()
  const id = params.id as string
  const { data: tenant, isLoading: loadingTenant } = useTenant(id)
  const { data: billing, isLoading: loadingBilling } = useTenantBillingHistory(id)
  const issueCredit = useIssueCredit()

  const [showCreditModal, setShowCreditModal] = useState(false)
  const [creditAmount, setCreditAmount] = useState('')
  const [creditReason, setCreditReason] = useState('')

  async function handleIssueCredit() {
    const amount = parseFloat(creditAmount)
    if (!amount || !creditReason.trim()) return
    await issueCredit.mutateAsync({ tenantId: id, amount, reason: creditReason })
    setShowCreditModal(false)
    setCreditAmount('')
    setCreditReason('')
  }

  if (loadingTenant || loadingBilling) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  const invoices = billing?.invoices ?? []
  const credits = billing?.credits ?? []
  const balance = billing?.currentBalance ?? 0

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link href={`/universities/${id}`}>
            <Button variant="ghost" size="icon">
              <ChevronLeft className="h-5 w-5" />
            </Button>
          </Link>
          <div>
            <h1 className="text-2xl font-bold">Billing</h1>
            <p className="text-sm text-muted-foreground">{tenant?.name}</p>
          </div>
        </div>
        <Button onClick={() => setShowCreditModal(true)}>
          <Plus className="mr-2 h-4 w-4" /> Issue Credit
        </Button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-3 gap-4">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <div className="rounded-lg bg-blue-100 p-2">
                <TrendingUp className="h-5 w-5 text-blue-600" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Current Balance</p>
                <p
                  className={`text-xl font-bold ${balance < 0 ? 'text-red-600' : 'text-green-600'}`}
                >
                  {formatCurrency(balance)}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <div className="rounded-lg bg-green-100 p-2">
                <Receipt className="h-5 w-5 text-green-600" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Total Invoiced</p>
                <p className="text-xl font-bold">
                  {formatCurrency(invoices.reduce((s, i) => s + i.amount, 0))}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <div className="rounded-lg bg-purple-100 p-2">
                <CreditCard className="h-5 w-5 text-purple-600" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Total Credits</p>
                <p className="text-xl font-bold">
                  {formatCurrency(credits.reduce((s, c) => s + c.amount, 0))}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Invoices Table */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Invoice History</CardTitle>
        </CardHeader>
        <CardContent>
          {invoices.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">No invoices yet</p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-muted-foreground">
                  <th className="pb-3 text-left font-medium">Invoice ID</th>
                  <th className="pb-3 text-left font-medium">Amount</th>
                  <th className="pb-3 text-left font-medium">Issued</th>
                  <th className="pb-3 text-left font-medium">Due</th>
                  <th className="pb-3 text-left font-medium">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {invoices.map((inv) => (
                  <tr key={inv.id}>
                    <td className="py-3 font-mono text-xs text-muted-foreground">
                      {inv.id.slice(0, 8)}…
                    </td>
                    <td className="py-3 font-semibold">{formatCurrency(inv.amount)}</td>
                    <td className="py-3">{formatDate(inv.issuedAt)}</td>
                    <td className="py-3">{formatDate(inv.dueAt)}</td>
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
        </CardContent>
      </Card>

      {/* Credits Table */}
      {credits.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Credit Adjustments</CardTitle>
          </CardHeader>
          <CardContent>
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-muted-foreground">
                  <th className="pb-3 text-left font-medium">Amount</th>
                  <th className="pb-3 text-left font-medium">Reason</th>
                  <th className="pb-3 text-left font-medium">Issued By</th>
                  <th className="pb-3 text-left font-medium">Date</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {credits.map((c) => (
                  <tr key={c.id}>
                    <td className="py-3 font-semibold text-green-600">
                      +{formatCurrency(c.amount)}
                    </td>
                    <td className="py-3 text-muted-foreground">{c.reason}</td>
                    <td className="py-3">{c.createdBy}</td>
                    <td className="py-3">{formatDate(c.createdAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </CardContent>
        </Card>
      )}

      {/* Credit Modal */}
      {showCreditModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="w-full max-w-md rounded-2xl bg-background p-6 shadow-xl">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold">Issue Credit</h2>
              <Button variant="ghost" size="icon" onClick={() => setShowCreditModal(false)}>
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Amount (USD)</Label>
                <Input
                  type="number"
                  placeholder="0.00"
                  value={creditAmount}
                  onChange={(e) => setCreditAmount(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>Reason</Label>
                <Input
                  placeholder="e.g. Service disruption compensation"
                  value={creditReason}
                  onChange={(e) => setCreditReason(e.target.value)}
                />
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <Button variant="outline" onClick={() => setShowCreditModal(false)}>
                  Cancel
                </Button>
                <Button
                  onClick={handleIssueCredit}
                  disabled={!creditAmount || !creditReason || issueCredit.isPending}
                >
                  {issueCredit.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  Issue Credit
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

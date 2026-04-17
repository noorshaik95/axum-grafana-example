'use client'

import { useState } from 'react'
import { useAuditLogs } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { formatDate } from '@/lib/utils'
import { Search, Loader2, FileText, Filter } from 'lucide-react'

const ACTION_COLORS: Record<string, string> = {
  create: 'bg-green-100 text-green-800',
  update: 'bg-blue-100 text-blue-800',
  delete: 'bg-red-100 text-red-800',
  login: 'bg-gray-100 text-gray-700',
  impersonate: 'bg-yellow-100 text-yellow-800',
  approve: 'bg-purple-100 text-purple-800',
  reject: 'bg-red-100 text-red-700',
  suspend: 'bg-orange-100 text-orange-800',
}

function actionColor(action: string) {
  const key = Object.keys(ACTION_COLORS).find((k) => action.toLowerCase().startsWith(k))
  return key ? ACTION_COLORS[key] : 'bg-gray-100 text-gray-600'
}

export default function AuditLogPage() {
  const [search, setSearch] = useState('')
  const [action, setAction] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [page, setPage] = useState(1)
  const [showFilters, setShowFilters] = useState(false)

  const { data, isLoading } = useAuditLogs({
    page,
    pageSize: 30,
    search: search || undefined,
    action: action || undefined,
    startDate: startDate || undefined,
    endDate: endDate || undefined,
  })

  const entries = data?.data ?? []
  const totalPages = data?.totalPages ?? 1

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Audit Log</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Full trail of administrative actions taken on this platform.
        </p>
      </div>

      {/* Search + filter bar */}
      <div className="flex gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-10"
            placeholder="Search by user, resource, or action…"
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <Button variant="outline" onClick={() => setShowFilters((v) => !v)}>
          <Filter className="mr-2 h-4 w-4" /> Filters
        </Button>
      </div>

      {showFilters && (
        <Card>
          <CardContent className="pt-4 grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label>Action Type</Label>
              <Input
                placeholder="e.g. create_tenant"
                value={action}
                onChange={(e) => {
                  setAction(e.target.value)
                  setPage(1)
                }}
              />
            </div>
            <div className="space-y-2">
              <Label>Start Date</Label>
              <Input
                type="date"
                value={startDate}
                onChange={(e) => {
                  setStartDate(e.target.value)
                  setPage(1)
                }}
              />
            </div>
            <div className="space-y-2">
              <Label>End Date</Label>
              <Input
                type="date"
                value={endDate}
                onChange={(e) => {
                  setEndDate(e.target.value)
                  setPage(1)
                }}
              />
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <FileText className="h-4 w-4" />
            Audit Entries
            {data?.total !== undefined && (
              <span className="ml-auto text-sm font-normal text-muted-foreground">
                {data.total.toLocaleString()} total
              </span>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex justify-center py-10">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : entries.length === 0 ? (
            <p className="py-10 text-center text-sm text-muted-foreground">
              No audit entries found
            </p>
          ) : (
            <div className="space-y-0 divide-y">
              {entries.map((entry) => (
                <div
                  key={entry.id}
                  className="flex items-start gap-4 py-3 hover:bg-muted/20 transition-colors"
                >
                  <div className="mt-0.5 flex-shrink-0">
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs font-medium ${actionColor(entry.action)}`}
                    >
                      {entry.action}
                    </span>
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm">
                      <span className="font-medium">{entry.userName}</span> acted on{' '}
                      <span className="font-medium">{entry.resource}</span>
                      {entry.resourceId && (
                        <span className="font-mono text-xs text-muted-foreground ml-1">
                          #{entry.resourceId.slice(0, 8)}
                        </span>
                      )}
                    </p>
                    {entry.details && (
                      <p className="text-xs text-muted-foreground mt-0.5">{entry.details}</p>
                    )}
                  </div>
                  <div className="flex-shrink-0 text-right">
                    <p className="text-xs text-muted-foreground">{formatDate(entry.timestamp)}</p>
                    <p className="text-xs text-muted-foreground">{entry.ipAddress}</p>
                  </div>
                </div>
              ))}
            </div>
          )}

          {totalPages > 1 && (
            <div className="flex items-center justify-between border-t pt-4 mt-2">
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

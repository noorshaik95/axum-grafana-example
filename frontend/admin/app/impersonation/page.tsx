'use client'

import { useMemo, useState } from 'react'
import { History, Loader2, LogIn, Search, UserCog, X, AlertTriangle } from 'lucide-react'
import {
  useAdminUsers,
  useImpersonationSessions,
  usePlatformTenants,
  useStartAdminImpersonation,
} from '@/lib/hooks/use-admin-queries'
import { EmptyStateIllustrated, StatusPill } from '../../../shared/components'
import { Spinner } from '../../../shared/components/ui'
import { formatDate } from '@/lib/utils'
import type { AdminUser } from '@/lib/api/platform'
import type { TenantSummary } from '@/lib/api/platform'

function StartImpersonationModal({ onClose }: { onClose: () => void }) {
  const [tenantId, setTenantId] = useState('')
  const [userId, setUserId] = useState('')
  const [userSearch, setUserSearch] = useState('')
  const [error, setError] = useState<string | null>(null)

  const tenantsQuery = usePlatformTenants()
  const usersQuery = useAdminUsers({
    tenant: tenantId || undefined,
    search: userSearch || undefined,
    pageSize: 25,
  })
  const impersonate = useStartAdminImpersonation()

  const tenants: TenantSummary[] = tenantsQuery.data?.tenants ?? []
  const users: AdminUser[] = usersQuery.data?.users ?? []

  const onStart = async () => {
    if (!tenantId || !userId) {
      setError('Pick a tenant and a target user.')
      return
    }
    setError(null)
    try {
      const res = await impersonate.mutateAsync({ tenantId, userId })
      window.location.href = res.redirect_url
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Impersonation failed')
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-warm-900/40 p-4">
      <div className="w-full max-w-lg space-y-5 rounded-2xl bg-white p-6 shadow-xl">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2">
            <UserCog className="h-5 w-5 text-warm-700" />
            <h2 className="font-display text-lg font-bold text-warm-900">Start impersonation</h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="rounded-full p-1 text-warm-700 hover:bg-warm-50"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-700" />
          <p className="text-xs text-warm-700">
            This action grants full access to the target user&apos;s account and is audit-logged.
            Only impersonate for legitimate support.
          </p>
        </div>

        <div className="space-y-2">
          <label className="text-xs font-medium uppercase tracking-wide text-warm-700">
            Tenant
          </label>
          <select
            className="w-full rounded-lg border border-warm-200 bg-white px-3 py-2 text-sm text-warm-900 focus:border-forest-500 focus:outline-none"
            value={tenantId}
            onChange={(e) => {
              setTenantId(e.target.value)
              setUserId('')
            }}
          >
            <option value="">
              {tenantsQuery.isLoading ? 'Loading tenants…' : 'Select tenant…'}
            </option>
            {tenants.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name} ({t.slug})
              </option>
            ))}
          </select>
          {tenantsQuery.isError ? (
            <p className="text-xs text-red-600">
              Could not load tenants — paste tenant ID below instead.
            </p>
          ) : null}
          <input
            type="text"
            placeholder="…or paste tenant UUID"
            value={tenantId}
            onChange={(e) => setTenantId(e.target.value)}
            className="w-full rounded-lg border border-warm-200 bg-white px-3 py-2 font-mono text-xs text-warm-900 focus:border-forest-500 focus:outline-none"
          />
        </div>

        <div className="space-y-2">
          <label className="text-xs font-medium uppercase tracking-wide text-warm-700">
            Target user
          </label>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-warm-500" />
            <input
              type="text"
              placeholder="Filter users by name or email…"
              value={userSearch}
              onChange={(e) => setUserSearch(e.target.value)}
              disabled={!tenantId}
              className="w-full rounded-lg border border-warm-200 bg-white py-2 pl-10 pr-3 text-sm text-warm-900 placeholder:text-warm-500 focus:border-forest-500 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            />
          </div>
          <select
            className="w-full rounded-lg border border-warm-200 bg-white px-3 py-2 text-sm text-warm-900 focus:border-forest-500 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            value={userId}
            onChange={(e) => setUserId(e.target.value)}
            disabled={!tenantId || usersQuery.isLoading}
          >
            <option value="">
              {!tenantId
                ? 'Pick a tenant first'
                : usersQuery.isLoading
                  ? 'Loading users…'
                  : 'Select user…'}
            </option>
            {users.map((u) => (
              <option key={u.id} value={u.id}>
                {u.first_name} {u.last_name} — {u.email}
              </option>
            ))}
          </select>
          <input
            type="text"
            placeholder="…or paste user UUID"
            value={userId}
            onChange={(e) => setUserId(e.target.value)}
            className="w-full rounded-lg border border-warm-200 bg-white px-3 py-2 font-mono text-xs text-warm-900 focus:border-forest-500 focus:outline-none"
          />
        </div>

        {error ? <p className="text-sm text-red-600">{error}</p> : null}

        <div className="flex items-center justify-end gap-2 pt-2">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-warm-200 bg-white px-4 py-2 text-sm font-medium text-warm-900 hover:bg-warm-50"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onStart}
            disabled={!tenantId || !userId || impersonate.isPending}
            className="inline-flex items-center gap-2 rounded-lg bg-amber-500 px-4 py-2 text-sm font-medium text-warm-900 transition-colors hover:bg-amber-400 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {impersonate.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <LogIn className="h-4 w-4" />
            )}
            Start session
          </button>
        </div>
      </div>
    </div>
  )
}

export default function ImpersonationPage() {
  const [showModal, setShowModal] = useState(false)
  const [search, setSearch] = useState('')

  const sessionsQuery = useImpersonationSessions({ page: 1, pageSize: 30 })

  const sessions = useMemo(() => {
    const rows = sessionsQuery.data?.data ?? []
    if (!search) return rows
    const needle = search.toLowerCase()
    return rows.filter(
      (s) =>
        s.targetUserName.toLowerCase().includes(needle) ||
        s.targetTenantName.toLowerCase().includes(needle)
    )
  }, [sessionsQuery.data?.data, search])

  return (
    <>
      <div className="space-y-6">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="font-display text-2xl font-bold text-warm-900">Impersonation</h1>
            <p className="mt-1 text-sm text-warm-700">
              Impersonate a tenant user for support. All sessions are audit-logged; the impersonated
              portal manages the session token and teardown.
            </p>
          </div>
          <button
            type="button"
            onClick={() => setShowModal(true)}
            className="inline-flex items-center gap-2 rounded-lg bg-forest-700 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-forest-800"
          >
            <LogIn className="h-4 w-4" />
            Start impersonation
          </button>
        </div>

        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-warm-500" />
          <input
            type="text"
            placeholder="Search sessions by user or tenant…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-warm-200 bg-white py-2 pl-10 pr-4 text-sm text-warm-900 placeholder:text-warm-500 focus:border-forest-500 focus:outline-none"
          />
        </div>

        <div className="overflow-hidden rounded-xl border border-warm-200 bg-white">
          <div className="flex items-center gap-2 border-b border-warm-200 bg-warm-50 px-5 py-3">
            <History className="h-4 w-4 text-warm-700" />
            <p className="text-sm font-semibold text-warm-900">Session history</p>
          </div>
          {sessionsQuery.isLoading ? (
            <div className="flex justify-center py-10">
              <Spinner size="sm" />
            </div>
          ) : sessions.length === 0 ? (
            <div className="p-6">
              <EmptyStateIllustrated
                illustration="empty-inbox"
                title="No session history"
                description="Session listing is not yet wired on the backend — start an impersonation above to test the flow."
              />
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-warm-200 bg-warm-50/60">
                  <th className="px-5 py-2 text-left text-xs font-medium uppercase text-warm-700">
                    Admin
                  </th>
                  <th className="px-5 py-2 text-left text-xs font-medium uppercase text-warm-700">
                    Target user
                  </th>
                  <th className="px-5 py-2 text-left text-xs font-medium uppercase text-warm-700">
                    Tenant
                  </th>
                  <th className="px-5 py-2 text-left text-xs font-medium uppercase text-warm-700">
                    Reason
                  </th>
                  <th className="px-5 py-2 text-left text-xs font-medium uppercase text-warm-700">
                    Started
                  </th>
                  <th className="px-5 py-2 text-left text-xs font-medium uppercase text-warm-700">
                    Status
                  </th>
                </tr>
              </thead>
              <tbody>
                {sessions.map((s) => (
                  <tr
                    key={s.id}
                    className="border-b border-warm-200 last:border-b-0 hover:bg-warm-50/60"
                  >
                    <td className="px-5 py-3 text-warm-900">{s.adminUserName}</td>
                    <td className="px-5 py-3 font-medium text-warm-900">{s.targetUserName}</td>
                    <td className="px-5 py-3 text-warm-700">{s.targetTenantName}</td>
                    <td className="max-w-[240px] truncate px-5 py-3 text-warm-700">{s.reason}</td>
                    <td className="px-5 py-3 text-warm-700">{formatDate(s.startedAt)}</td>
                    <td className="px-5 py-3">
                      <StatusPill
                        tone={s.endedAt ? 'gray' : 'amber'}
                        label={s.endedAt ? 'ended' : 'active'}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {showModal ? <StartImpersonationModal onClose={() => setShowModal(false)} /> : null}
    </>
  )
}

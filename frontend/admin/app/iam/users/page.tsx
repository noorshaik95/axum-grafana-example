'use client'

import { useState } from 'react'
import {
  Plus,
  Search,
  Edit,
  Shield,
  CheckCircle2,
  XCircle,
  Loader2,
  AlertCircle,
  ChevronLeft,
  ChevronRight,
  Users,
} from 'lucide-react'
import { useUsers } from '../../../../shared/lib/api/hooks'
import type { User, Role } from '../../../../shared/lib/api/types'

const ROLE_COLORS: Record<string, string> = {
  admin: 'bg-red-50 text-red-600',
  superadmin: 'bg-red-50 text-red-700',
  instructor: 'bg-blue-50 text-blue-600',
  student: 'bg-gray-100 text-gray-600',
  user: 'bg-gray-100 text-gray-500',
  manager: 'bg-amber-50 text-amber-600',
}

function RoleBadge({ role }: { role: Role }) {
  const colors = ROLE_COLORS[role.name] ?? 'bg-gray-100 text-gray-600'
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${colors}`}
    >
      <Shield className="h-3 w-3" />
      {role.name}
    </span>
  )
}

function SkeletonRow() {
  return (
    <tr className="border-b border-[var(--color-border)]">
      {Array.from({ length: 6 }).map((_, i) => (
        <td key={i} className="px-5 py-3">
          <div className="h-4 w-24 animate-pulse rounded bg-gray-200" />
        </td>
      ))}
    </tr>
  )
}

export default function IAMUsersPage() {
  const [searchTerm, setSearchTerm] = useState('')
  const [page, setPage] = useState(1)
  const PAGE_SIZE = 20

  const { data, isLoading, isError, error } = useUsers({
    search: searchTerm || undefined,
    page,
    pageSize: PAGE_SIZE,
  })

  const users: readonly User[] = data?.data ?? []
  const totalUsers = data?.pagination?.totalItems ?? 0
  const totalPages = data?.pagination?.totalPages ?? 1
  const activeCount = users.filter((u) => u.isActive).length
  const inactiveCount = users.filter((u) => !u.isActive).length

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-[var(--color-text)]">Users</h1>
          <p className="mt-1 text-sm text-[var(--color-text-muted)]">
            Manage all users in the Slate LMS platform.
          </p>
        </div>
        <button className="inline-flex items-center gap-2 rounded-lg bg-[var(--color-primary)] px-4 py-2.5 text-sm font-medium text-white hover:bg-[var(--color-primary-hover)] transition-colors">
          <Plus className="h-4 w-4" />
          Invite User
        </button>
      </div>

      {/* Stat cards */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
          <p className="text-sm font-medium text-[var(--color-text-muted)]">Total Users</p>
          <p className="mt-1 text-2xl font-bold text-[var(--color-text)]">
            {isLoading ? <Loader2 className="h-6 w-6 animate-spin" /> : totalUsers}
          </p>
        </div>
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
          <p className="text-sm font-medium text-[var(--color-text-muted)]">Active (this page)</p>
          <p className="mt-1 text-2xl font-bold text-green-600">{isLoading ? '--' : activeCount}</p>
        </div>
        <div className="rounded-xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
          <p className="text-sm font-medium text-[var(--color-text-muted)]">Inactive (this page)</p>
          <p className="mt-1 text-2xl font-bold text-[var(--color-text-muted)]">
            {isLoading ? '--' : inactiveCount}
          </p>
        </div>
      </div>

      {/* Table card */}
      <div className="rounded-xl border border-[var(--color-border)] bg-white shadow-sm">
        {/* Toolbar */}
        <div className="flex flex-col gap-3 p-5 sm:flex-row sm:items-center sm:justify-between">
          <div className="relative w-full max-w-sm">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--color-text-muted)]" />
            <input
              type="text"
              placeholder="Search users..."
              value={searchTerm}
              onChange={(e) => {
                setSearchTerm(e.target.value)
                setPage(1)
              }}
              className="w-full rounded-lg border border-[var(--color-border)] bg-white py-2 pl-10 pr-4 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-muted)] focus:border-[var(--color-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--color-primary)]"
            />
          </div>
        </div>

        {/* Table */}
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-t border-[var(--color-border)] bg-[var(--color-bg-muted)]">
                <th className="px-5 py-3 text-left text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                  Name
                </th>
                <th className="px-5 py-3 text-left text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                  Email
                </th>
                <th className="px-5 py-3 text-left text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                  Role
                </th>
                <th className="px-5 py-3 text-left text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                  Status
                </th>
                <th className="px-5 py-3 text-left text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                  Auth Method
                </th>
                <th className="px-5 py-3 text-left text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                  Joined
                </th>
                <th className="px-5 py-3 text-right text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                Array.from({ length: 5 }).map((_, i) => <SkeletonRow key={i} />)
              ) : isError ? (
                <tr>
                  <td colSpan={7} className="px-5 py-12 text-center">
                    <div className="flex items-center justify-center gap-2 text-[var(--color-error)]">
                      <AlertCircle className="h-5 w-5" />
                      <span className="text-sm">
                        Failed to load users
                        {error instanceof Error ? `: ${error.message}` : ''}
                      </span>
                    </div>
                  </td>
                </tr>
              ) : users.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-5 py-12 text-center">
                    <Users className="mx-auto h-8 w-8 text-[var(--color-text-muted)]" />
                    <p className="mt-2 text-sm text-[var(--color-text-muted)]">No users found.</p>
                  </td>
                </tr>
              ) : (
                users.map((user) => (
                  <tr
                    key={user.id}
                    className="border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-bg-muted)]/50 transition-colors"
                  >
                    <td className="px-5 py-3">
                      <div className="flex items-center gap-3">
                        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-indigo-100 text-xs font-medium text-indigo-600">
                          {user.firstName.charAt(0)}
                          {user.lastName.charAt(0)}
                        </div>
                        <span className="text-sm font-medium text-[var(--color-text)]">
                          {user.firstName} {user.lastName}
                        </span>
                      </div>
                    </td>
                    <td className="px-5 py-3 text-sm text-[var(--color-text-muted)]">
                      {user.email}
                    </td>
                    <td className="px-5 py-3">
                      <div className="flex flex-wrap gap-1">
                        {user.roles.map((role) => (
                          <RoleBadge key={role.id} role={role} />
                        ))}
                      </div>
                    </td>
                    <td className="px-5 py-3">
                      {user.isActive ? (
                        <span className="inline-flex items-center gap-1 rounded-full bg-green-50 px-2 py-0.5 text-xs font-medium text-green-600">
                          <CheckCircle2 className="h-3 w-3" />
                          Active
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-500">
                          <XCircle className="h-3 w-3" />
                          Inactive
                        </span>
                      )}
                    </td>
                    <td className="px-5 py-3">
                      <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 capitalize">
                        {user.authMethod}
                      </span>
                    </td>
                    <td className="px-5 py-3 text-sm text-[var(--color-text-muted)]">
                      {new Date(user.createdAt).toLocaleDateString()}
                    </td>
                    <td className="px-5 py-3 text-right">
                      <button className="rounded-lg p-1.5 text-[var(--color-text-muted)] hover:bg-[var(--color-bg-muted)] hover:text-[var(--color-text)]">
                        <Edit className="h-4 w-4" />
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {!isLoading && totalPages > 1 && (
          <div className="flex items-center justify-between border-t border-[var(--color-border)] px-5 py-3">
            <span className="text-sm text-[var(--color-text-muted)]">
              Page {page} of {totalPages} ({totalUsers} total)
            </span>
            <div className="flex gap-2">
              <button
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
                className="inline-flex items-center gap-1 rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-sm font-medium text-[var(--color-text)] hover:bg-[var(--color-bg-muted)] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                <ChevronLeft className="h-4 w-4" />
                Previous
              </button>
              <button
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
                className="inline-flex items-center gap-1 rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-sm font-medium text-[var(--color-text)] hover:bg-[var(--color-bg-muted)] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                Next
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

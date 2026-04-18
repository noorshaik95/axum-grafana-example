'use client'

import { useParams } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeft, AlertTriangle, RefreshCw, ScrollText, UserCheck } from 'lucide-react'

const MOCK_USERS: Record<
  string,
  {
    name: string
    role: string
    school: string
    email: string
    lastActive: string
    actions: Array<{ desc: string; time: string }>
    permissions: string[]
  }
> = {
  u1: {
    name: 'Dr. Jennifer Park',
    role: 'Tenant Admin',
    school: 'Stanford University',
    email: 'jpark@stanford.edu',
    lastActive: '2 hours ago',
    actions: [
      { desc: 'Edited course "CS101 · Fall 2026"', time: '2h ago' },
      { desc: 'Invited 3 new faculty members', time: '5h ago' },
      { desc: 'Approved 12 student enrollments', time: '1d ago' },
      { desc: 'Updated SSO configuration', time: '3d ago' },
    ],
    permissions: ['courses:write', 'users:invite', 'billing:read', 'settings:write'],
  },
  u2: {
    name: 'Alex Torres',
    role: 'Platform Admin',
    school: 'Slate (Internal)',
    email: 'alex@slate.io',
    lastActive: '5 min ago',
    actions: [
      { desc: 'Resolved incident INC-001', time: '2h ago' },
      { desc: 'Provisioned new tenant: Yale', time: '1d ago' },
    ],
    permissions: ['platform:admin', 'tenants:write', 'incidents:manage', 'impersonation:allow'],
  },
}

const DEFAULT_USER = MOCK_USERS['u1']

export default function UserDetailPage() {
  const params = useParams()
  const id = typeof params.id === 'string' ? params.id : 'u1'
  const user = MOCK_USERS[id] ?? DEFAULT_USER

  const initials = user.name
    .split(' ')
    .map((n) => n[0])
    .join('')
    .slice(0, 2)
    .toUpperCase()

  return (
    <div className="max-w-xl space-y-6">
      <Link
        href="/iam/users"
        className="inline-flex items-center gap-1 text-sm text-[#6a6e62] hover:text-[#234e32]"
      >
        <ArrowLeft className="h-3.5 w-3.5" /> Back to users
      </Link>

      {/* Header */}
      <div className="flex items-center gap-4">
        <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-[#dde9df] text-lg font-bold text-[#234e32]">
          {initials}
        </div>
        <div>
          <h1 className="text-xl font-bold text-[#12170f]">{user.name}</h1>
          <div className="flex items-center gap-2 mt-0.5">
            <span className="text-xs px-2 py-0.5 rounded-full bg-[#234e32] text-white">
              {user.role}
            </span>
            <span className="text-xs text-[#6a6e62]">{user.school}</span>
          </div>
          <p className="text-xs text-[#6a6e62] mt-1">
            {user.email} · last active {user.lastActive}
          </p>
        </div>
      </div>

      {/* Recent actions */}
      <div className="rounded-xl border border-[#e4e0d4] bg-white p-5">
        <p className="text-sm font-semibold text-[#12170f] mb-3">Recent actions</p>
        <div className="space-y-2">
          {user.actions.map((a, i) => (
            <div key={i} className="flex items-start justify-between gap-4">
              <p className="text-sm text-[#12170f]">{a.desc}</p>
              <p className="text-xs text-[#6a6e62] shrink-0">{a.time}</p>
            </div>
          ))}
        </div>
      </div>

      {/* Permissions */}
      <div className="rounded-xl border border-[#e4e0d4] bg-white p-5">
        <p className="text-sm font-semibold text-[#12170f] mb-3">Permissions</p>
        <div className="flex flex-wrap gap-2">
          {user.permissions.map((perm) => (
            <span
              key={perm}
              className="px-2 py-0.5 rounded-md bg-[#f6f3ec] text-xs font-mono text-[#6a6e62] border border-[#e4e0d4]"
            >
              {perm}
            </span>
          ))}
        </div>
      </div>

      {/* Impersonate card */}
      <div className="rounded-xl border-2 border-[#d97757]/30 bg-[#fde8e0]/30 p-5">
        <div className="flex items-start gap-3">
          <AlertTriangle className="h-5 w-5 text-[#d97757] shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-semibold text-[#d97757]">Impersonation</p>
            <p className="text-xs text-[#6a6e62] mt-0.5 leading-relaxed">
              Impersonating a user grants full access to their account. This action is audit-logged
              and visible to platform admins.
            </p>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-3 flex-wrap">
        <button className="flex items-center gap-2 px-4 py-2 rounded-lg bg-[#d97757] text-white text-sm font-medium hover:bg-[#c4694a] transition-colors">
          <UserCheck className="h-4 w-4" />
          Impersonate
        </button>
        <button className="flex items-center gap-2 px-4 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm font-medium text-[#12170f] hover:bg-[#f6f3ec] transition-colors">
          <RefreshCw className="h-4 w-4" />
          Reset MFA
        </button>
        <Link
          href="/iam/audit"
          className="flex items-center gap-2 px-4 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm font-medium text-[#12170f] hover:bg-[#f6f3ec] transition-colors"
        >
          <ScrollText className="h-4 w-4" />
          Audit log
        </Link>
      </div>
    </div>
  )
}

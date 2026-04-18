'use client'

import { useParams } from 'next/navigation'
import Link from 'next/link'
import { AlertTriangle, ArrowLeft, CheckCircle, ExternalLink } from 'lucide-react'

const MOCK_SCHOOLS: Record<
  string,
  {
    name: string
    location: string
    plan: string
    seats: number
    mau: number
    courses: number
    nps: number
    color: string
    initials: string
    admin: string
    adminEmail: string
    incident: { title: string; priority: string } | null
    integrations: Array<{ name: string; status: string }>
    contract: { value: string; start: string; end: string }
  }
> = {
  '1': {
    name: 'Stanford University',
    location: 'Palo Alto, CA',
    plan: 'Enterprise',
    seats: 15000,
    mau: 9800,
    courses: 1240,
    nps: 72,
    color: '#8B0000',
    initials: 'SU',
    admin: 'Dr. Jennifer Park',
    adminEmail: 'jpark@stanford.edu',
    incident: { title: 'Grade sync backlog — 1,247 submissions', priority: 'P1' },
    integrations: [
      { name: 'Canvas LMS', status: 'degraded' },
      { name: 'SAML SSO', status: 'healthy' },
      { name: 'Zoom', status: 'healthy' },
    ],
    contract: { value: '$480,000/yr', start: 'Jan 2024', end: 'Dec 2026' },
  },
  '2': {
    name: 'MIT',
    location: 'Cambridge, MA',
    plan: 'Enterprise',
    seats: 12000,
    mau: 8900,
    courses: 980,
    nps: 81,
    color: '#A31F34',
    initials: 'MIT',
    admin: 'Prof. Alan Chen',
    adminEmail: 'achen@mit.edu',
    incident: null,
    integrations: [
      { name: 'Canvas LMS', status: 'healthy' },
      { name: 'SAML SSO', status: 'healthy' },
    ],
    contract: { value: '$420,000/yr', start: 'Feb 2024', end: 'Jan 2027' },
  },
}

const STATUS_COLORS: Record<string, { dot: string; label: string }> = {
  healthy: { dot: '#3e7d4f', label: 'Healthy' },
  degraded: { dot: '#d97757', label: 'Degraded' },
  down: { dot: '#ef4444', label: 'Down' },
}

export default function SchoolDetailPage() {
  const params = useParams()
  const id = typeof params.id === 'string' ? params.id : '1'
  const school = MOCK_SCHOOLS[id] ?? MOCK_SCHOOLS['1']

  return (
    <div className="space-y-6 max-w-3xl">
      <Link
        href="/schools"
        className="inline-flex items-center gap-1 text-sm text-[#6a6e62] hover:text-[#234e32]"
      >
        <ArrowLeft className="h-3.5 w-3.5" /> Back to schools
      </Link>

      {/* Header */}
      <div className="flex items-start gap-4">
        <div
          className="flex h-14 w-14 shrink-0 items-center justify-center rounded-xl text-white text-sm font-bold"
          style={{ backgroundColor: school.color }}
        >
          {school.initials}
        </div>
        <div>
          <h1
            className="text-2xl font-bold text-[#12170f]"
            style={{ fontFamily: 'Instrument Serif, Georgia, serif' }}
          >
            {school.name}
          </h1>
          <p className="text-sm text-[#6a6e62]">
            {school.location} · {school.plan} plan · {school.seats.toLocaleString()} seats
          </p>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4">
        {[
          { label: 'MAU', value: school.mau.toLocaleString() },
          { label: 'Courses', value: school.courses.toLocaleString() },
          { label: 'NPS', value: school.nps.toString() },
        ].map((s) => (
          <div
            key={s.label}
            className="rounded-xl border border-[#e4e0d4] bg-white p-4 text-center"
          >
            <p className="text-2xl font-bold text-[#12170f]">{s.value}</p>
            <p className="text-xs text-[#6a6e62] mt-1">{s.label}</p>
          </div>
        ))}
      </div>

      {/* Active incident */}
      {school.incident && (
        <div className="rounded-xl border-2 border-[#d97757]/40 bg-[#fde8e0]/50 p-4">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-[#d97757] shrink-0 mt-0.5" />
            <div className="flex-1">
              <p className="text-sm font-semibold text-[#d97757]">
                {school.incident.priority} Incident
              </p>
              <p className="text-sm text-[#12170f]">{school.incident.title}</p>
            </div>
            <Link
              href="/incidents/1"
              className="text-xs text-[#d97757] hover:underline flex items-center gap-1"
            >
              View <ExternalLink className="h-3 w-3" />
            </Link>
          </div>
        </div>
      )}

      {/* Info rows */}
      <div className="rounded-xl border border-[#e4e0d4] bg-white divide-y divide-[#e4e0d4]">
        <div className="px-5 py-3.5 flex items-center justify-between">
          <span className="text-xs font-medium text-[#6a6e62] uppercase tracking-wide">
            Primary admin
          </span>
          <span className="text-sm text-[#12170f]">
            {school.admin} <span className="text-[#6a6e62]">— {school.adminEmail}</span>
          </span>
        </div>
        <div className="px-5 py-3.5">
          <p className="text-xs font-medium text-[#6a6e62] uppercase tracking-wide mb-2">
            Integrations
          </p>
          <div className="flex items-center gap-3">
            {school.integrations.map((int) => {
              const sc = STATUS_COLORS[int.status] ?? STATUS_COLORS.healthy
              return (
                <div key={int.name} className="flex items-center gap-1.5">
                  <span className="h-2 w-2 rounded-full" style={{ backgroundColor: sc.dot }} />
                  <span className="text-sm text-[#12170f]">{int.name}</span>
                  <span className="text-xs text-[#6a6e62]">({sc.label})</span>
                </div>
              )
            })}
          </div>
        </div>
        <div className="px-5 py-3.5 flex items-center justify-between">
          <span className="text-xs font-medium text-[#6a6e62] uppercase tracking-wide">
            Contract
          </span>
          <span className="text-sm text-[#12170f]">
            {school.contract.value} · {school.contract.start} – {school.contract.end}
          </span>
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-3">
        <Link
          href="/incidents/new"
          className="px-4 py-2 rounded-lg bg-[#d97757] text-white text-sm font-medium hover:bg-[#c4694a] transition-colors"
        >
          Open incident
        </Link>
        <Link
          href="/iam/users"
          className="px-4 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm font-medium text-[#12170f] hover:bg-[#f6f3ec] transition-colors"
        >
          View users
        </Link>
        <button className="px-4 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm font-medium text-[#12170f] hover:bg-[#f6f3ec] transition-colors">
          Edit contract
        </button>
      </div>
    </div>
  )
}

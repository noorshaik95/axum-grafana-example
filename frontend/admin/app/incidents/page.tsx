'use client'

import Link from 'next/link'
import { AlertTriangle, CheckCircle, Eye } from 'lucide-react'
import { useState } from 'react'

type Incident = {
  id: string
  title: string
  school: string
  age: string
  priority: 'P1' | 'P2' | 'P3'
  status: 'open' | 'watch' | 'resolved'
}

const INCIDENTS: Incident[] = [
  {
    id: '1',
    title: 'Grade sync backlog — 1,247 submissions',
    school: 'Stanford',
    age: '2h 14m',
    priority: 'P1',
    status: 'open',
  },
  {
    id: '2',
    title: 'Canvas webhook timeout intermittent',
    school: 'Harvard',
    age: '6h 05m',
    priority: 'P3',
    status: 'watch',
  },
  {
    id: '3',
    title: 'Video CDN latency spike — EU region',
    school: 'Oxford',
    age: '18h 30m',
    priority: 'P3',
    status: 'watch',
  },
  {
    id: '4',
    title: 'Auth service response degraded',
    school: 'Platform',
    age: '2d',
    priority: 'P1',
    status: 'resolved',
  },
  {
    id: '5',
    title: 'Email delivery delay — welcome emails',
    school: 'MIT',
    age: '3d',
    priority: 'P3',
    status: 'resolved',
  },
  {
    id: '6',
    title: 'Search indexing lag > 5min',
    school: 'Berkeley',
    age: '5d',
    priority: 'P3',
    status: 'resolved',
  },
]

const TABS = ['open', 'watch', 'resolved'] as const

export default function IncidentsPage() {
  const [tab, setTab] = useState<(typeof TABS)[number]>('open')

  const filtered = INCIDENTS.filter((i) => i.status === tab)
  const counts = TABS.reduce(
    (acc, t) => ({ ...acc, [t]: INCIDENTS.filter((i) => i.status === t).length }),
    {} as Record<string, number>
  )

  return (
    <div className="space-y-6">
      <div>
        <h1
          className="text-2xl font-bold text-[#12170f]"
          style={{ fontFamily: 'Instrument Serif, Georgia, serif' }}
        >
          Incidents
        </h1>
        <p className="text-sm text-[#6a6e62] mt-1">Track and manage platform incidents</p>
      </div>

      {/* Segment tabs */}
      <div className="flex items-center gap-1 p-1 rounded-lg bg-[#f6f3ec] border border-[#e4e0d4] w-fit">
        {TABS.map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-1.5 rounded-md text-sm font-medium transition-colors capitalize ${
              tab === t
                ? 'bg-white text-[#12170f] shadow-sm'
                : 'text-[#6a6e62] hover:text-[#12170f]'
            }`}
          >
            {t} {counts[t] > 0 && <span className="ml-1 text-xs opacity-70">{counts[t]}</span>}
          </button>
        ))}
      </div>

      {/* Incident rows */}
      <div className="space-y-3">
        {filtered.length === 0 && (
          <div className="rounded-xl border border-[#e4e0d4] bg-white p-8 text-center text-sm text-[#6a6e62]">
            No {tab} incidents
          </div>
        )}
        {filtered.map((inc) => (
          <div
            key={inc.id}
            className={`rounded-xl border p-4 flex items-center gap-4 ${
              inc.status === 'open' && inc.priority === 'P1'
                ? 'border-[#d97757]/40 bg-[#fde8e0]/40'
                : inc.status === 'watch'
                  ? 'border-[#ffb648]/40 bg-[#fff8e6]/40'
                  : 'border-[#e4e0d4] bg-white'
            }`}
          >
            {inc.status === 'open' ? (
              <AlertTriangle className="h-5 w-5 text-[#d97757] shrink-0" />
            ) : inc.status === 'watch' ? (
              <Eye className="h-5 w-5 text-[#ffb648] shrink-0" />
            ) : (
              <CheckCircle className="h-5 w-5 text-[#3e7d4f] shrink-0" />
            )}
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span
                  className={`text-[10px] font-bold px-1.5 py-0.5 rounded ${
                    inc.priority === 'P1'
                      ? 'bg-[#d97757] text-white'
                      : 'bg-[#ffb648]/30 text-[#856404]'
                  }`}
                >
                  {inc.priority}
                </span>
                <p className="text-sm font-medium text-[#12170f] truncate">{inc.title}</p>
              </div>
              <p className="text-xs text-[#6a6e62] mt-0.5">
                {inc.school} · {inc.age} ago
              </p>
            </div>
            <div className="shrink-0">
              {inc.status === 'open' ? (
                <Link
                  href={`/incidents/${inc.id}`}
                  className="px-3 py-1.5 rounded-lg bg-[#d97757] text-white text-xs font-medium hover:bg-[#c4694a] transition-colors"
                >
                  Open
                </Link>
              ) : inc.status === 'watch' ? (
                <span className="px-3 py-1.5 rounded-lg bg-[#ffb648]/20 text-[#856404] text-xs font-medium border border-[#ffb648]/40">
                  watch
                </span>
              ) : (
                <span className="px-3 py-1.5 rounded-lg bg-[#dde9df] text-[#234e32] text-xs font-medium">
                  done
                </span>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

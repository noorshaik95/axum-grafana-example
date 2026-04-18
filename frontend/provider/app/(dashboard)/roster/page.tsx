'use client'

import Link from 'next/link'
import { AlertTriangle, TrendingDown, CheckCircle2 } from 'lucide-react'

type Risk = 'at-risk' | 'slipping' | 'healthy'

interface Student {
  id: string
  name: string
  risk: Risk
  grade: string
  detail: string
  trend?: string
  missed?: number
}

const students: Student[] = [
  {
    id: 'stu-001',
    name: 'Alex Morrison',
    risk: 'at-risk',
    grade: '54%',
    detail: 'Missed PS2, PS3, PS4',
    missed: 3,
  },
  {
    id: 'stu-002',
    name: 'Jordan Kim',
    risk: 'at-risk',
    grade: '61%',
    detail: 'Missed PS3, no lecture attendance',
    missed: 2,
  },
  {
    id: 'stu-003',
    name: 'Sam Torres',
    risk: 'slipping',
    grade: '72%',
    detail: 'Grade dropped 8pts since PS2',
    trend: '−8pts',
  },
  {
    id: 'stu-004',
    name: 'Riley Park',
    risk: 'slipping',
    grade: '74%',
    detail: 'Late submissions trending',
    trend: 'late ×3',
  },
  {
    id: 'stu-005',
    name: 'Casey Walsh',
    risk: 'healthy',
    grade: '91%',
    detail: 'Consistent performer',
  },
  {
    id: 'stu-006',
    name: 'Morgan Lee',
    risk: 'healthy',
    grade: '88%',
    detail: 'Strong engagement',
  },
  {
    id: 'stu-007',
    name: 'Drew Harris',
    risk: 'healthy',
    grade: '85%',
    detail: 'Regular OH attendance',
  },
]

const riskConfig: Record<
  Risk,
  { avatarBg: string; avatarText: string; label: string; labelBg: string; labelColor: string }
> = {
  'at-risk': {
    avatarBg: 'rgba(217,119,87,0.15)',
    avatarText: '#d97757',
    label: 'at risk',
    labelBg: 'rgba(217,119,87,0.15)',
    labelColor: '#d97757',
  },
  slipping: {
    avatarBg: 'rgba(255,182,72,0.15)',
    avatarText: '#c48d1a',
    label: 'slipping',
    labelBg: 'rgba(255,182,72,0.15)',
    labelColor: '#c48d1a',
  },
  healthy: {
    avatarBg: '#f2f7f3',
    avatarText: '#3e7d4f',
    label: 'healthy',
    labelBg: '#dde9df',
    labelColor: '#234e32',
  },
}

export default function RosterPage() {
  const atRiskCount = students.filter((s) => s.risk === 'at-risk').length
  const slippingCount = students.filter((s) => s.risk === 'slipping').length

  return (
    <div className="space-y-5">
      {/* Header */}
      <div>
        <h1 className="font-serif text-2xl text-[#12170f]">
          {students.length} students · {atRiskCount} need you
        </h1>
        <p className="text-sm mt-1" style={{ color: '#6a6e62' }}>
          Sorted by risk — green healthy, amber slipping, rust at risk.
        </p>
      </div>

      {/* Risk summary */}
      <div className="flex items-center gap-3">
        {[
          {
            label: `${atRiskCount} at risk`,
            bg: 'rgba(217,119,87,0.12)',
            color: '#d97757',
            Icon: AlertTriangle,
          },
          {
            label: `${slippingCount} slipping`,
            bg: 'rgba(255,182,72,0.12)',
            color: '#c48d1a',
            Icon: TrendingDown,
          },
          {
            label: `${students.length - atRiskCount - slippingCount} healthy`,
            bg: '#f2f7f3',
            color: '#3e7d4f',
            Icon: CheckCircle2,
          },
        ].map((item) => (
          <span
            key={item.label}
            className="flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium"
            style={{ background: item.bg, color: item.color }}
          >
            <item.Icon className="h-3 w-3" />
            {item.label}
          </span>
        ))}
      </div>

      {/* Student list */}
      <div className="space-y-2">
        {students.map((student) => {
          const cfg = riskConfig[student.risk]
          const isAtRisk = student.risk === 'at-risk'
          const isSlipping = student.risk === 'slipping'
          return (
            <Link
              key={student.id}
              href={`/roster/${student.id}`}
              className="flex items-center gap-4 rounded-xl border px-4 py-3.5 group transition-all hover:shadow-sm"
              style={{
                background: isAtRisk
                  ? 'rgba(217,119,87,0.04)'
                  : isSlipping
                    ? 'rgba(255,182,72,0.04)'
                    : '#ffffff',
                borderColor: isAtRisk
                  ? 'rgba(217,119,87,0.3)'
                  : isSlipping
                    ? 'rgba(255,182,72,0.3)'
                    : '#e4e0d4',
              }}
            >
              {/* Avatar */}
              <div
                className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-sm font-semibold"
                style={{ background: cfg.avatarBg, color: cfg.avatarText }}
              >
                {student.name
                  .split(' ')
                  .map((n) => n[0])
                  .join('')}
              </div>

              {/* Info */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-sm text-[#12170f]">{student.name}</span>
                  <span
                    className="rounded-full px-2 py-0.5 text-xs font-mono"
                    style={{ background: cfg.labelBg, color: cfg.labelColor }}
                  >
                    {cfg.label}
                  </span>
                  {student.trend && (
                    <span className="text-xs font-mono" style={{ color: '#c48d1a' }}>
                      {student.trend}
                    </span>
                  )}
                </div>
                <p className="text-xs mt-0.5" style={{ color: '#6a6e62' }}>
                  {student.detail}
                </p>
              </div>

              {/* Grade */}
              <span
                className="font-serif text-lg font-bold shrink-0"
                style={{
                  color: isAtRisk ? '#d97757' : isSlipping ? '#c48d1a' : '#234e32',
                }}
              >
                {student.grade}
              </span>

              {/* Action button */}
              {isAtRisk && (
                <button
                  className="shrink-0 rounded-xl px-3 py-1.5 text-xs font-semibold transition-all hover:opacity-80"
                  style={{ background: '#d97757', color: '#ffffff' }}
                  onClick={(e) => {
                    e.preventDefault()
                    window.location.href = `/roster/${student.id}`
                  }}
                >
                  Reach out
                </button>
              )}
              {isSlipping && (
                <button
                  className="shrink-0 rounded-xl border px-3 py-1.5 text-xs font-semibold transition-all hover:bg-[#f6f3ec]"
                  style={{ borderColor: '#e4e0d4', color: '#12170f' }}
                  onClick={(e) => {
                    e.preventDefault()
                    window.location.href = `/roster/${student.id}`
                  }}
                >
                  Check in
                </button>
              )}
            </Link>
          )
        })}
      </div>
    </div>
  )
}

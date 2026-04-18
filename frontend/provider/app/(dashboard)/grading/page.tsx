'use client'

import { useState } from 'react'
import Link from 'next/link'
import { Search, Settings2 } from 'lucide-react'

type Risk = 'at-risk' | 'slipping' | 'healthy'

interface StudentRow {
  id: string
  name: string
  risk: Risk
  hw1: number | null
  hw2: number | null
  hw3: number | null
  ps4: number | null
  avg: number | null
}

const students: StudentRow[] = [
  {
    id: 'stu-001',
    name: 'Alex Morrison',
    risk: 'at-risk',
    hw1: 72,
    hw2: 65,
    hw3: null,
    ps4: null,
    avg: 54,
  },
  {
    id: 'stu-002',
    name: 'Jordan Kim',
    risk: 'at-risk',
    hw1: 80,
    hw2: 78,
    hw3: 72,
    ps4: null,
    avg: 61,
  },
  {
    id: 'stu-003',
    name: 'Sam Torres',
    risk: 'slipping',
    hw1: 88,
    hw2: 82,
    hw3: 75,
    ps4: null,
    avg: 72,
  },
  {
    id: 'stu-004',
    name: 'Riley Park',
    risk: 'slipping',
    hw1: 85,
    hw2: 80,
    hw3: 78,
    ps4: null,
    avg: 74,
  },
  {
    id: 'stu-005',
    name: 'Casey Walsh',
    risk: 'healthy',
    hw1: 95,
    hw2: 92,
    hw3: 90,
    ps4: 88,
    avg: 91,
  },
  {
    id: 'stu-006',
    name: 'Morgan Lee',
    risk: 'healthy',
    hw1: 91,
    hw2: 88,
    hw3: 85,
    ps4: 84,
    avg: 88,
  },
  {
    id: 'stu-007',
    name: 'Drew Harris',
    risk: 'healthy',
    hw1: 88,
    hw2: 85,
    hw3: 82,
    ps4: 80,
    avg: 85,
  },
  {
    id: 'stu-008',
    name: 'Avery Chen',
    risk: 'healthy',
    hw1: 90,
    hw2: 87,
    hw3: 84,
    ps4: 82,
    avg: 86,
  },
  {
    id: 'stu-009',
    name: 'Blake Singh',
    risk: 'healthy',
    hw1: 78,
    hw2: 80,
    hw3: 79,
    ps4: 77,
    avg: 79,
  },
  {
    id: 'stu-010',
    name: 'Charlie Davis',
    risk: 'healthy',
    hw1: 82,
    hw2: 79,
    hw3: 81,
    ps4: 78,
    avg: 80,
  },
]

const riskRowStyle: Record<Risk, { bg: string; border: string }> = {
  'at-risk': { bg: 'rgba(217,119,87,0.06)', border: 'rgba(217,119,87,0.25)' },
  slipping: { bg: 'rgba(255,182,72,0.06)', border: 'rgba(255,182,72,0.25)' },
  healthy: { bg: 'transparent', border: 'transparent' },
}

const riskBadge: Record<Risk, { label: string; bg: string; color: string }> = {
  'at-risk': { label: 'risk', bg: 'rgba(217,119,87,0.15)', color: '#d97757' },
  slipping: { label: 'slip', bg: 'rgba(255,182,72,0.15)', color: '#c48d1a' },
  healthy: { label: '', bg: '', color: '' },
}

function GradeCell({ value }: { value: number | null }) {
  if (value === null) {
    return (
      <td className="px-4 py-3 text-center">
        <span className="font-mono text-xs" style={{ color: '#e4e0d4' }}>
          —
        </span>
      </td>
    )
  }
  const color =
    value >= 85 ? '#3e7d4f' : value >= 70 ? '#12170f' : value >= 60 ? '#c48d1a' : '#d97757'
  return (
    <td className="px-4 py-3 text-center">
      <span className="font-mono text-sm font-medium" style={{ color }}>
        {value}
      </span>
    </td>
  )
}

export default function GradingPage() {
  const [filter, setFilter] = useState('')

  const filtered = filter
    ? students.filter((s) => s.name.toLowerCase().includes(filter.toLowerCase()))
    : students

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="font-serif text-2xl text-[#12170f]">Gradebook</h1>
          <p className="text-sm mt-1" style={{ color: '#6a6e62' }}>
            CS 3110 · Spring 2026 · 47 students
          </p>
        </div>
        <Link
          href="/grading/rules"
          className="flex items-center gap-2 rounded-xl border px-4 py-2 text-sm font-medium transition-colors hover:bg-[#f6f3ec]"
          style={{ borderColor: '#e4e0d4', color: '#12170f' }}
        >
          <Settings2 className="h-4 w-4" />
          Grading rules
        </Link>
      </div>

      {/* Filter bar */}
      <div
        className="flex items-center gap-2 rounded-xl border px-4 py-2.5"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <Search className="h-4 w-4 shrink-0" style={{ color: '#6a6e62' }} />
        <input
          type="text"
          placeholder="Filter students... (⌘F)"
          className="flex-1 text-sm bg-transparent outline-none"
          style={{ color: '#12170f' }}
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
        <span className="font-mono text-xs" style={{ color: '#6a6e62' }}>
          {filtered.length} / {students.length}
        </span>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-3 text-xs">
        {[
          { label: 'At risk', bg: 'rgba(217,119,87,0.12)', color: '#d97757' },
          { label: 'Slipping', bg: 'rgba(255,182,72,0.12)', color: '#c48d1a' },
          { label: 'Missing grade', color: '#e4e0d4', bg: '#f6f3ec' },
        ].map((item) => (
          <span
            key={item.label}
            className="flex items-center gap-1.5 rounded-full px-2.5 py-1"
            style={{ background: item.bg, color: item.color }}
          >
            {item.label}
          </span>
        ))}
      </div>

      {/* Gradebook table */}
      <div
        className="rounded-xl border overflow-hidden overflow-x-auto"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <table className="w-full min-w-[640px]">
          <thead>
            <tr style={{ borderBottom: '1px solid #e4e0d4', background: '#f6f3ec' }}>
              <th
                className="px-4 py-3 text-left text-xs font-mono font-medium"
                style={{ color: '#6a6e62' }}
              >
                STUDENT
              </th>
              {['HW1', 'HW2', 'HW3', 'PS4', 'AVG'].map((col) => (
                <th
                  key={col}
                  className="px-4 py-3 text-center text-xs font-mono font-medium"
                  style={{ color: '#6a6e62' }}
                >
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {filtered.map((student, i) => {
              const rowStyle = riskRowStyle[student.risk]
              const badge = riskBadge[student.risk]
              const isLast = i === filtered.length - 1
              return (
                <tr
                  key={student.id}
                  style={{
                    background: rowStyle.bg,
                    borderBottom: isLast ? 'none' : '1px solid #e4e0d4',
                  }}
                >
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <Link
                        href={`/roster/${student.id}`}
                        className="text-sm font-medium hover:underline"
                        style={{ color: '#12170f' }}
                      >
                        {student.name}
                      </Link>
                      {badge.label && (
                        <span
                          className="rounded-full px-1.5 py-0.5 text-xs font-mono"
                          style={{ background: badge.bg, color: badge.color }}
                        >
                          {badge.label}
                        </span>
                      )}
                    </div>
                  </td>
                  <GradeCell value={student.hw1} />
                  <GradeCell value={student.hw2} />
                  <GradeCell value={student.hw3} />
                  <GradeCell value={student.ps4} />
                  <td className="px-4 py-3 text-center">
                    {student.avg !== null ? (
                      <span
                        className="font-mono text-sm font-semibold"
                        style={{
                          color:
                            student.avg >= 85
                              ? '#234e32'
                              : student.avg >= 70
                                ? '#12170f'
                                : student.avg >= 60
                                  ? '#c48d1a'
                                  : '#d97757',
                        }}
                      >
                        {student.avg}%
                      </span>
                    ) : (
                      <span className="font-mono text-xs" style={{ color: '#e4e0d4' }}>
                        —
                      </span>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {/* Export link */}
      <div className="flex items-center justify-end">
        <Link
          href="/exports"
          className="text-sm font-medium transition-colors hover:text-[#234e32]"
          style={{ color: '#6a6e62' }}
        >
          Export grades →
        </Link>
      </div>
    </div>
  )
}

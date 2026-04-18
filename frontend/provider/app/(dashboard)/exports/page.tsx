'use client'

import { useState } from 'react'
import { Download, FileText, FileSpreadsheet, Archive, Loader2, Check } from 'lucide-react'

const exportItems = [
  {
    id: 'grade-roster',
    label: 'Final grade roster',
    description: 'All students, final grades, letter grades, GPA points',
    formats: ['CSV', 'PDF'],
    icon: FileText,
  },
  {
    id: 'midterm-report',
    label: 'Mid-term progress report',
    description: 'Current standing, trend analysis, at-risk flags per student',
    formats: ['PDF'],
    icon: FileText,
  },
  {
    id: 'ta-stats',
    label: 'TA grading statistics',
    description: 'Per-TA grade distributions, time-to-grade, consistency scores',
    formats: ['XLSX'],
    icon: FileSpreadsheet,
  },
  {
    id: 'eot-bundle',
    label: 'End-of-term bundle',
    description: 'All grades, feedback, submission archives, analytics summary',
    formats: ['ZIP'],
    icon: Archive,
  },
]

const formatColors: Record<string, { bg: string; text: string }> = {
  CSV: { bg: '#dde9df', text: '#234e32' },
  PDF: { bg: 'rgba(217,119,87,0.12)', text: '#c55a36' },
  XLSX: { bg: 'rgba(255,182,72,0.15)', text: '#c48d1a' },
  ZIP: { bg: '#f6f3ec', text: '#6a6e62' },
}

export default function ExportsPage() {
  const [exporting, setExporting] = useState<string | null>(null)
  const [exported, setExported] = useState<Set<string>>(new Set())

  const handleExport = async (id: string, format: string) => {
    const key = `${id}-${format}`
    setExporting(key)
    await new Promise((res) => setTimeout(res, 1500))
    setExporting(null)
    setExported((prev) => new Set([...prev, key]))
  }

  return (
    <div className="space-y-5 max-w-2xl">
      {/* Header */}
      <div>
        <h1 className="font-serif text-2xl text-[#12170f]">Export data</h1>
        <p className="text-sm mt-1" style={{ color: '#6a6e62' }}>
          Download grades, reports, and archives for your records.
        </p>
      </div>

      {/* Export rows */}
      <div
        className="rounded-xl border overflow-hidden"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <div className="divide-y" style={{ borderColor: '#e4e0d4' }}>
          {exportItems.map((item) => {
            const Icon = item.icon
            return (
              <div key={item.id} className="flex items-center gap-4 px-5 py-4">
                <div
                  className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg"
                  style={{ background: '#f2f7f3' }}
                >
                  <Icon className="h-5 w-5" style={{ color: '#234e32' }} />
                </div>

                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-[#12170f]">{item.label}</p>
                  <p className="text-xs mt-0.5 leading-snug" style={{ color: '#6a6e62' }}>
                    {item.description}
                  </p>
                </div>

                {/* Format buttons */}
                <div className="flex items-center gap-2 shrink-0">
                  {item.formats.map((fmt) => {
                    const key = `${item.id}-${fmt}`
                    const isExporting = exporting === key
                    const isDone = exported.has(key)
                    const colors = formatColors[fmt]
                    return (
                      <button
                        key={fmt}
                        onClick={() => handleExport(item.id, fmt)}
                        disabled={isExporting || isDone}
                        className="flex items-center gap-1.5 rounded-xl px-3 py-2 text-xs font-mono font-medium transition-all hover:opacity-80"
                        style={{
                          background: isDone ? '#dde9df' : colors.bg,
                          color: isDone ? '#234e32' : colors.text,
                        }}
                      >
                        {isExporting ? (
                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        ) : isDone ? (
                          <Check className="h-3.5 w-3.5" />
                        ) : (
                          <Download className="h-3.5 w-3.5" />
                        )}
                        {isDone ? 'Downloaded' : fmt}
                      </button>
                    )
                  })}
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* Note */}
      <p className="text-xs" style={{ color: '#6a6e62' }}>
        Exports are generated on demand and may take a few seconds. Files are available for 24
        hours.
      </p>
    </div>
  )
}

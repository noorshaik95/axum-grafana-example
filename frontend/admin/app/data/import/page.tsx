'use client'

import type { ReactNode } from 'react'
import { CheckCircle, Circle, PlayCircle } from 'lucide-react'

type StepStatus = 'done' | 'active' | 'pending'

const STEPS: Array<{ label: string; detail: string; status: StepStatus; progress?: number }> = [
  { label: 'Connect source · Canvas', detail: 'Authenticated via API key', status: 'done' },
  {
    label: 'Import courses · 2,147',
    detail: '1,089 of 2,147 imported',
    status: 'active',
    progress: 51,
  },
  { label: 'Reconcile rosters', detail: 'Waiting for courses to finish', status: 'pending' },
  { label: 'Verify & go live', detail: 'Final checks before cutover', status: 'pending' },
]

const STATUS_ICON: Record<StepStatus, ReactNode> = {
  done: <CheckCircle className="h-5 w-5 text-[#3e7d4f] shrink-0 mt-0.5" />,
  active: <PlayCircle className="h-5 w-5 text-[#ffb648] shrink-0 mt-0.5 animate-pulse" />,
  pending: <Circle className="h-5 w-5 text-[#e4e0d4] shrink-0 mt-0.5" />,
}

export default function DataImportPage() {
  return (
    <div className="max-w-xl space-y-8">
      {/* Header */}
      <div>
        <p className="text-xs font-medium text-[#6a6e62] uppercase tracking-wide mb-1">
          Migration · Stanford
        </p>
        <h1
          className="text-2xl font-bold text-[#12170f]"
          style={{ fontFamily: 'Instrument Serif, Georgia, serif' }}
        >
          Import data
        </h1>
        <div className="mt-3 flex items-start gap-2 rounded-lg bg-[#f2f7f3] border border-[#b8d2bd] px-4 py-3">
          <span className="text-[#3e7d4f] text-base shrink-0 mt-0.5">ℹ</span>
          <p className="text-sm text-[#234e32]">
            Import is non-destructive — Canvas stays intact until you cut over.
          </p>
        </div>
      </div>

      {/* Steps */}
      <div className="rounded-xl border border-[#e4e0d4] bg-white overflow-hidden">
        {STEPS.map((step, i) => (
          <div
            key={step.label}
            className={`px-5 py-4 flex items-start gap-4 ${
              i < STEPS.length - 1 ? 'border-b border-[#e4e0d4]' : ''
            } ${step.status === 'active' ? 'bg-[#fffbf0]' : ''}`}
          >
            {STATUS_ICON[step.status]}
            <div className="flex-1 min-w-0">
              <div className="flex items-center justify-between">
                <p
                  className={`text-sm font-medium ${
                    step.status === 'done'
                      ? 'text-[#6a6e62] line-through'
                      : step.status === 'active'
                        ? 'text-[#12170f]'
                        : 'text-[#9a9e92]'
                  }`}
                >
                  {step.label}
                </p>
                {step.status === 'done' && (
                  <span className="text-xs text-[#3e7d4f] font-medium">Done</span>
                )}
                {step.status === 'active' && (
                  <span className="text-xs text-[#ffb648] font-medium">In progress</span>
                )}
              </div>
              <p className="text-xs text-[#6a6e62] mt-0.5">{step.detail}</p>
              {step.status === 'active' && typeof step.progress === 'number' && (
                <div className="mt-2">
                  <div className="flex items-center justify-between text-xs text-[#6a6e62] mb-1">
                    <span>{step.progress}%</span>
                    <span>~8 min remaining</span>
                  </div>
                  <div className="h-1.5 rounded-full bg-[#e4e0d4] overflow-hidden">
                    <div
                      className="h-full rounded-full bg-[#ffb648] transition-all"
                      style={{ width: `${step.progress}%` }}
                    />
                  </div>
                </div>
              )}
            </div>
          </div>
        ))}
      </div>

      {/* Actions */}
      <div className="flex items-center gap-3">
        <button
          disabled
          className="px-5 py-2 rounded-lg bg-[#234e32] text-white text-sm font-medium disabled:opacity-40 disabled:cursor-not-allowed"
        >
          Continue to reconcile
        </button>
        <button className="px-4 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm font-medium text-[#6a6e62] hover:bg-[#f6f3ec] transition-colors">
          Cancel import
        </button>
      </div>
    </div>
  )
}

'use client'

import { useState } from 'react'

type Flag = {
  id: string
  name: string
  scope: string
  rollout: 'GA' | 'pilot' | 'rollout' | 'off'
  enabled: boolean
}

const INITIAL_FLAGS: Flag[] = [
  {
    id: 'new-grading-queue',
    name: 'New grading queue',
    scope: 'Replaces legacy Kafka consumer — all tenants',
    rollout: 'rollout',
    enabled: true,
  },
  {
    id: 'ai-draft-feedback',
    name: 'AI draft feedback',
    scope: 'AI-generated feedback on student drafts',
    rollout: 'pilot',
    enabled: true,
  },
  {
    id: 'study-plan-v2',
    name: 'Study plan v2',
    scope: 'Redesigned adaptive study plan UI',
    rollout: 'pilot',
    enabled: false,
  },
  {
    id: 'live-class-pulse',
    name: 'Live class pulse',
    scope: 'Real-time engagement metrics during video class',
    rollout: 'off',
    enabled: false,
  },
  {
    id: 'mobile-push',
    name: 'Mobile push',
    scope: 'Push notifications via Expo for iOS & Android',
    rollout: 'GA',
    enabled: true,
  },
]

const ROLLOUT_BADGE: Record<Flag['rollout'], { label: string; cls: string }> = {
  GA: { label: 'GA', cls: 'bg-[#dde9df] text-[#234e32]' },
  pilot: { label: 'pilot', cls: 'bg-[#fff3cd] text-[#856404]' },
  rollout: { label: 'rollout', cls: 'bg-[#dbeafe] text-[#1d4ed8]' },
  off: { label: 'off', cls: 'bg-[#f6f3ec] text-[#6a6e62]' },
}

export default function FlagsPage() {
  const [flags, setFlags] = useState(INITIAL_FLAGS)

  const toggle = (id: string) => {
    setFlags((prev) => prev.map((f) => (f.id === id ? { ...f, enabled: !f.enabled } : f)))
  }

  return (
    <div className="space-y-6">
      <div>
        <h1
          className="text-2xl font-bold text-[#12170f]"
          style={{ fontFamily: 'Instrument Serif, Georgia, serif' }}
        >
          Feature Flags
        </h1>
        <p className="text-sm text-[#6a6e62] mt-1">Control feature rollout across the platform</p>
      </div>

      <div className="rounded-xl border border-[#e4e0d4] bg-white overflow-hidden">
        {flags.map((flag, i) => {
          const badge = ROLLOUT_BADGE[flag.rollout]
          return (
            <div
              key={flag.id}
              className={`flex items-center gap-4 px-5 py-4 ${i < flags.length - 1 ? 'border-b border-[#e4e0d4]' : ''}`}
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium text-[#12170f]">{flag.name}</p>
                  <span
                    className={`text-[10px] font-medium px-1.5 py-0.5 rounded-full ${badge.cls}`}
                  >
                    {badge.label}
                  </span>
                </div>
                <p className="text-xs text-[#6a6e62] mt-0.5">{flag.scope}</p>
              </div>

              {/* Toggle switch */}
              <button
                onClick={() => toggle(flag.id)}
                className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors ${
                  flag.enabled ? 'bg-[#3e7d4f]' : 'bg-[#e4e0d4]'
                }`}
                role="switch"
                aria-checked={flag.enabled}
              >
                <span
                  className={`inline-block h-3.5 w-3.5 rounded-full bg-white shadow-sm transform transition-transform ${
                    flag.enabled ? 'translate-x-4.5' : 'translate-x-0.5'
                  }`}
                  style={{ transform: flag.enabled ? 'translateX(18px)' : 'translateX(2px)' }}
                />
              </button>
            </div>
          )
        })}
      </div>
    </div>
  )
}

'use client'

import * as React from 'react'
import { cn } from '../../utils/index'

export type NowBarTone = 'neutral' | 'amber' | 'red' | 'forest' | 'blue'

export interface NowBarChip {
  id?: string
  label: string
  tone?: NowBarTone
  /** Optional href — if provided, chip renders as an anchor. */
  href?: string
  onClick?: () => void
  onDismiss?: () => void
  /** Leading icon (already forest-contrast coloured by the chip). */
  icon?: React.ReactNode
}

export interface NowBarProps {
  chips: NowBarChip[]
  /** Optional left-side label describing the context (e.g. "Now"). */
  label?: string
  className?: string
  ariaLabel?: string
}

const toneClass: Record<NowBarTone, string> = {
  neutral: 'bg-forest-800/40 text-forest-50 ring-1 ring-inset ring-forest-600/40',
  forest: 'bg-forest-600 text-white ring-1 ring-inset ring-forest-400/60',
  amber: 'bg-amber-400/95 text-warm-900 ring-1 ring-inset ring-amber-500/40',
  red: 'bg-red-500/95 text-white ring-1 ring-inset ring-red-700/40',
  blue: 'bg-blue-500/95 text-white ring-1 ring-inset ring-blue-700/40',
}

/**
 * forest-700 contextual strip that surfaces the most urgent action chips
 * for the current section. Auto-hides when `chips` is empty so pages
 * without contextual work don't ship a bare bar.
 */
export function NowBar({ chips, label, className, ariaLabel = 'Current actions' }: NowBarProps) {
  if (!chips || chips.length === 0) return null

  return (
    <div
      role="region"
      aria-label={ariaLabel}
      data-testid="now-bar"
      className={cn(
        'w-full bg-forest-700 text-forest-50',
        'border-b border-forest-800/60',
        className
      )}
    >
      <div className="mx-auto flex max-w-[1400px] items-center gap-3 overflow-x-auto px-6 py-2">
        {label ? (
          <span className="shrink-0 text-[11px] font-semibold uppercase tracking-[0.16em] text-forest-200">
            {label}
          </span>
        ) : null}
        <ul className="flex items-center gap-2">
          {chips.map((chip, i) => {
            const key = chip.id ?? `${chip.label}-${i}`
            const tone = chip.tone ?? 'neutral'
            const chipClass = cn(
              'inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-3 py-1 text-xs font-medium',
              toneClass[tone]
            )
            const content = (
              <>
                {chip.icon ? (
                  <span aria-hidden="true" className="inline-flex text-current">
                    {chip.icon}
                  </span>
                ) : null}
                <span>{chip.label}</span>
                {chip.onDismiss ? (
                  <button
                    type="button"
                    aria-label={`Dismiss: ${chip.label}`}
                    onClick={(e) => {
                      e.stopPropagation()
                      chip.onDismiss?.()
                    }}
                    className="-mr-1 ml-1 inline-flex h-4 w-4 items-center justify-center rounded-full text-current/80 hover:bg-white/15 hover:text-current"
                  >
                    <span aria-hidden="true">×</span>
                  </button>
                ) : null}
              </>
            )

            return (
              <li key={key}>
                {chip.href ? (
                  <a href={chip.href} className={chipClass}>
                    {content}
                  </a>
                ) : chip.onClick ? (
                  <button type="button" onClick={chip.onClick} className={chipClass}>
                    {content}
                  </button>
                ) : (
                  <span className={chipClass}>{content}</span>
                )}
              </li>
            )
          })}
        </ul>
      </div>
    </div>
  )
}

NowBar.displayName = 'NowBar'

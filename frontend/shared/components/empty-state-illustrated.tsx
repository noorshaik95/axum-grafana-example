import * as React from 'react'
import { cn } from '../utils/index'

export type IllustrationKind = 'empty-inbox' | 'no-results' | 'all-done'

export interface EmptyStateIllustratedProps extends React.HTMLAttributes<HTMLDivElement> {
  illustration: IllustrationKind
  title: string
  description?: string
  action?: React.ReactNode
}

function Illustration({ kind }: { kind: IllustrationKind }) {
  // Forest-palette SVGs — kept small + inline so no extra assets to bundle.
  const common = 'h-28 w-28'
  switch (kind) {
    case 'empty-inbox':
      return (
        <svg
          data-testid="illustration-empty-inbox"
          viewBox="0 0 120 120"
          className={common}
          aria-hidden="true"
        >
          <rect x="14" y="38" width="92" height="60" rx="8" fill="var(--color-forest-100)" />
          <path
            d="M14 72h24l6 10h32l6-10h24"
            stroke="var(--color-forest-600)"
            strokeWidth="3"
            fill="none"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
          <rect
            x="32"
            y="22"
            width="56"
            height="32"
            rx="4"
            fill="var(--color-paper)"
            stroke="var(--color-forest-600)"
            strokeWidth="2"
          />
          <path
            d="M42 34h36M42 42h24"
            stroke="var(--color-forest-600)"
            strokeWidth="2"
            strokeLinecap="round"
          />
        </svg>
      )
    case 'no-results':
      return (
        <svg
          data-testid="illustration-no-results"
          viewBox="0 0 120 120"
          className={common}
          aria-hidden="true"
        >
          <circle
            cx="52"
            cy="52"
            r="28"
            fill="var(--color-forest-100)"
            stroke="var(--color-forest-600)"
            strokeWidth="3"
          />
          <line
            x1="74"
            y1="74"
            x2="98"
            y2="98"
            stroke="var(--color-forest-600)"
            strokeWidth="5"
            strokeLinecap="round"
          />
          <path
            d="M42 52h20M52 42v20"
            stroke="var(--color-forest-600)"
            strokeWidth="2"
            strokeLinecap="round"
            opacity="0.3"
          />
        </svg>
      )
    case 'all-done':
      return (
        <svg
          data-testid="illustration-all-done"
          viewBox="0 0 120 120"
          className={common}
          aria-hidden="true"
        >
          <circle cx="60" cy="60" r="44" fill="var(--color-forest-100)" />
          <path
            d="M42 62l12 12 26-28"
            stroke="var(--color-forest-600)"
            strokeWidth="6"
            strokeLinecap="round"
            strokeLinejoin="round"
            fill="none"
          />
        </svg>
      )
  }
}

export function EmptyStateIllustrated({
  illustration,
  title,
  description,
  action,
  className,
  ...rest
}: EmptyStateIllustratedProps) {
  return (
    <div
      data-testid="empty-state-illustrated"
      className={cn('flex flex-col items-center justify-center py-12 text-center', className)}
      {...rest}
    >
      <Illustration kind={illustration} />
      <h3 className="mt-4 text-lg font-semibold text-warm-900">{title}</h3>
      {description ? <p className="mt-1 max-w-sm text-sm text-warm-700">{description}</p> : null}
      {action ? <div className="mt-4">{action}</div> : null}
    </div>
  )
}

EmptyStateIllustrated.displayName = 'EmptyStateIllustrated'

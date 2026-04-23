'use client'

import * as React from 'react'
import { cn } from '../utils/index'
import { Button } from './ui/button'

export interface ActionHeroProps extends React.HTMLAttributes<HTMLElement> {
  title: string
  subtitle?: string
  ctaLabel?: string
  ctaHref?: string
  onCta?: () => void
  /** Optional right-side slot, e.g. a secondary link or stats chip. */
  aside?: React.ReactNode
}

/**
 * Landing hero for student /today and provider /teach.
 * Instrument Serif heading, warm subtitle, primary CTA.
 */
export function ActionHero({
  title,
  subtitle,
  ctaLabel,
  ctaHref,
  onCta,
  aside,
  className,
  ...rest
}: ActionHeroProps) {
  const hasCta = Boolean(ctaLabel)
  return (
    <section
      data-testid="action-hero"
      className={cn(
        'flex flex-col gap-4 rounded-card bg-paper p-6 sm:flex-row sm:items-center sm:justify-between sm:p-8',
        className
      )}
      {...rest}
    >
      <div className="flex-1 min-w-0">
        <h1 className="font-display text-3xl leading-tight text-warm-900 sm:text-4xl">{title}</h1>
        {subtitle ? <p className="mt-2 text-sm text-warm-700 sm:text-base">{subtitle}</p> : null}
      </div>
      {(hasCta || aside) && (
        <div className="flex shrink-0 items-center gap-3">
          {aside}
          {hasCta ? (
            ctaHref ? (
              <a href={ctaHref}>
                <Button variant="default">{ctaLabel}</Button>
              </a>
            ) : (
              <Button variant="default" onClick={onCta}>
                {ctaLabel}
              </Button>
            )
          ) : null}
        </div>
      )}
    </section>
  )
}

ActionHero.displayName = 'ActionHero'

/**
 * Unit tests for NowBar — hide-when-empty, tone classes, dismiss, optional label.
 */

import * as React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { NowBar, type NowBarChip } from '../../../components/app-shell/now-bar'

describe('NowBar', () => {
  it('auto-hides when chips is empty (renders nothing)', () => {
    const { container } = render(<NowBar chips={[]} />)
    expect(container).toBeEmptyDOMElement()
    expect(screen.queryByTestId('now-bar')).toBeNull()
  })

  it('auto-hides when chips is undefined (safe default)', () => {
    const { container } = render(<NowBar chips={undefined as unknown as NowBarChip[]} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders the forest-700 strip when given at least one chip', () => {
    render(<NowBar chips={[{ label: '1 P1 open · 42 min' }]} />)
    const bar = screen.getByTestId('now-bar')
    expect(bar).toBeInTheDocument()
    expect(bar.className).toMatch(/bg-forest-700/)
    expect(bar.className).toMatch(/text-forest-50/)
  })

  it('renders each chip label and respects the optional context label', () => {
    const chips: NowBarChip[] = [
      { label: '1 P1 open · 42 min' },
      { label: 'Stanford · grade export fails', tone: 'amber' },
      { label: 'Berkeley onboarding · 67%' },
    ]
    render(<NowBar chips={chips} label="Now" />)

    expect(screen.getByText('Now')).toBeInTheDocument()
    chips.forEach((c) => expect(screen.getByText(c.label)).toBeInTheDocument())
  })

  it('applies the amber tone class to amber chips', () => {
    render(<NowBar chips={[{ label: 'Stanford · grade export fails', tone: 'amber' }]} />)
    const chip = screen.getByText('Stanford · grade export fails').parentElement!
    expect(chip.className).toMatch(/bg-amber-400/)
  })

  it('calls onDismiss when the × button is clicked', () => {
    const onDismiss = jest.fn()
    render(<NowBar chips={[{ label: 'dismissible', onDismiss }]} />)

    const dismiss = screen.getByRole('button', { name: /dismiss: dismissible/i })
    fireEvent.click(dismiss)
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('renders a chip as an anchor when href is provided', () => {
    render(<NowBar chips={[{ label: 'incident INC-42', href: '/incidents/42' }]} />)
    const link = screen.getByRole('link', { name: /incident inc-42/i })
    expect(link).toHaveAttribute('href', '/incidents/42')
  })

  it('renders a chip as a button when onClick is provided without href', () => {
    const onClick = jest.fn()
    render(<NowBar chips={[{ label: 'Acknowledge', onClick }]} />)

    const btn = screen.getByRole('button', { name: /acknowledge/i })
    fireEvent.click(btn)
    expect(onClick).toHaveBeenCalledTimes(1)
  })
})

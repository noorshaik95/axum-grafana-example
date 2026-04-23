/**
 * Unit tests for TabNav — covers active-state detection, count badges, and
 * the neutral/amber/red/forest/blue tone map for non-active badges.
 */

import * as React from 'react'
import { render, screen } from '@testing-library/react'
import { TabNav, type TabItem } from '../../../components/app-shell/tab-nav'

jest.mock('next/link', () => ({
  __esModule: true,
  default: ({ href, children, ...rest }: any) =>
    React.createElement('a', { href, ...rest }, children),
}))

const pathnameMock = jest.fn<string, []>()
jest.mock('next/navigation', () => ({
  usePathname: () => pathnameMock(),
}))

const TABS: TabItem[] = [
  { label: 'Overview', href: '/overview' },
  { label: 'Schools', href: '/schools', count: 42 },
  { label: 'People', href: '/people' },
  { label: 'Support', href: '/support', count: 3, badgeTone: 'amber' },
]

describe('TabNav', () => {
  beforeEach(() => {
    pathnameMock.mockReturnValue('/overview')
  })

  it('marks the tab matching the current pathname as active', () => {
    pathnameMock.mockReturnValue('/schools')
    render(<TabNav tabs={TABS} />)

    const active = screen.getByRole('link', { name: /schools/i })
    expect(active).toHaveAttribute('aria-current', 'page')
    expect(active).toHaveAttribute('data-active', 'true')
    expect(active.className).toMatch(/bg-forest-100/)
    expect(active.className).toMatch(/text-forest-800/)
  })

  it('treats descendants of a tab href as active (e.g. /schools/eastfield)', () => {
    pathnameMock.mockReturnValue('/schools/eastfield')
    render(<TabNav tabs={TABS} />)

    const active = screen.getByRole('link', { name: /schools/i })
    expect(active).toHaveAttribute('aria-current', 'page')
  })

  it('renders non-active tabs with warm-700 text and forest-50 hover', () => {
    pathnameMock.mockReturnValue('/overview')
    render(<TabNav tabs={TABS} />)

    const people = screen.getByRole('link', { name: /people/i })
    expect(people).not.toHaveAttribute('aria-current')
    expect(people.className).toMatch(/text-warm-700/)
    expect(people.className).toMatch(/hover:bg-forest-50/)
  })

  it('respects the explicit activeHref override', () => {
    pathnameMock.mockReturnValue('/overview')
    render(<TabNav tabs={TABS} activeHref="/people" />)

    expect(screen.getByRole('link', { name: /overview/i })).not.toHaveAttribute('aria-current')
    expect(screen.getByRole('link', { name: /people/i })).toHaveAttribute('aria-current', 'page')
  })

  it('shows a count badge next to the label', () => {
    pathnameMock.mockReturnValue('/overview')
    render(<TabNav tabs={TABS} />)

    const counts = screen.getAllByTestId('tab-count')
    expect(counts.map((c) => c.textContent)).toEqual(['42', '3'])
  })

  it('applies the badgeTone colour when tab is inactive', () => {
    pathnameMock.mockReturnValue('/overview')
    render(<TabNav tabs={TABS} />)

    const support = screen.getByRole('link', { name: /support/i })
    const badge = support.querySelector('[data-testid="tab-count"]') as HTMLElement
    expect(badge.className).toMatch(/bg-amber-100/)
    expect(badge.className).toMatch(/text-amber-700/)
  })

  it('shows the active-state badge in forest-600/white when the tab is current', () => {
    pathnameMock.mockReturnValue('/schools')
    render(<TabNav tabs={TABS} />)

    const schools = screen.getByRole('link', { name: /schools/i })
    const badge = schools.querySelector('[data-testid="tab-count"]') as HTMLElement
    expect(badge.className).toMatch(/bg-forest-600/)
    expect(badge.className).toMatch(/text-white/)
  })

  it('renders nav with role=navigation and provided aria-label', () => {
    pathnameMock.mockReturnValue('/overview')
    render(<TabNav tabs={TABS} ariaLabel="Admin primary nav" />)

    expect(screen.getByRole('navigation', { name: 'Admin primary nav' })).toBeInTheDocument()
  })
})

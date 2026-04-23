import * as React from 'react'
import { render, screen } from '@testing-library/react'
import { PriorityBadge } from '../../components/priority-badge'

describe('<PriorityBadge />', () => {
  it('renders the priority code', () => {
    render(<PriorityBadge priority="P0" />)
    expect(screen.getByTestId('priority-badge')).toHaveTextContent('P0')
  })

  it('P0 uses red-500 background + white text', () => {
    render(<PriorityBadge priority="P0" />)
    const el = screen.getByTestId('priority-badge')
    expect(el.className).toMatch(/bg-red-500/)
    expect(el.className).toMatch(/text-white/)
  })

  it('P1 uses red-100/red-700', () => {
    render(<PriorityBadge priority="P1" />)
    const el = screen.getByTestId('priority-badge')
    expect(el.className).toMatch(/bg-red-100/)
    expect(el.className).toMatch(/text-red-700/)
  })

  it('P2 uses amber-100/amber-700', () => {
    render(<PriorityBadge priority="P2" />)
    expect(screen.getByTestId('priority-badge').className).toMatch(/bg-amber-100/)
  })

  it('P3 uses warm-100/warm-700', () => {
    render(<PriorityBadge priority="P3" />)
    expect(screen.getByTestId('priority-badge').className).toMatch(/bg-warm-100/)
  })

  it('P4 uses blue-100/blue-700', () => {
    render(<PriorityBadge priority="P4" />)
    expect(screen.getByTestId('priority-badge').className).toMatch(/bg-blue-100/)
  })

  it('renders optional label alongside the code', () => {
    render(<PriorityBadge priority="P2" label="Major" />)
    expect(screen.getByText('Major')).toBeInTheDocument()
  })
})

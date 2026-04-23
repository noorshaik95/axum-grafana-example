import * as React from 'react'
import { render, screen } from '@testing-library/react'
import { AgendaItem } from '../../components/agenda-item'

describe('<AgendaItem />', () => {
  it('renders title, time and urgency chip label', () => {
    render(<AgendaItem title="PS4 due" urgency="due-soon" timeUntil="in 2h" />)
    expect(screen.getByText('PS4 due')).toBeInTheDocument()
    expect(screen.getByText('in 2h')).toBeInTheDocument()
    expect(screen.getByText('Due soon')).toBeInTheDocument()
  })

  it('tags urgency data attribute', () => {
    render(<AgendaItem title="x" urgency="overdue" timeUntil="2d late" />)
    expect(screen.getByTestId('agenda-item')).toHaveAttribute('data-urgency', 'overdue')
  })

  it('overdue chip uses red palette', () => {
    render(<AgendaItem title="x" urgency="overdue" timeUntil="1d" />)
    const chip = screen.getByText('Overdue')
    expect(chip.className).toMatch(/bg-red-100/)
    expect(chip.className).toMatch(/text-red-700/)
  })

  it('upcoming chip uses forest palette', () => {
    render(<AgendaItem title="x" urgency="upcoming" timeUntil="tomorrow" />)
    const chip = screen.getByText('Upcoming')
    expect(chip.className).toMatch(/bg-forest-100/)
  })

  it('renders optional icon when provided', () => {
    render(
      <AgendaItem
        title="x"
        urgency="upcoming"
        timeUntil="later"
        icon={<span data-testid="ic">I</span>}
      />
    )
    expect(screen.getByTestId('ic')).toBeInTheDocument()
  })
})

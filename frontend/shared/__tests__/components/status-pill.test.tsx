import * as React from 'react'
import { render, screen } from '@testing-library/react'
import { StatusPill } from '../../components/status-pill'

describe('<StatusPill />', () => {
  it('renders label and tone attribute', () => {
    render(<StatusPill tone="green" label="Healthy" />)
    const pill = screen.getByTestId('status-pill')
    expect(pill).toHaveAttribute('data-tone', 'green')
    expect(pill).toHaveTextContent('Healthy')
  })

  it('applies forest palette for green tone', () => {
    render(<StatusPill tone="green" label="Ok" />)
    expect(screen.getByTestId('status-pill').className).toMatch(/bg-forest-100/)
  })

  it('applies red palette for red tone', () => {
    render(<StatusPill tone="red" label="Down" />)
    expect(screen.getByTestId('status-pill').className).toMatch(/bg-red-100/)
  })

  it('renders an optional count chip', () => {
    render(<StatusPill tone="amber" label="Schools" count={42} />)
    expect(screen.getByText('42')).toBeInTheDocument()
  })

  it('omits count when not provided', () => {
    render(<StatusPill tone="blue" label="Open" />)
    expect(screen.queryByText(/^\d+$/)).toBeNull()
  })
})

import * as React from 'react'
import { render, screen } from '@testing-library/react'
import { ListRow } from '../../components/list-row'

describe('<ListRow />', () => {
  it('renders primary and secondary labels', () => {
    render(<ListRow primary="Alice" secondary="Student · CS101" />)
    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('Student · CS101')).toBeInTheDocument()
  })

  it('applies warm-50 when index is odd', () => {
    render(<ListRow index={1} primary="Row" />)
    expect(screen.getByTestId('list-row').className).toMatch(/bg-warm-50/)
  })

  it('applies cream when index is even', () => {
    render(<ListRow index={0} primary="Row" />)
    expect(screen.getByTestId('list-row').className).toMatch(/bg-cream/)
  })

  it('renders leading, trailing and action slots', () => {
    render(
      <ListRow
        primary="Row"
        leading={<span>L</span>}
        trailing={<span>T</span>}
        action={<button type="button">A</button>}
      />
    )
    expect(screen.getByText('L')).toBeInTheDocument()
    expect(screen.getByText('T')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'A' })).toBeInTheDocument()
  })

  it('adds interactive styles when interactive=true', () => {
    render(<ListRow primary="Row" interactive />)
    expect(screen.getByTestId('list-row').className).toMatch(/cursor-pointer/)
  })
})

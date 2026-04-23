import * as React from 'react'
import { render, screen } from '@testing-library/react'
import { EmptyStateIllustrated } from '../../components/empty-state-illustrated'

describe('<EmptyStateIllustrated />', () => {
  it('renders the empty-inbox illustration', () => {
    render(<EmptyStateIllustrated illustration="empty-inbox" title="All caught up" />)
    expect(screen.getByTestId('illustration-empty-inbox')).toBeInTheDocument()
    expect(screen.getByText('All caught up')).toBeInTheDocument()
  })

  it('renders the no-results illustration', () => {
    render(<EmptyStateIllustrated illustration="no-results" title="Nothing found" />)
    expect(screen.getByTestId('illustration-no-results')).toBeInTheDocument()
  })

  it('renders the all-done illustration', () => {
    render(<EmptyStateIllustrated illustration="all-done" title="Done!" />)
    expect(screen.getByTestId('illustration-all-done')).toBeInTheDocument()
  })

  it('renders description and action when provided', () => {
    render(
      <EmptyStateIllustrated
        illustration="empty-inbox"
        title="Title"
        description="Nothing to see"
        action={<button type="button">Go</button>}
      />
    )
    expect(screen.getByText('Nothing to see')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Go' })).toBeInTheDocument()
  })
})

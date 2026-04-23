import * as React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { ImpersonationBanner } from '../../components/impersonation-banner'

describe('<ImpersonationBanner />', () => {
  it('renders email and end button', () => {
    const onEnd = jest.fn()
    render(<ImpersonationBanner email="alice@eastfield.edu" onEnd={onEnd} />)
    expect(screen.getByTestId('impersonation-banner')).toHaveTextContent(
      'Impersonating alice@eastfield.edu'
    )
    fireEvent.click(screen.getByRole('button', { name: /end session/i }))
    expect(onEnd).toHaveBeenCalledTimes(1)
  })

  it('renders optional tenant name', () => {
    render(
      <ImpersonationBanner email="alice@eastfield.edu" tenantName="Eastfield" onEnd={() => {}} />
    )
    expect(screen.getByTestId('impersonation-banner')).toHaveTextContent('(Eastfield)')
  })

  it('respects custom end label', () => {
    render(<ImpersonationBanner email="a@b" onEnd={() => {}} endLabel="Return to Admin" />)
    expect(screen.getByRole('button', { name: 'Return to Admin' })).toBeInTheDocument()
  })
})

import * as React from 'react'
import { render, screen } from '@testing-library/react'
import { RiskBadge } from '../../components/risk-badge'

describe('<RiskBadge />', () => {
  it.each([
    ['healthy', 'forest-100', 'forest-700', 'Healthy'],
    ['slipping', 'amber-100', 'amber-700', 'Slipping'],
    ['at_risk', 'red-100', 'red-700', 'At risk'],
  ] as const)('%s uses %s bg and %s text and label "%s"', (risk, bg, text, label) => {
    render(<RiskBadge risk={risk} />)
    const el = screen.getByTestId('risk-badge')
    expect(el).toHaveAttribute('data-risk', risk)
    expect(el.className).toMatch(new RegExp(`bg-${bg}`))
    expect(el.className).toMatch(new RegExp(`text-${text}`))
    expect(el).toHaveTextContent(label)
  })

  it('supports label override', () => {
    render(<RiskBadge risk="at_risk" label="Critical" />)
    expect(screen.getByTestId('risk-badge')).toHaveTextContent('Critical')
  })
})

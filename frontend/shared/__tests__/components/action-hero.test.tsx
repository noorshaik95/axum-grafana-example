import * as React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { ActionHero } from '../../components/action-hero'

describe('<ActionHero />', () => {
  it('renders title with display font class', () => {
    render(<ActionHero title="Good morning, Alice" />)
    const heading = screen.getByRole('heading', { name: 'Good morning, Alice' })
    expect(heading.className).toMatch(/font-display/)
  })

  it('renders subtitle in warm-700 when provided', () => {
    render(<ActionHero title="Hello" subtitle="Welcome back" />)
    const subtitle = screen.getByText('Welcome back')
    expect(subtitle.className).toMatch(/text-warm-700/)
  })

  it('renders a CTA button when ctaLabel is given', () => {
    const onCta = jest.fn()
    render(<ActionHero title="Hi" ctaLabel="Start" onCta={onCta} />)
    const btn = screen.getByRole('button', { name: 'Start' })
    fireEvent.click(btn)
    expect(onCta).toHaveBeenCalledTimes(1)
  })

  it('wraps CTA in an anchor when ctaHref is given', () => {
    render(<ActionHero title="Hi" ctaLabel="Start" ctaHref="/today" />)
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', '/today')
    expect(link).toHaveTextContent('Start')
  })

  it('does not render CTA when ctaLabel missing', () => {
    render(<ActionHero title="Hi" />)
    expect(screen.queryByRole('button')).toBeNull()
  })
})

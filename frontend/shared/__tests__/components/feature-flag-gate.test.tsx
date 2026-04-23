import * as React from 'react'
import { render, screen } from '@testing-library/react'
import { FeatureFlagGate, FlagProvider, useFeatureFlag } from '../../components/feature-flag-gate'

describe('<FeatureFlagGate />', () => {
  it('renders fallback by default when no provider is mounted', () => {
    render(
      <FeatureFlagGate name="ai_draft_feedback" fallback={<span>fallback</span>}>
        <span>on</span>
      </FeatureFlagGate>
    )
    expect(screen.getByText('fallback')).toBeInTheDocument()
    expect(screen.queryByText('on')).toBeNull()
  })

  it('renders children when the provider says the flag is on', () => {
    const useFlag = (name: string) => name === 'ai_draft_feedback'
    render(
      <FlagProvider useFlag={useFlag}>
        <FeatureFlagGate name="ai_draft_feedback">
          <span>on</span>
        </FeatureFlagGate>
      </FlagProvider>
    )
    expect(screen.getByText('on')).toBeInTheDocument()
  })

  it('renders fallback when provider returns false', () => {
    render(
      <FlagProvider useFlag={() => false}>
        <FeatureFlagGate name="x" fallback={<span>fb</span>}>
          <span>on</span>
        </FeatureFlagGate>
      </FlagProvider>
    )
    expect(screen.getByText('fb')).toBeInTheDocument()
  })

  it('exposes the raw hook', () => {
    const Probe = () => {
      const on = useFeatureFlag('x')
      return <span>{String(on)}</span>
    }
    render(
      <FlagProvider useFlag={(n) => n === 'x'}>
        <Probe />
      </FlagProvider>
    )
    expect(screen.getByText('true')).toBeInTheDocument()
  })
})

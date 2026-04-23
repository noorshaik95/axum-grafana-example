'use client'

import * as React from 'react'

export type UseFlagHook = (name: string) => boolean

export interface FlagProviderProps {
  /**
   * Implementation-specific hook. Each portal wires its own pointing at
   * feature-flag-service. The default (no provider) returns false for every flag.
   */
  useFlag: UseFlagHook
  children: React.ReactNode
}

const defaultUseFlag: UseFlagHook = () => false

const FlagContext = React.createContext<UseFlagHook>(defaultUseFlag)

export function FlagProvider({ useFlag, children }: FlagProviderProps) {
  return <FlagContext.Provider value={useFlag}>{children}</FlagContext.Provider>
}

/** Hook form — call anywhere inside a FlagProvider. */
export function useFeatureFlag(name: string): boolean {
  const useFlag = React.useContext(FlagContext)
  return useFlag(name)
}

export interface FeatureFlagGateProps {
  name: string
  fallback?: React.ReactNode
  children: React.ReactNode
}

/**
 * Render `children` iff the given flag resolves truthy; otherwise render `fallback`
 * (default `null`). Safe to use outside a `FlagProvider`; the gate then always
 * renders the fallback.
 */
export function FeatureFlagGate({ name, fallback = null, children }: FeatureFlagGateProps) {
  const enabled = useFeatureFlag(name)
  return <>{enabled ? children : fallback}</>
}

FeatureFlagGate.displayName = 'FeatureFlagGate'

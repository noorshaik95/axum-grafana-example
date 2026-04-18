'use client'

import { useState } from 'react'
import { ArrowLeft, ArrowRight, CheckCircle } from 'lucide-react'

const STEPS = [
  { label: 'School details', done: true },
  { label: 'Identity & SSO', done: false },
  { label: 'Data import', done: false },
  { label: 'Verify & launch', done: false },
]

const PROVIDERS = [
  { id: 'saml', label: 'SAML 2.0', desc: 'Works with Okta, Azure AD, ADFS', icon: '🔐' },
  { id: 'oidc', label: 'OIDC / OAuth2', desc: 'Works with Auth0, PingIdentity', icon: '🔑' },
  {
    id: 'google',
    label: 'Google Workspace',
    desc: 'Sign in with institutional Google accounts',
    icon: '🇬',
  },
  {
    id: 'email',
    label: 'Email + Password',
    desc: 'Managed credentials, no SSO required',
    icon: '✉️',
  },
]

export default function OnboardSchoolPage() {
  const [selected, setSelected] = useState<string | null>('saml')
  const currentStep = 1

  return (
    <div className="max-w-lg mx-auto space-y-8">
      {/* Step indicator */}
      <div>
        <p className="text-xs font-medium text-[#6a6e62] mb-3">
          Step {currentStep + 1} of {STEPS.length} · {STEPS[currentStep].label}
        </p>
        <div className="flex items-center gap-2">
          {STEPS.map((step, i) => (
            <div key={step.label} className="flex items-center gap-2 flex-1">
              <div className="flex items-center gap-1.5 shrink-0">
                {step.done ? (
                  <CheckCircle className="h-4 w-4 text-[#3e7d4f]" />
                ) : i === currentStep ? (
                  <div className="h-4 w-4 rounded-full border-2 border-[#3e7d4f] bg-white" />
                ) : (
                  <div className="h-4 w-4 rounded-full border-2 border-[#e4e0d4]" />
                )}
              </div>
              {i < STEPS.length - 1 && (
                <div className={`flex-1 h-0.5 ${step.done ? 'bg-[#3e7d4f]' : 'bg-[#e4e0d4]'}`} />
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Heading */}
      <div>
        <h1
          className="text-3xl font-medium text-[#12170f]"
          style={{ fontFamily: 'Instrument Serif, Georgia, serif' }}
        >
          Connect Stanford&apos;s identity.
        </h1>
        <p className="text-sm text-[#6a6e62] mt-2">
          Choose how faculty and students will authenticate. You can add more providers later.
        </p>
      </div>

      {/* Provider cards 2x2 */}
      <div className="grid grid-cols-2 gap-3">
        {PROVIDERS.map((prov) => (
          <button
            key={prov.id}
            onClick={() => setSelected(prov.id)}
            className={`rounded-xl p-4 text-left border transition-all ${
              selected === prov.id
                ? 'border-[#3e7d4f] bg-[#f2f7f3] shadow-sm'
                : 'border-[#e4e0d4] bg-white hover:border-[#8ab694]'
            }`}
          >
            <span className="text-2xl mb-2 block">{prov.icon}</span>
            <p className="text-sm font-semibold text-[#12170f]">{prov.label}</p>
            <p className="text-xs text-[#6a6e62] mt-0.5 leading-snug">{prov.desc}</p>
            {selected === prov.id && (
              <div className="mt-2 flex items-center gap-1 text-[#3e7d4f] text-xs font-medium">
                <CheckCircle className="h-3 w-3" /> Selected
              </div>
            )}
          </button>
        ))}
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between pt-2">
        <button className="flex items-center gap-2 px-4 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm font-medium text-[#6a6e62] hover:bg-[#f6f3ec] transition-colors">
          <ArrowLeft className="h-4 w-4" />
          Back
        </button>
        <button
          disabled={!selected}
          className="flex items-center gap-2 px-5 py-2 rounded-lg bg-[#234e32] text-white text-sm font-medium hover:bg-[#1a3a26] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          Continue
          <ArrowRight className="h-4 w-4" />
        </button>
      </div>
    </div>
  )
}

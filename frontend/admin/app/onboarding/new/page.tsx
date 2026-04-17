'use client'

import { OnboardingWizard } from '@/components/onboarding/wizard/OnboardingWizard'
import Link from 'next/link'
import { ChevronLeft } from 'lucide-react'

export default function NewOnboardingPage() {
  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/onboarding"
          className="mb-2 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
        >
          <ChevronLeft className="h-4 w-4" />
          Back to Onboarding
        </Link>
        <h1 className="text-2xl font-bold">New University Onboarding</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Complete the wizard to onboard a new institution
        </p>
      </div>

      <OnboardingWizard />
    </div>
  )
}

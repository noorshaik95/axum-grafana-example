'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '../../../../shared/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../../../../shared/components/ui/card'
import { Input } from '../../../../shared/components/ui/input'
import { Label } from '../../../../shared/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../../../../shared/components/ui/select'
import { Progress } from '../../../../shared/components/ui/progress'
import { useToast } from '@/hooks/use-toast'
import { useStartOnboarding } from '@/lib/hooks/use-admin-queries'
import { onboardingApi } from '@/lib/api/onboarding'
import type { InstitutionDetails, AdminUserDetails, ResourcePlan } from '@/lib/api/types'
import { Check, ChevronLeft, ChevronRight, Loader2 } from 'lucide-react'

const STEPS = [
  { id: 1, title: 'Institution Details' },
  { id: 2, title: 'Admin User' },
  { id: 3, title: 'Resource Plan' },
  { id: 4, title: 'Review & Submit' },
]

const TIMEZONES = [
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'Europe/London',
  'Europe/Berlin',
  'Asia/Tokyo',
  'Asia/Kolkata',
  'Australia/Sydney',
]

const PLAN_TIERS: Array<{
  value: ResourcePlan['planTier']
  label: string
  maxUsers: number
  maxCourses: number
  storageGb: number
}> = [
  {
    value: 'starter',
    label: 'Starter',
    maxUsers: 500,
    maxCourses: 50,
    storageGb: 10,
  },
  {
    value: 'standard',
    label: 'Standard',
    maxUsers: 5000,
    maxCourses: 500,
    storageGb: 100,
  },
  {
    value: 'enterprise',
    label: 'Enterprise',
    maxUsers: 50000,
    maxCourses: 5000,
    storageGb: 1000,
  },
]

export function OnboardingWizard() {
  const router = useRouter()
  const { toast } = useToast()
  const startOnboarding = useStartOnboarding()
  const [currentStep, setCurrentStep] = useState(1)

  const [institution, setInstitution] = useState<InstitutionDetails>({
    name: '',
    address: '',
    city: '',
    state: '',
    country: '',
    postalCode: '',
    phone: '',
    website: '',
    timezone: 'America/New_York',
    locale: 'en-US',
  })

  const [adminUser, setAdminUser] = useState<AdminUserDetails>({
    email: '',
    firstName: '',
    lastName: '',
    phone: '',
  })

  const [resourcePlan, setResourcePlan] = useState<ResourcePlan>({
    planTier: 'standard',
    maxUsers: 5000,
    maxCourses: 500,
    storageQuotaGb: 100,
    features: ['video_conferencing', 'analytics', 'api_access'],
  })

  const progress = (currentStep / STEPS.length) * 100

  const canProceed = (): boolean => {
    switch (currentStep) {
      case 1:
        return !!(
          institution.name &&
          institution.address &&
          institution.city &&
          institution.country
        )
      case 2:
        return !!(adminUser.email && adminUser.firstName && adminUser.lastName)
      case 3:
        return !!resourcePlan.planTier
      case 4:
        return true
      default:
        return false
    }
  }

  const handleSubmit = async () => {
    try {
      // Restate OnboardingWorkflow::start expects only the minimal
      // snake_case shape per services/onboarding-service/src/state.rs.
      const result = await startOnboarding.mutateAsync({
        institution_name: institution.name,
        admin_email: adminUser.email,
      })
      // Persist the rich wizard payload as step 1 so the workflow has the
      // full context (slug/domain/resource plan/admin name) available for
      // downstream steps. Non-blocking: the workflow is already started.
      try {
        await onboardingApi.saveStep(result.id, 1, {
          institutionDetails: institution,
          adminUser,
          resourcePlan,
        })
      } catch {
        // save-step failure doesn't invalidate the started workflow —
        // admin can retry from the detail page.
      }
      toast({
        title: 'Onboarding started',
        description: `Institution "${institution.name}" has been submitted for review.`,
      })
      router.push(`/onboarding/${result.id}`)
    } catch (err) {
      toast({
        title: 'Failed to start onboarding',
        description: err instanceof Error ? err.message : 'An error occurred',
        variant: 'destructive',
      })
    }
  }

  const updateInstitution = (field: keyof InstitutionDetails, value: string) =>
    setInstitution((prev) => ({ ...prev, [field]: value }))

  const updateAdmin = (field: keyof AdminUserDetails, value: string) =>
    setAdminUser((prev) => ({ ...prev, [field]: value }))

  const selectPlanTier = (tier: ResourcePlan['planTier']) => {
    const preset = PLAN_TIERS.find((p) => p.value === tier)
    if (preset) {
      setResourcePlan((prev) => ({
        ...prev,
        planTier: tier,
        maxUsers: preset.maxUsers,
        maxCourses: preset.maxCourses,
        storageQuotaGb: preset.storageGb,
      }))
    }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      {/* Progress */}
      <div>
        <div className="mb-2 flex items-center justify-between text-sm">
          <span className="font-medium">
            Step {currentStep} of {STEPS.length}
          </span>
          <span className="text-muted-foreground">{STEPS[currentStep - 1].title}</span>
        </div>
        <Progress value={progress} className="h-2" />
        <div className="mt-3 flex justify-between">
          {STEPS.map((step) => (
            <div
              key={step.id}
              className={`flex items-center gap-1.5 text-xs ${
                step.id <= currentStep ? 'text-primary font-medium' : 'text-muted-foreground'
              }`}
            >
              <div
                className={`flex h-6 w-6 items-center justify-center rounded-full border text-xs ${
                  step.id < currentStep
                    ? 'border-primary bg-primary text-primary-foreground'
                    : step.id === currentStep
                      ? 'border-primary text-primary'
                      : 'border-muted-foreground/30 text-muted-foreground'
                }`}
              >
                {step.id < currentStep ? <Check className="h-3 w-3" /> : step.id}
              </div>
              <span className="hidden sm:inline">{step.title}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Step Content */}
      <Card>
        <CardHeader>
          <CardTitle>{STEPS[currentStep - 1].title}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {currentStep === 1 && (
            <>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2 sm:col-span-2">
                  <Label htmlFor="instName">Institution Name *</Label>
                  <Input
                    id="instName"
                    value={institution.name}
                    onChange={(e) => updateInstitution('name', e.target.value)}
                    placeholder="University of Example"
                  />
                </div>
                <div className="space-y-2 sm:col-span-2">
                  <Label htmlFor="address">Address *</Label>
                  <Input
                    id="address"
                    value={institution.address}
                    onChange={(e) => updateInstitution('address', e.target.value)}
                    placeholder="123 Main Street"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="city">City *</Label>
                  <Input
                    id="city"
                    value={institution.city}
                    onChange={(e) => updateInstitution('city', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="state">State / Province</Label>
                  <Input
                    id="state"
                    value={institution.state}
                    onChange={(e) => updateInstitution('state', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="country">Country *</Label>
                  <Input
                    id="country"
                    value={institution.country}
                    onChange={(e) => updateInstitution('country', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="postalCode">Postal Code</Label>
                  <Input
                    id="postalCode"
                    value={institution.postalCode}
                    onChange={(e) => updateInstitution('postalCode', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="phone">Phone</Label>
                  <Input
                    id="phone"
                    value={institution.phone}
                    onChange={(e) => updateInstitution('phone', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="website">Website</Label>
                  <Input
                    id="website"
                    value={institution.website}
                    onChange={(e) => updateInstitution('website', e.target.value)}
                    placeholder="https://example.edu"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Timezone</Label>
                  <Select
                    value={institution.timezone}
                    onValueChange={(v) => updateInstitution('timezone', v)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {TIMEZONES.map((tz) => (
                        <SelectItem key={tz} value={tz}>
                          {tz}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>Locale</Label>
                  <Select
                    value={institution.locale}
                    onValueChange={(v) => updateInstitution('locale', v)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="en-US">English (US)</SelectItem>
                      <SelectItem value="en-GB">English (UK)</SelectItem>
                      <SelectItem value="es-ES">Spanish</SelectItem>
                      <SelectItem value="fr-FR">French</SelectItem>
                      <SelectItem value="de-DE">German</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </>
          )}

          {currentStep === 2 && (
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2 sm:col-span-2">
                <Label htmlFor="adminEmail">Admin Email *</Label>
                <Input
                  id="adminEmail"
                  type="email"
                  name="adminEmail"
                  value={adminUser.email}
                  onChange={(e) => updateAdmin('email', e.target.value)}
                  placeholder="admin@university.edu"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="firstName">First Name *</Label>
                <Input
                  id="firstName"
                  value={adminUser.firstName}
                  onChange={(e) => updateAdmin('firstName', e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="lastName">Last Name *</Label>
                <Input
                  id="lastName"
                  value={adminUser.lastName}
                  onChange={(e) => updateAdmin('lastName', e.target.value)}
                />
              </div>
              <div className="space-y-2 sm:col-span-2">
                <Label htmlFor="adminPhone">Phone (optional)</Label>
                <Input
                  id="adminPhone"
                  value={adminUser.phone ?? ''}
                  onChange={(e) => updateAdmin('phone', e.target.value)}
                />
              </div>
            </div>
          )}

          {currentStep === 3 && (
            <div className="space-y-6">
              <div className="space-y-2">
                <Label>Plan Tier</Label>
                <Select
                  value={resourcePlan.planTier}
                  onValueChange={(v) => selectPlanTier(v as ResourcePlan['planTier'])}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {PLAN_TIERS.map((t) => (
                      <SelectItem key={t.value} value={t.value}>
                        {t.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-4 sm:grid-cols-3">
                <div className="rounded-lg border p-4 text-center">
                  <p className="text-2xl font-bold">{resourcePlan.maxUsers.toLocaleString()}</p>
                  <p className="text-xs text-muted-foreground">Max Users</p>
                </div>
                <div className="rounded-lg border p-4 text-center">
                  <p className="text-2xl font-bold">{resourcePlan.maxCourses.toLocaleString()}</p>
                  <p className="text-xs text-muted-foreground">Max Courses</p>
                </div>
                <div className="rounded-lg border p-4 text-center">
                  <p className="text-2xl font-bold">{resourcePlan.storageQuotaGb} GB</p>
                  <p className="text-xs text-muted-foreground">Storage Quota</p>
                </div>
              </div>
              <div className="space-y-2">
                <Label>Features</Label>
                <div className="flex flex-wrap gap-2">
                  {[
                    'video_conferencing',
                    'analytics',
                    'api_access',
                    'custom_branding',
                    'sso',
                    'priority_support',
                  ].map((feature) => {
                    const active = resourcePlan.features.includes(feature)
                    return (
                      <button
                        key={feature}
                        type="button"
                        onClick={() =>
                          setResourcePlan((prev) => ({
                            ...prev,
                            features: active
                              ? prev.features.filter((f) => f !== feature)
                              : [...prev.features, feature],
                          }))
                        }
                        className={`rounded-full border px-3 py-1 text-xs font-medium transition-colors ${
                          active
                            ? 'border-primary bg-primary/10 text-primary'
                            : 'border-muted-foreground/30 text-muted-foreground hover:border-primary/50'
                        }`}
                      >
                        {feature.replace(/_/g, ' ')}
                      </button>
                    )
                  })}
                </div>
              </div>
            </div>
          )}

          {currentStep === 4 && (
            <div className="space-y-6">
              <div className="rounded-lg border p-4">
                <h4 className="mb-2 font-medium">Institution</h4>
                <dl className="grid gap-1 text-sm sm:grid-cols-2">
                  <div>
                    <dt className="text-muted-foreground">Name</dt>
                    <dd className="font-medium">{institution.name}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Location</dt>
                    <dd className="font-medium">
                      {institution.city}, {institution.country}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Timezone</dt>
                    <dd className="font-medium">{institution.timezone}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Website</dt>
                    <dd className="font-medium">{institution.website || 'N/A'}</dd>
                  </div>
                </dl>
              </div>
              <div className="rounded-lg border p-4">
                <h4 className="mb-2 font-medium">Admin User</h4>
                <dl className="grid gap-1 text-sm sm:grid-cols-2">
                  <div>
                    <dt className="text-muted-foreground">Name</dt>
                    <dd className="font-medium">
                      {adminUser.firstName} {adminUser.lastName}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Email</dt>
                    <dd className="font-medium">{adminUser.email}</dd>
                  </div>
                </dl>
              </div>
              <div className="rounded-lg border p-4">
                <h4 className="mb-2 font-medium">Resource Plan</h4>
                <dl className="grid gap-1 text-sm sm:grid-cols-3">
                  <div>
                    <dt className="text-muted-foreground">Tier</dt>
                    <dd className="font-medium capitalize">{resourcePlan.planTier}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Max Users</dt>
                    <dd className="font-medium">{resourcePlan.maxUsers.toLocaleString()}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Storage</dt>
                    <dd className="font-medium">{resourcePlan.storageQuotaGb} GB</dd>
                  </div>
                </dl>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Navigation */}
      <div className="flex items-center justify-between">
        <Button
          variant="outline"
          disabled={currentStep === 1}
          onClick={() => setCurrentStep((s) => s - 1)}
        >
          <ChevronLeft className="mr-1 h-4 w-4" />
          Back
        </Button>

        {currentStep < STEPS.length ? (
          <Button disabled={!canProceed()} onClick={() => setCurrentStep((s) => s + 1)}>
            Next
            <ChevronRight className="ml-1 h-4 w-4" />
          </Button>
        ) : (
          <Button disabled={startOnboarding.isPending} onClick={handleSubmit}>
            {startOnboarding.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Submit for Review
          </Button>
        )}
      </div>
    </div>
  )
}

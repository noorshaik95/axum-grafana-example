'use client'

import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { useState } from 'react'
import { useTenant, useUpdateTenantPlan } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ChevronLeft, Save, Loader2, Package, Users, BookOpen, HardDrive, Zap } from 'lucide-react'
import type { ResourcePlan } from '@/lib/api/types'

const PLAN_PRESETS: Record<string, ResourcePlan> = {
  starter: {
    planTier: 'starter',
    maxUsers: 100,
    maxCourses: 20,
    storageQuotaGb: 50,
    features: ['basic_courses', 'assignments', 'messaging'],
  },
  standard: {
    planTier: 'standard',
    maxUsers: 1000,
    maxCourses: 200,
    storageQuotaGb: 500,
    features: ['basic_courses', 'assignments', 'messaging', 'video_conferencing', 'analytics'],
  },
  enterprise: {
    planTier: 'enterprise',
    maxUsers: 50000,
    maxCourses: 10000,
    storageQuotaGb: 10000,
    features: [
      'basic_courses',
      'assignments',
      'messaging',
      'video_conferencing',
      'analytics',
      'sso',
      'api_access',
      'white_label',
    ],
  },
}

const ALL_FEATURES = [
  { key: 'basic_courses', label: 'Basic Courses' },
  { key: 'assignments', label: 'Assignments & Grading' },
  { key: 'messaging', label: 'In-Platform Messaging' },
  { key: 'video_conferencing', label: 'Video Conferencing' },
  { key: 'analytics', label: 'Analytics & Reports' },
  { key: 'sso', label: 'Single Sign-On (SSO)' },
  { key: 'api_access', label: 'API Access' },
  { key: 'white_label', label: 'White Labelling' },
  { key: 'priority_support', label: 'Priority Support' },
]

function tierColor(tier: string) {
  if (tier === 'enterprise') return 'bg-purple-100 text-purple-800 border-purple-200'
  if (tier === 'standard') return 'bg-blue-100 text-blue-800 border-blue-200'
  return 'bg-gray-100 text-gray-700 border-gray-200'
}

export default function UniversityPlanPage() {
  const params = useParams()
  const router = useRouter()
  const id = params.id as string
  const { data: tenant, isLoading } = useTenant(id)
  const updatePlan = useUpdateTenantPlan()

  const [plan, setPlan] = useState<ResourcePlan | null>(null)
  const [dirty, setDirty] = useState(false)

  const current =
    plan ??
    (tenant
      ? {
          planTier: tenant.plan as ResourcePlan['planTier'],
          maxUsers: tenant.maxUsers,
          maxCourses: 200,
          storageQuotaGb: tenant.storageQuotaGb,
          features: [],
        }
      : null)

  function applyPreset(tier: keyof typeof PLAN_PRESETS) {
    setPlan({ ...PLAN_PRESETS[tier] })
    setDirty(true)
  }

  function setField(field: keyof ResourcePlan, value: string | number | string[]) {
    if (!current) return
    setPlan({ ...current, [field]: value })
    setDirty(true)
  }

  function toggleFeature(key: string) {
    if (!current) return
    const features = current.features.includes(key)
      ? current.features.filter((f) => f !== key)
      : [...current.features, key]
    setPlan({ ...current, features })
    setDirty(true)
  }

  async function handleSave() {
    if (!current) return
    await updatePlan.mutateAsync({ id, plan: current })
    setDirty(false)
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!tenant || !current) {
    return <div className="py-20 text-center text-muted-foreground">University not found</div>
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link href={`/universities/${id}`}>
            <Button variant="ghost" size="icon">
              <ChevronLeft className="h-5 w-5" />
            </Button>
          </Link>
          <div>
            <h1 className="text-2xl font-bold">Resource Plan</h1>
            <p className="text-sm text-muted-foreground">{tenant.name}</p>
          </div>
        </div>
        <Button onClick={handleSave} disabled={!dirty || updatePlan.isPending}>
          {updatePlan.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Save className="mr-2 h-4 w-4" />
          )}
          Save Changes
        </Button>
      </div>

      {/* Plan Tier Presets */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Package className="h-4 w-4" /> Plan Tier
          </CardTitle>
          <CardDescription>Select a preset or customize individual limits below.</CardDescription>
        </CardHeader>
        <CardContent className="grid grid-cols-3 gap-4">
          {(['starter', 'standard', 'enterprise'] as const).map((tier) => (
            <button
              key={tier}
              onClick={() => applyPreset(tier)}
              className={`rounded-xl border-2 p-4 text-left transition-all hover:shadow-md ${
                current.planTier === tier ? 'border-primary shadow-sm' : 'border-border'
              }`}
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-semibold capitalize">{tier}</span>
                <Badge variant="outline" className={tierColor(tier)}>
                  {tier}
                </Badge>
              </div>
              <p className="text-xs text-muted-foreground">
                {PLAN_PRESETS[tier].maxUsers.toLocaleString()} users ·{' '}
                {PLAN_PRESETS[tier].storageQuotaGb}GB storage
              </p>
            </button>
          ))}
        </CardContent>
      </Card>

      {/* Resource Limits */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Resource Limits</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-6">
          <div className="space-y-2">
            <Label className="flex items-center gap-2">
              <Users className="h-4 w-4" /> Max Users
            </Label>
            <Input
              type="number"
              value={current.maxUsers}
              onChange={(e) => setField('maxUsers', parseInt(e.target.value) || 0)}
            />
          </div>
          <div className="space-y-2">
            <Label className="flex items-center gap-2">
              <BookOpen className="h-4 w-4" /> Max Courses
            </Label>
            <Input
              type="number"
              value={current.maxCourses}
              onChange={(e) => setField('maxCourses', parseInt(e.target.value) || 0)}
            />
          </div>
          <div className="space-y-2">
            <Label className="flex items-center gap-2">
              <HardDrive className="h-4 w-4" /> Storage Quota (GB)
            </Label>
            <Input
              type="number"
              value={current.storageQuotaGb}
              onChange={(e) => setField('storageQuotaGb', parseInt(e.target.value) || 0)}
            />
          </div>
        </CardContent>
      </Card>

      {/* Feature Flags */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Zap className="h-4 w-4" /> Feature Flags
          </CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-3">
          {ALL_FEATURES.map((feat) => (
            <label
              key={feat.key}
              className="flex items-center gap-3 cursor-pointer p-3 rounded-lg border hover:bg-muted/50 transition-colors"
            >
              <input
                type="checkbox"
                checked={current.features.includes(feat.key)}
                onChange={() => toggleFeature(feat.key)}
                className="h-4 w-4 accent-primary"
              />
              <span className="text-sm">{feat.label}</span>
            </label>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}

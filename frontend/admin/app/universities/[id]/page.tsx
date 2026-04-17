'use client'

import { useParams } from 'next/navigation'
import Link from 'next/link'
import { useTenant, useTenantUsage } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { formatDate } from '@/lib/utils'
import {
  ChevronLeft,
  Users,
  BookOpen,
  HardDrive,
  Globe,
  CreditCard,
  Settings,
  UserCog,
  Loader2,
} from 'lucide-react'

export default function UniversityProfilePage() {
  const params = useParams()
  const id = params.id as string
  const { data: tenant, isLoading } = useTenant(id)
  const { data: usage } = useTenantUsage(id)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!tenant) {
    return <div className="py-20 text-center text-muted-foreground">University not found</div>
  }

  const storagePercent =
    tenant.storageQuotaGb > 0 ? (tenant.storageUsedGb / tenant.storageQuotaGb) * 100 : 0
  const userPercent = tenant.maxUsers > 0 ? (tenant.currentUsers / tenant.maxUsers) * 100 : 0

  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/universities"
          className="mb-2 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
        >
          <ChevronLeft className="h-4 w-4" />
          Back to Universities
        </Link>
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">{tenant.name}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{tenant.slug}.slate.local</p>
          </div>
          <Badge variant={tenant.status === 'active' ? 'success' : 'warning'} className="text-sm">
            {tenant.status}
          </Badge>
        </div>
      </div>

      {/* Quick nav */}
      <div className="flex gap-2">
        <Button variant="outline" size="sm" asChild>
          <Link href={`/universities/${id}/plan`}>
            <Settings className="mr-2 h-4 w-4" />
            Resource Plan
          </Link>
        </Button>
        <Button variant="outline" size="sm" asChild>
          <Link href={`/universities/${id}/billing`}>
            <CreditCard className="mr-2 h-4 w-4" />
            Billing
          </Link>
        </Button>
        <Button variant="outline" size="sm" asChild>
          <Link href={`/universities/${id}/users`}>
            <UserCog className="mr-2 h-4 w-4" />
            Users
          </Link>
        </Button>
      </div>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium text-muted-foreground">Users</p>
              <Users className="h-4 w-4 text-muted-foreground" />
            </div>
            <p className="mt-2 text-2xl font-bold">{tenant.currentUsers.toLocaleString()}</p>
            <div className="mt-2">
              <Progress value={userPercent} className="h-1.5" />
              <p className="mt-1 text-xs text-muted-foreground">
                of {tenant.maxUsers.toLocaleString()} max
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium text-muted-foreground">Courses</p>
              <BookOpen className="h-4 w-4 text-muted-foreground" />
            </div>
            <p className="mt-2 text-2xl font-bold">
              {usage?.totalCourses?.toLocaleString() ?? '--'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium text-muted-foreground">Storage</p>
              <HardDrive className="h-4 w-4 text-muted-foreground" />
            </div>
            <p className="mt-2 text-2xl font-bold">{tenant.storageUsedGb.toFixed(1)} GB</p>
            <div className="mt-2">
              <Progress value={storagePercent} className="h-1.5" />
              <p className="mt-1 text-xs text-muted-foreground">
                of {tenant.storageQuotaGb} GB quota
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium text-muted-foreground">Domain</p>
              <Globe className="h-4 w-4 text-muted-foreground" />
            </div>
            <p className="mt-2 text-lg font-bold">{tenant.domain || 'N/A'}</p>
          </CardContent>
        </Card>
      </div>

      {/* Details */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Details</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-3">
            <div>
              <dt className="text-muted-foreground">Plan</dt>
              <dd className="font-medium capitalize">{tenant.plan}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Admin Email</dt>
              <dd className="font-medium">{tenant.adminEmail}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Created</dt>
              <dd className="font-medium">{formatDate(tenant.createdAt)}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Last Updated</dt>
              <dd className="font-medium">{formatDate(tenant.updatedAt)}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Active Users (DAU)</dt>
              <dd className="font-medium">{usage?.activeUsers?.toLocaleString() ?? '--'}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">API Calls Today</dt>
              <dd className="font-medium">{usage?.apiCallsToday?.toLocaleString() ?? '--'}</dd>
            </div>
          </dl>
        </CardContent>
      </Card>
    </div>
  )
}

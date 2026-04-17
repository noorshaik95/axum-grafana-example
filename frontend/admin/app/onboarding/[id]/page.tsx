'use client'

import { useParams } from 'next/navigation'
import Link from 'next/link'
import {
  useOnboardingStatus,
  useApproveOnboarding,
  useRejectOnboarding,
} from '@/lib/hooks/use-admin-queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useToast } from '@/hooks/use-toast'
import type { OnboardingStatus } from '@/lib/api/types'
import { ChevronLeft, Check, X, Clock, AlertCircle, Loader2 } from 'lucide-react'

const STATUS_CONFIG: Record<
  OnboardingStatus,
  {
    variant: 'default' | 'success' | 'warning' | 'destructive' | 'secondary' | 'info'
    label: string
  }
> = {
  draft: { variant: 'secondary', label: 'Draft' },
  pending_review: { variant: 'warning', label: 'Pending Review' },
  approved: { variant: 'info', label: 'Approved' },
  provisioning: { variant: 'default', label: 'Provisioning' },
  active: { variant: 'success', label: 'Active' },
  failed: { variant: 'destructive', label: 'Failed' },
  rejected: { variant: 'destructive', label: 'Rejected' },
}

const STEP_ICONS: Record<string, React.ElementType> = {
  completed: Check,
  in_progress: Loader2,
  pending: Clock,
  failed: AlertCircle,
}

export default function OnboardingDetailPage() {
  const params = useParams()
  const id = params.id as string
  const { data, isLoading } = useOnboardingStatus(id)
  const approve = useApproveOnboarding()
  const reject = useRejectOnboarding()
  const { toast } = useToast()

  const handleApprove = async () => {
    try {
      await approve.mutateAsync(id)
      toast({ title: 'Onboarding approved' })
    } catch {
      toast({ title: 'Failed to approve', variant: 'destructive' })
    }
  }

  const handleReject = async () => {
    const reason = window.prompt('Reason for rejection:')
    if (!reason) return
    try {
      await reject.mutateAsync({ id, reason })
      toast({ title: 'Onboarding rejected' })
    } catch {
      toast({ title: 'Failed to reject', variant: 'destructive' })
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!data) {
    return <div className="py-20 text-center text-muted-foreground">Onboarding job not found</div>
  }

  const { job, steps } = data
  const statusConfig = STATUS_CONFIG[job.status]

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
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">{job.institutionName}</h1>
            <p className="mt-1 text-sm text-muted-foreground">Onboarding ID: {job.id}</p>
          </div>
          <Badge variant={statusConfig.variant} className="text-sm">
            {statusConfig.label}
          </Badge>
        </div>
      </div>

      {/* Actions for pending review */}
      {job.status === 'pending_review' && (
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <p className="flex-1 text-sm font-medium">
              This onboarding is awaiting review. Approve or reject to continue.
            </p>
            <Button
              onClick={handleApprove}
              disabled={approve.isPending}
              className="bg-green-600 hover:bg-green-700"
            >
              {approve.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Check className="mr-2 h-4 w-4" />
              )}
              Approve
            </Button>
            <Button variant="destructive" onClick={handleReject} disabled={reject.isPending}>
              {reject.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <X className="mr-2 h-4 w-4" />
              )}
              Reject
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Error info */}
      {job.failureReason && (
        <Card className="border-destructive">
          <CardContent className="flex items-start gap-3 p-4">
            <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-destructive" />
            <div>
              <p className="font-medium text-destructive">Failure Reason</p>
              <p className="text-sm text-muted-foreground">{job.failureReason}</p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Steps */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Onboarding Steps</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {steps.map((step) => {
              const Icon = STEP_ICONS[step.status] ?? Clock
              const isActive = step.status === 'in_progress'
              return (
                <div
                  key={step.step}
                  className={`flex items-start gap-4 rounded-lg border p-4 ${
                    isActive ? 'border-primary bg-primary/5' : ''
                  } ${step.status === 'failed' ? 'border-destructive bg-destructive/5' : ''}`}
                >
                  <div
                    className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${
                      step.status === 'completed'
                        ? 'bg-green-100 text-green-600'
                        : step.status === 'in_progress'
                          ? 'bg-primary/10 text-primary'
                          : step.status === 'failed'
                            ? 'bg-destructive/10 text-destructive'
                            : 'bg-muted text-muted-foreground'
                    }`}
                  >
                    <Icon className={`h-4 w-4 ${isActive ? 'animate-spin' : ''}`} />
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">
                        Step {step.step}: {step.title}
                      </span>
                      <Badge
                        variant={
                          step.status === 'completed'
                            ? 'success'
                            : step.status === 'failed'
                              ? 'destructive'
                              : 'secondary'
                        }
                        className="text-xs"
                      >
                        {step.status.replace('_', ' ')}
                      </Badge>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        </CardContent>
      </Card>

      {/* Meta info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Details</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid gap-3 text-sm sm:grid-cols-2">
            <div>
              <dt className="text-muted-foreground">Admin Email</dt>
              <dd className="font-medium">{job.adminEmail}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Created</dt>
              <dd className="font-medium">{new Date(job.createdAt).toLocaleString()}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Last Updated</dt>
              <dd className="font-medium">{new Date(job.updatedAt).toLocaleString()}</dd>
            </div>
            {job.approvedBy && (
              <div>
                <dt className="text-muted-foreground">Approved By</dt>
                <dd className="font-medium">{job.approvedBy}</dd>
              </div>
            )}
          </dl>
        </CardContent>
      </Card>
    </div>
  )
}

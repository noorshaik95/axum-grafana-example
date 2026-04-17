'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useOnboardingJobs } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Progress } from '@/components/ui/progress'
import type { OnboardingStatus } from '@/lib/api/types'
import { Plus, Eye, ChevronLeft, ChevronRight, Loader2, ClipboardList } from 'lucide-react'
import { formatRelativeTime } from '@/lib/utils'

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

export default function OnboardingListPage() {
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState<OnboardingStatus | 'all'>('all')
  const pageSize = 10

  const { data, isLoading } = useOnboardingJobs({
    page,
    pageSize,
    status: statusFilter === 'all' ? undefined : statusFilter,
  })

  const jobs = data?.data ?? []
  const totalPages = data?.totalPages ?? 1
  const total = data?.total ?? 0

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Onboarding</h1>
          <p className="mt-1 text-sm text-muted-foreground">Manage university onboarding jobs</p>
        </div>
        <Button asChild>
          <Link href="/onboarding/new">
            <Plus className="mr-2 h-4 w-4" />
            New Onboarding
          </Link>
        </Button>
      </div>

      <Card>
        <div className="flex items-center justify-between p-5">
          <Select
            value={statusFilter}
            onValueChange={(v) => {
              setStatusFilter(v as OnboardingStatus | 'all')
              setPage(1)
            }}
          >
            <SelectTrigger className="w-48">
              <SelectValue placeholder="All statuses" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All statuses</SelectItem>
              <SelectItem value="draft">Draft</SelectItem>
              <SelectItem value="pending_review">Pending Review</SelectItem>
              <SelectItem value="approved">Approved</SelectItem>
              <SelectItem value="provisioning">Provisioning</SelectItem>
              <SelectItem value="active">Active</SelectItem>
              <SelectItem value="failed">Failed</SelectItem>
              <SelectItem value="rejected">Rejected</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Institution</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Progress</TableHead>
                <TableHead>Admin Email</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 5 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 6 }).map((__, j) => (
                      <TableCell key={j}>
                        <div className="h-4 w-20 animate-pulse rounded bg-muted" />
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : jobs.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="py-12 text-center">
                    <ClipboardList className="mx-auto h-8 w-8 text-muted-foreground" />
                    <p className="mt-2 text-sm text-muted-foreground">No onboarding jobs found</p>
                  </TableCell>
                </TableRow>
              ) : (
                jobs.map((job) => {
                  const config = STATUS_CONFIG[job.status]
                  const progressPct =
                    job.totalSteps > 0 ? (job.currentStep / job.totalSteps) * 100 : 0
                  return (
                    <TableRow key={job.id}>
                      <TableCell className="font-medium">
                        <Link href={`/onboarding/${job.id}`} className="hover:underline">
                          {job.institutionName}
                        </Link>
                      </TableCell>
                      <TableCell>
                        <Badge variant={config.variant}>{config.label}</Badge>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Progress value={progressPct} className="h-2 w-20" />
                          <span className="text-xs text-muted-foreground">
                            {job.currentStep}/{job.totalSteps}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {job.adminEmail}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatRelativeTime(job.createdAt)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button variant="ghost" size="icon" className="h-8 w-8" asChild>
                          <Link href={`/onboarding/${job.id}`}>
                            <Eye className="h-4 w-4" />
                          </Link>
                        </Button>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
            </TableBody>
          </Table>

          {!isLoading && totalPages > 1 && (
            <div className="flex items-center justify-between border-t px-5 py-3">
              <span className="text-sm text-muted-foreground">
                Page {page} of {totalPages} ({total} total)
              </span>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => p - 1)}
                >
                  <ChevronLeft className="mr-1 h-4 w-4" />
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                  <ChevronRight className="ml-1 h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

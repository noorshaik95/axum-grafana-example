'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useTenants, useToggleTenantAccess } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useToast } from '@/hooks/use-toast'
import type { TenantStatus } from '@/lib/api/types'
import {
  Search,
  ChevronLeft,
  ChevronRight,
  Eye,
  Power,
  UserCog,
  Loader2,
  Building2,
} from 'lucide-react'

const STATUS_BADGE: Record<
  TenantStatus,
  { variant: 'default' | 'success' | 'warning' | 'destructive' | 'secondary'; label: string }
> = {
  active: { variant: 'success', label: 'Active' },
  suspended: { variant: 'destructive', label: 'Suspended' },
  pending: { variant: 'warning', label: 'Pending' },
  in_review: { variant: 'info' as 'default', label: 'In Review' },
}

export function UniversityList() {
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<TenantStatus | 'all'>('all')
  const [page, setPage] = useState(1)
  const pageSize = 10

  const { data, isLoading } = useTenants({
    page,
    pageSize,
    search: search || undefined,
    status: statusFilter === 'all' ? undefined : statusFilter,
  })

  const toggleAccess = useToggleTenantAccess()
  const { toast } = useToast()

  const handleToggle = async (id: string, currentlyActive: boolean) => {
    const enabled = !currentlyActive
    try {
      await toggleAccess.mutateAsync({
        id,
        enabled,
        reason: enabled ? 'Re-enabled by admin' : 'Suspended by admin',
      })
      toast({
        title: enabled ? 'University enabled' : 'University suspended',
      })
    } catch {
      toast({
        title: 'Action failed',
        variant: 'destructive',
      })
    }
  }

  const tenants = data?.data ?? []
  const totalPages = data?.totalPages ?? 1
  const total = data?.total ?? 0

  return (
    <Card>
      <div className="flex flex-col gap-3 p-5 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative w-full max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search universities..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setPage(1)
            }}
            className="pl-10"
          />
        </div>
        <Select
          value={statusFilter}
          onValueChange={(v) => {
            setStatusFilter(v as TenantStatus | 'all')
            setPage(1)
          }}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="All statuses" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="suspended">Suspended</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
            <SelectItem value="in_review">In Review</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>University</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Plan</TableHead>
              <TableHead>Users</TableHead>
              <TableHead>Storage</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 7 }).map((__, j) => (
                    <TableCell key={j}>
                      <div className="h-4 w-20 animate-pulse rounded bg-muted" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : tenants.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="py-12 text-center">
                  <Building2 className="mx-auto h-8 w-8 text-muted-foreground" />
                  <p className="mt-2 text-sm text-muted-foreground">No universities found</p>
                </TableCell>
              </TableRow>
            ) : (
              tenants.map((tenant) => {
                const badge = STATUS_BADGE[tenant.status]
                return (
                  <TableRow key={tenant.id}>
                    <TableCell>
                      <div>
                        <Link
                          href={`/universities/${tenant.id}`}
                          className="font-medium hover:underline"
                        >
                          {tenant.name}
                        </Link>
                        <p className="text-xs text-muted-foreground">{tenant.slug}.slate.local</p>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={badge.variant}>{badge.label}</Badge>
                    </TableCell>
                    <TableCell className="capitalize">{tenant.plan}</TableCell>
                    <TableCell>
                      {tenant.currentUsers.toLocaleString()} / {tenant.maxUsers.toLocaleString()}
                    </TableCell>
                    <TableCell>
                      {tenant.storageUsedGb.toFixed(1)} / {tenant.storageQuotaGb} GB
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(tenant.createdAt).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Button variant="ghost" size="icon" className="h-8 w-8" asChild>
                          <Link href={`/universities/${tenant.id}`}>
                            <Eye className="h-4 w-4" />
                          </Link>
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8"
                          onClick={() => handleToggle(tenant.id, tenant.status === 'active')}
                          disabled={toggleAccess.isPending}
                        >
                          <Power className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="icon" className="h-8 w-8" asChild>
                          <Link href={`/universities/${tenant.id}/users`}>
                            <UserCog className="h-4 w-4" />
                          </Link>
                        </Button>
                      </div>
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
  )
}

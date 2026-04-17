'use client'

import { useState } from 'react'
import { useAdminRoles, useUpdateRolePermissions } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ADMIN_PERMISSIONS } from '@/lib/api/iam'
import { Loader2, Save, Shield, Lock } from 'lucide-react'

const PERMISSION_LABELS: Record<string, { label: string; description: string }> = {
  manage_tenants: { label: 'Manage Tenants', description: 'Create, update, suspend universities' },
  manage_billing: { label: 'Manage Billing', description: 'View invoices, issue credits' },
  manage_users: { label: 'Manage Users', description: 'Invite, deactivate admin users' },
  manage_roles: { label: 'Manage Roles', description: 'Edit role permissions' },
  view_metrics: { label: 'View Metrics', description: 'View platform analytics' },
  view_audit: { label: 'View Audit Log', description: 'Read audit trail' },
  impersonate: { label: 'Impersonate', description: 'Impersonate university users' },
  manage_system: { label: 'Manage System', description: 'System health, alerts, Kafka' },
  manage_onboarding: { label: 'Manage Onboarding', description: 'Approve/reject onboarding jobs' },
  manage_content: { label: 'Manage Content', description: 'Access content management' },
}

export default function IAMRolesPage() {
  const { data: roles, isLoading } = useAdminRoles()
  const updatePermissions = useUpdateRolePermissions()
  const [pendingChanges, setPendingChanges] = useState<Record<string, Set<string>>>({})
  const [saving, setSaving] = useState<string | null>(null)

  function getPermissionsForRole(roleId: string, originalPerms: string[]): string[] {
    if (pendingChanges[roleId]) {
      return Array.from(pendingChanges[roleId])
    }
    return originalPerms
  }

  function togglePermission(roleId: string, perm: string, original: string[]) {
    const current = new Set(getPermissionsForRole(roleId, original))
    if (current.has(perm)) current.delete(perm)
    else current.add(perm)
    setPendingChanges((prev) => ({ ...prev, [roleId]: current }))
  }

  async function saveRole(roleId: string) {
    const perms = pendingChanges[roleId]
    if (!perms) return
    setSaving(roleId)
    await updatePermissions.mutateAsync({ roleId, permissions: Array.from(perms) })
    setSaving(null)
    setPendingChanges((prev) => {
      const n = { ...prev }
      delete n[roleId]
      return n
    })
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  const roleList = roles ?? []

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Roles & Permissions</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Configure what each admin role can do across the platform.
        </p>
      </div>

      <div className="overflow-x-auto rounded-xl border bg-card shadow-sm">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/30">
              <th className="py-3 pl-4 text-left font-medium text-muted-foreground min-w-[200px]">
                Permission
              </th>
              {roleList.map((role) => (
                <th key={role.id} className="px-4 py-3 text-center font-medium min-w-[140px]">
                  <div className="flex flex-col items-center gap-1">
                    <div className="flex items-center gap-1.5">
                      <Shield className="h-3.5 w-3.5 text-muted-foreground" />
                      <span>{role.name}</span>
                      {role.isSystem && <Lock className="h-3 w-3 text-muted-foreground" />}
                    </div>
                    <Badge variant="outline" className="text-xs">
                      {role.userCount} users
                    </Badge>
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y">
            {ADMIN_PERMISSIONS.map((perm) => {
              const info = PERMISSION_LABELS[perm]
              return (
                <tr key={perm} className="hover:bg-muted/20 transition-colors">
                  <td className="py-3 pl-4">
                    <p className="font-medium">{info?.label ?? perm}</p>
                    <p className="text-xs text-muted-foreground">{info?.description}</p>
                  </td>
                  {roleList.map((role) => {
                    const perms = getPermissionsForRole(role.id, role.permissions)
                    const checked = perms.includes(perm)
                    return (
                      <td key={role.id} className="px-4 py-3 text-center">
                        <input
                          type="checkbox"
                          checked={checked}
                          disabled={role.isSystem}
                          onChange={() => togglePermission(role.id, perm, role.permissions)}
                          className="h-4 w-4 accent-primary disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed"
                        />
                      </td>
                    )
                  })}
                </tr>
              )
            })}
          </tbody>
        </table>

        {/* Save buttons per role */}
        <div className="flex justify-end gap-3 border-t p-4">
          {roleList
            .filter((r) => pendingChanges[r.id] && !r.isSystem)
            .map((role) => (
              <Button
                key={role.id}
                size="sm"
                onClick={() => saveRole(role.id)}
                disabled={saving === role.id}
              >
                {saving === role.id ? (
                  <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Save className="mr-2 h-3.5 w-3.5" />
                )}
                Save {role.name}
              </Button>
            ))}
          {roleList.filter((r) => pendingChanges[r.id]).length === 0 && (
            <p className="text-sm text-muted-foreground">No pending changes</p>
          )}
        </div>
      </div>

      {/* Role Cards */}
      <div className="grid grid-cols-2 gap-4">
        {roleList.map((role) => (
          <Card key={role.id}>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <CardTitle className="text-base flex items-center gap-2">
                  <Shield className="h-4 w-4" />
                  {role.name}
                  {role.isSystem && <Lock className="h-3.5 w-3.5 text-muted-foreground" />}
                </CardTitle>
                <Badge variant="outline">{role.userCount} users</Badge>
              </div>
              <CardDescription>{role.description}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-1.5">
                {role.permissions.map((p) => (
                  <span key={p} className="rounded-full bg-muted px-2 py-0.5 text-xs">
                    {PERMISSION_LABELS[p]?.label ?? p}
                  </span>
                ))}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}

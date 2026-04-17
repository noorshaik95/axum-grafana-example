'use client'

import { useState } from 'react'
import { useAdminRoles, useUpdateRolePermissions } from '@/lib/hooks/use-admin-queries'
import { ADMIN_PERMISSIONS } from '@/lib/api/iam'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useToast } from '@/hooks/use-toast'
import { Loader2, Save, Lock } from 'lucide-react'

export function RolePermissionMatrix() {
  const { data: roles, isLoading } = useAdminRoles()
  const updatePermissions = useUpdateRolePermissions()
  const { toast } = useToast()
  const [pendingChanges, setPendingChanges] = useState<Record<string, string[]>>({})

  const togglePermission = (roleId: string, permission: string) => {
    setPendingChanges((prev) => {
      const currentRole = roles?.find((r) => r.id === roleId)
      if (!currentRole) return prev

      const currentPerms = prev[roleId] ?? [...currentRole.permissions]
      const updated = currentPerms.includes(permission)
        ? currentPerms.filter((p) => p !== permission)
        : [...currentPerms, permission]

      return { ...prev, [roleId]: updated }
    })
  }

  const getPermissions = (roleId: string): string[] => {
    if (pendingChanges[roleId]) return pendingChanges[roleId]
    return roles?.find((r) => r.id === roleId)?.permissions ?? []
  }

  const hasPendingChanges = Object.keys(pendingChanges).length > 0

  const handleSave = async () => {
    try {
      await Promise.all(
        Object.entries(pendingChanges).map(([roleId, permissions]) =>
          updatePermissions.mutateAsync({ roleId, permissions })
        )
      )
      setPendingChanges({})
      toast({ title: 'Permissions updated successfully' })
    } catch (err) {
      toast({
        title: 'Failed to update permissions',
        description: err instanceof Error ? err.message : 'An error occurred',
        variant: 'destructive',
      })
    }
  }

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-base">Role-Permission Matrix</CardTitle>
        {hasPendingChanges && (
          <Button size="sm" onClick={handleSave} disabled={updatePermissions.isPending}>
            {updatePermissions.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Save className="mr-2 h-4 w-4" />
            )}
            Save Changes
          </Button>
        )}
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b">
                <th className="p-3 text-left font-medium text-muted-foreground">Role</th>
                {ADMIN_PERMISSIONS.map((perm) => (
                  <th key={perm} className="p-3 text-center font-medium text-muted-foreground">
                    <span className="block text-xs whitespace-nowrap">
                      {perm.replace(/_/g, ' ')}
                    </span>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {roles?.map((role) => (
                <tr key={role.id} className="border-b last:border-0">
                  <td className="p-3">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{role.name}</span>
                      {role.isSystem && (
                        <Badge variant="secondary" className="text-xs">
                          <Lock className="mr-1 h-3 w-3" />
                          System
                        </Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {role.userCount} user{role.userCount !== 1 ? 's' : ''}
                    </p>
                  </td>
                  {ADMIN_PERMISSIONS.map((perm) => {
                    const hasPermission = getPermissions(role.id).includes(perm)
                    const hasChanged =
                      pendingChanges[role.id] !== undefined &&
                      hasPermission !== role.permissions.includes(perm)
                    return (
                      <td key={perm} className="p-3 text-center">
                        <button
                          type="button"
                          disabled={role.isSystem}
                          onClick={() => togglePermission(role.id, perm)}
                          className={`inline-flex h-5 w-5 items-center justify-center rounded border transition-colors ${
                            hasPermission
                              ? 'border-primary bg-primary text-primary-foreground'
                              : 'border-muted-foreground/30 bg-transparent'
                          } ${
                            hasChanged ? 'ring-2 ring-yellow-400' : ''
                          } ${role.isSystem ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer hover:border-primary/50'}`}
                        >
                          {hasPermission && (
                            <svg className="h-3 w-3" viewBox="0 0 12 12" fill="none">
                              <path
                                d="M2 6L5 9L10 3"
                                stroke="currentColor"
                                strokeWidth="2"
                                strokeLinecap="round"
                                strokeLinejoin="round"
                              />
                            </svg>
                          )}
                        </button>
                      </td>
                    )
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  )
}

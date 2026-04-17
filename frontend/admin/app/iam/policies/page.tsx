'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useToast } from '@/hooks/use-toast'
import { Plus, Shield, Edit, Trash2, Users } from 'lucide-react'

const MOCK_POLICIES = [
  {
    id: '1',
    name: 'Full Access',
    description: 'Complete access to all admin and onboarding services',
    permissions: [
      { service: 'onboarding', actions: ['*'] },
      { service: 'admin', actions: ['*'] },
      { service: 'billing', actions: ['*'] },
      { service: 'analytics', actions: ['*'] },
      { service: 'iam', actions: ['*'] },
    ],
    users: 1,
    created_at: '2024-01-01',
  },
  {
    id: '2',
    name: 'Onboarding Admin',
    description: 'Full control over user onboarding and integrations',
    permissions: [
      { service: 'onboarding', actions: ['create', 'read', 'update', 'delete', 'sync'] },
      { service: 'analytics', actions: ['read'] },
    ],
    users: 2,
    created_at: '2024-01-05',
  },
  {
    id: '3',
    name: 'Billing Admin',
    description: 'Manage billing, payments, and subscriptions',
    permissions: [
      { service: 'billing', actions: ['create', 'read', 'update'] },
      { service: 'admin', actions: ['read'] },
      { service: 'analytics', actions: ['read'] },
    ],
    users: 1,
    created_at: '2024-01-10',
  },
  {
    id: '4',
    name: 'Usage Analytics',
    description: 'Read-only access to usage analytics and reports',
    permissions: [
      { service: 'analytics', actions: ['read'] },
      { service: 'admin', actions: ['read'] },
    ],
    users: 3,
    created_at: '2024-01-12',
  },
  {
    id: '5',
    name: 'Billing Read',
    description: 'View billing information and invoices',
    permissions: [{ service: 'billing', actions: ['read'] }],
    users: 1,
    created_at: '2024-01-15',
  },
]

export default function IAMPoliciesPage() {
  const [policies, setPolicies] = useState(MOCK_POLICIES)
  const { toast } = useToast()

  const handleDeletePolicy = (id: string, name: string) => {
    setPolicies(policies.filter((policy) => policy.id !== id))
    toast({
      title: 'Policy Deleted',
      description: `${name} policy has been removed`,
    })
  }

  const getServiceBadge = (service: string) => {
    const colors: Record<string, string> = {
      onboarding: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
      admin: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
      billing: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
      analytics: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
      iam: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
    }
    return colors[service] || ''
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">IAM Policies</h1>
          <p className="text-muted-foreground">
            Define custom access permissions for admin services
          </p>
        </div>
        <Button>
          <Plus className="mr-2 h-4 w-4" />
          Create Policy
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Total Policies</CardDescription>
            <CardTitle className="text-3xl">{policies.length}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Services</CardDescription>
            <CardTitle className="text-3xl">5</CardTitle>
            <p className="text-xs text-muted-foreground">
              Onboarding, Admin, Billing, Analytics, IAM
            </p>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Total Users</CardDescription>
            <CardTitle className="text-3xl">
              {policies.reduce((sum, p) => sum + p.users, 0)}
            </CardTitle>
            <p className="text-xs text-muted-foreground">With assigned policies</p>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Most Used</CardDescription>
            <CardTitle className="text-lg">Usage Analytics</CardTitle>
            <p className="text-xs text-muted-foreground">3 users</p>
          </CardHeader>
        </Card>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        {policies.map((policy) => (
          <Card key={policy.id}>
            <CardHeader>
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded-lg bg-primary/10">
                    <Shield className="h-5 w-5 text-primary" />
                  </div>
                  <div>
                    <CardTitle className="text-lg">{policy.name}</CardTitle>
                    <CardDescription className="text-sm">{policy.description}</CardDescription>
                  </div>
                </div>
                <Badge variant="outline" className="flex items-center gap-1">
                  <Users className="h-3 w-3" />
                  {policy.users}
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <div className="text-sm font-medium mb-2">Permissions</div>
                <div className="space-y-2">
                  {policy.permissions.map((perm, idx) => (
                    <div key={idx} className="flex items-start gap-2">
                      <Badge className={getServiceBadge(perm.service)}>{perm.service}</Badge>
                      <div className="flex flex-wrap gap-1">
                        {perm.actions.map((action, aidx) => (
                          <Badge key={aidx} variant="outline" className="text-xs">
                            {action}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="flex items-center justify-between pt-2 border-t">
                <span className="text-xs text-muted-foreground">
                  Created {new Date(policy.created_at).toLocaleDateString()}
                </span>
                <div className="flex gap-2">
                  <Button variant="ghost" size="sm">
                    <Edit className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDeletePolicy(policy.id, policy.name)}
                    disabled={policy.users > 0}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Service Permissions</CardTitle>
          <CardDescription>Available actions for each service</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div>
              <Badge className={getServiceBadge('onboarding')}>Onboarding Service</Badge>
              <div className="mt-2 flex flex-wrap gap-1">
                {['create', 'read', 'update', 'delete', 'sync', 'configure'].map((action) => (
                  <Badge key={action} variant="outline" className="text-xs">
                    {action}
                  </Badge>
                ))}
              </div>
            </div>

            <div>
              <Badge className={getServiceBadge('admin')}>Admin Service</Badge>
              <div className="mt-2 flex flex-wrap gap-1">
                {['read', 'configure', 'manage_plans', 'manage_resources'].map((action) => (
                  <Badge key={action} variant="outline" className="text-xs">
                    {action}
                  </Badge>
                ))}
              </div>
            </div>

            <div>
              <Badge className={getServiceBadge('billing')}>Billing Service</Badge>
              <div className="mt-2 flex flex-wrap gap-1">
                {['read', 'create', 'update', 'manage_payment', 'export'].map((action) => (
                  <Badge key={action} variant="outline" className="text-xs">
                    {action}
                  </Badge>
                ))}
              </div>
            </div>

            <div>
              <Badge className={getServiceBadge('analytics')}>Analytics Service</Badge>
              <div className="mt-2 flex flex-wrap gap-1">
                {['read', 'export', 'create_reports'].map((action) => (
                  <Badge key={action} variant="outline" className="text-xs">
                    {action}
                  </Badge>
                ))}
              </div>
            </div>

            <div>
              <Badge className={getServiceBadge('iam')}>IAM Service</Badge>
              <div className="mt-2 flex flex-wrap gap-1">
                {['read', 'create', 'update', 'delete', 'assign'].map((action) => (
                  <Badge key={action} variant="outline" className="text-xs">
                    {action}
                  </Badge>
                ))}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

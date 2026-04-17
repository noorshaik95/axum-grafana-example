'use client'

import { useState } from 'react'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useToast } from '@/hooks/use-toast'
import { Database, CheckCircle2, Settings, Loader2, Plus, Cloud, Building2 } from 'lucide-react'

const MOCK_INTEGRATIONS = [
  {
    id: '1',
    type: 'ldap',
    name: 'University LDAP',
    enabled: true,
    status: 'connected',
    lastSync: '2 hours ago',
    users: 12543,
    icon: Database,
  },
  {
    id: '2',
    type: 'saml',
    name: 'Campus SSO',
    enabled: true,
    status: 'connected',
    lastSync: '30 minutes ago',
    users: 8432,
    icon: CheckCircle2,
  },
  {
    id: '3',
    type: 'google_workspace',
    name: 'Google Workspace',
    enabled: true,
    status: 'connected',
    lastSync: '1 hour ago',
    users: 15234,
    icon: Cloud,
  },
  {
    id: '4',
    type: 'microsoft_365',
    name: 'Microsoft 365',
    enabled: false,
    status: 'disconnected',
    lastSync: 'Never',
    users: 0,
    icon: Building2,
  },
]

export default function IntegrationsPage() {
  const [integrations, setIntegrations] = useState(MOCK_INTEGRATIONS)
  const [syncing, setSyncing] = useState<string | null>(null)
  const { toast } = useToast()

  const handleSync = (id: string) => {
    setSyncing(id)
    setTimeout(() => {
      setSyncing(null)
      toast({
        title: 'Sync Completed',
        description: 'Successfully synced users from the directory',
      })
    }, 2000)
  }

  const handleToggle = (id: string) => {
    setIntegrations(
      integrations.map((int) =>
        int.id === id
          ? { ...int, enabled: !int.enabled, status: !int.enabled ? 'connected' : 'disconnected' }
          : int
      )
    )
    toast({
      title: integrations.find((i) => i.id === id)?.enabled
        ? 'Integration Disabled'
        : 'Integration Enabled',
      description: 'Integration status updated successfully',
    })
  }

  return (
    <>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Integrations</h1>
            <p className="text-muted-foreground">
              Connect with LDAP, SAML, Google Workspace, and Microsoft 365
            </p>
          </div>
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            Add Integration
          </Button>
        </div>

        <div className="grid gap-6 md:grid-cols-2">
          {integrations.map((integration) => {
            const Icon = integration.icon
            return (
              <Card key={integration.id}>
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <div className="p-2 rounded-lg bg-primary/10">
                        <Icon className="h-6 w-6 text-primary" />
                      </div>
                      <div>
                        <CardTitle className="text-lg">{integration.name}</CardTitle>
                        <CardDescription className="text-sm">
                          {integration.type.toUpperCase()}
                        </CardDescription>
                      </div>
                    </div>
                    <Badge variant={integration.enabled ? 'success' : 'outline'}>
                      {integration.status}
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <div className="text-muted-foreground">Total Users</div>
                      <div className="font-medium">{integration.users.toLocaleString()}</div>
                    </div>
                    <div>
                      <div className="text-muted-foreground">Last Sync</div>
                      <div className="font-medium">{integration.lastSync}</div>
                    </div>
                  </div>

                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      className="flex-1"
                      onClick={() => handleSync(integration.id)}
                      disabled={!integration.enabled || syncing === integration.id}
                    >
                      {syncing === integration.id ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Syncing...
                        </>
                      ) : (
                        <>
                          <Database className="mr-2 h-4 w-4" />
                          Sync Now
                        </>
                      )}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleToggle(integration.id)}
                    >
                      {integration.enabled ? 'Disable' : 'Enable'}
                    </Button>
                    <Button variant="outline" size="sm">
                      <Settings className="h-4 w-4" />
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Integration Features</CardTitle>
            <CardDescription>Powerful directory synchronization capabilities</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-5 w-5 text-green-600" />
                  <span className="font-medium">LDAP Support</span>
                </div>
                <p className="text-sm text-muted-foreground">
                  Active Directory and OpenLDAP integration
                </p>
              </div>
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-5 w-5 text-green-600" />
                  <span className="font-medium">SAML 2.0</span>
                </div>
                <p className="text-sm text-muted-foreground">
                  Single Sign-On with campus identity providers
                </p>
              </div>
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-5 w-5 text-green-600" />
                  <span className="font-medium">Google Workspace</span>
                </div>
                <p className="text-sm text-muted-foreground">Seamless G Suite integration</p>
              </div>
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-5 w-5 text-green-600" />
                  <span className="font-medium">Microsoft 365</span>
                </div>
                <p className="text-sm text-muted-foreground">Azure AD and Office 365 sync</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </>
  )
}

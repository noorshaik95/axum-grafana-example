'use client'

import { Card, CardContent } from '../../../../shared/components/ui/card'
import { Badge } from '../../../../shared/components/ui/badge'
import { Bell, BellOff, AlertTriangle, CheckCircle2, Loader2 } from 'lucide-react'
import { useActiveAlerts } from '../../../lib/hooks/use-admin-queries'
import type { Alert, AlertSeverity } from '../../../lib/api/platform'

function severityBadge(s: AlertSeverity) {
  if (s === 'CRITICAL') return 'bg-red-100 text-red-800'
  if (s === 'WARNING') return 'bg-yellow-100 text-yellow-800'
  return 'bg-blue-100 text-blue-800'
}

function formatAlertType(t: string): string {
  return t.toLowerCase().replace(/_/g, ' ')
}

export default function AlertsPage() {
  const { data, isLoading, isError, error } = useActiveAlerts()

  const alerts: Alert[] = data?.alerts ?? []
  const criticalCount = alerts.filter((a) => a.is_active && a.severity === 'CRITICAL').length

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Active Alerts</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Threshold alerts currently firing across the platform.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Badge className="bg-red-100 text-red-800 text-sm px-3 py-1">
            {criticalCount} critical active
          </Badge>
        </div>
      </div>

      {isLoading && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading alerts…
        </div>
      )}

      {isError && (
        <Card>
          <CardContent className="pt-4 pb-4">
            <div className="flex items-center gap-2 text-sm text-red-600">
              <AlertTriangle className="h-4 w-4" />
              Failed to load alerts: {error instanceof Error ? error.message : 'unknown error'}
            </div>
          </CardContent>
        </Card>
      )}

      {!isLoading && !isError && alerts.length === 0 && (
        <Card>
          <CardContent className="pt-6 pb-6">
            <div className="flex items-center gap-2 text-sm text-green-600">
              <CheckCircle2 className="h-4 w-4" />
              No active alerts. All systems operational.
            </div>
          </CardContent>
        </Card>
      )}

      {!isLoading && !isError && alerts.length > 0 && (
        <div className="space-y-3">
          {alerts.map((alert) => (
            <Card key={alert.id} className={!alert.is_active ? 'opacity-60' : ''}>
              <CardContent className="pt-4 pb-4">
                <div className="flex items-center gap-4">
                  <div
                    className={`rounded-lg p-2 ${alert.is_active ? 'bg-primary/10' : 'bg-muted'}`}
                  >
                    {alert.is_active ? (
                      <Bell className="h-5 w-5 text-primary" />
                    ) : (
                      <BellOff className="h-5 w-5 text-muted-foreground" />
                    )}
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <p className="font-medium">{alert.title}</p>
                      <Badge className={`text-xs ${severityBadge(alert.severity)}`}>
                        {alert.severity.toLowerCase()}
                      </Badge>
                      {!alert.is_active && (
                        <Badge variant="outline" className="text-xs">
                          resolved
                        </Badge>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground">
                      <span className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
                        {formatAlertType(alert.type)}
                      </span>{' '}
                      current <span className="font-semibold">{alert.current_value}</span>
                      {' / threshold '}
                      <span className="font-semibold">{alert.threshold_value}</span>
                      {alert.tenant_name ? ` · ${alert.tenant_name}` : ''}
                    </p>
                    {alert.message && (
                      <p className="text-xs text-muted-foreground mt-0.5">{alert.message}</p>
                    )}
                    {alert.triggered_at && (
                      <p className="text-xs text-red-600 mt-0.5 flex items-center gap-1">
                        <AlertTriangle className="h-3 w-3" />
                        Triggered {new Date(alert.triggered_at).toLocaleString()}
                      </p>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}

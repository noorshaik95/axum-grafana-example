'use client'

import { useServiceHealth } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { Activity, Loader2 } from 'lucide-react'
import type { ServiceStatus } from '@/lib/api/types'

const statusConfig: Record<ServiceStatus, { label: string; color: string; dot: string }> = {
  healthy: {
    label: 'Healthy',
    color: 'bg-green-100 text-green-800',
    dot: 'bg-green-500',
  },
  degraded: {
    label: 'Degraded',
    color: 'bg-yellow-100 text-yellow-800',
    dot: 'bg-yellow-500',
  },
  down: {
    label: 'Down',
    color: 'bg-red-100 text-red-800',
    dot: 'bg-red-500',
  },
  unknown: {
    label: 'Unknown',
    color: 'bg-gray-100 text-gray-800',
    dot: 'bg-gray-500',
  },
}

export function ServiceHealthGrid() {
  const { data: services, isLoading } = useServiceHealth()

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Activity className="h-4 w-4" />
            Service Health
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Activity className="h-4 w-4" />
          Service Health
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-3 sm:grid-cols-2">
          {services?.map((service) => {
            const config = statusConfig[service.status]
            return (
              <div
                key={service.name}
                className="flex items-center justify-between rounded-lg border p-3"
              >
                <div className="flex items-center gap-3">
                  <div className={cn('h-2.5 w-2.5 rounded-full', config.dot)} />
                  <div>
                    <p className="text-sm font-medium">{service.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {service.latencyMs}ms / {service.uptime.toFixed(1)}% uptime
                    </p>
                  </div>
                </div>
                <Badge className={cn('text-xs', config.color)}>{config.label}</Badge>
              </div>
            )
          })}
          {(!services || services.length === 0) && (
            <p className="col-span-2 py-4 text-center text-sm text-muted-foreground">
              No services reporting
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

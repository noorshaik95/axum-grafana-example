'use client'

import { useServiceHealth, usePlatformStats, useKafkaLag } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { formatDate } from '@/lib/utils'
import {
  Activity,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  Clock,
  Cpu,
  RefreshCw,
  Loader2,
} from 'lucide-react'
import type { ServiceHealth } from '@/lib/api/types'

function StatusIcon({ status }: { status: ServiceHealth['status'] }) {
  if (status === 'healthy') return <CheckCircle2 className="h-5 w-5 text-green-500" />
  if (status === 'degraded') return <AlertTriangle className="h-5 w-5 text-yellow-500" />
  if (status === 'down') return <XCircle className="h-5 w-5 text-red-500" />
  return <Clock className="h-5 w-5 text-gray-400" />
}

function StatusBadge({ status }: { status: ServiceHealth['status'] }) {
  const map = {
    healthy: 'bg-green-100 text-green-800',
    degraded: 'bg-yellow-100 text-yellow-800',
    down: 'bg-red-100 text-red-800',
    unknown: 'bg-gray-100 text-gray-600',
  }
  return (
    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${map[status]}`}>{status}</span>
  )
}

function latencyColor(ms: number) {
  if (ms < 100) return 'text-green-600'
  if (ms < 500) return 'text-yellow-600'
  return 'text-red-600'
}

export default function SystemHealthPage() {
  const { data: services, isLoading: loadingServices, dataUpdatedAt } = useServiceHealth()
  const { data: stats } = usePlatformStats()
  const { data: kafkaLag } = useKafkaLag()

  const healthy = (services ?? []).filter((s) => s.status === 'healthy').length
  const degraded = (services ?? []).filter((s) => s.status === 'degraded').length
  const down = (services ?? []).filter((s) => s.status === 'down').length

  const totalLag = (kafkaLag ?? []).reduce((sum, k) => sum + k.lag, 0)

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">System Health</h1>
          <p className="mt-1 text-sm text-muted-foreground flex items-center gap-2">
            <RefreshCw className="h-3.5 w-3.5 animate-spin" />
            Auto-refreshes every 15s
            {dataUpdatedAt > 0 && (
              <span className="ml-2">
                · Last updated {new Date(dataUpdatedAt).toLocaleTimeString()}
              </span>
            )}
          </p>
        </div>
        <div className="flex gap-3">
          <Badge className="bg-green-100 text-green-800 text-sm px-3 py-1">
            ✓ {healthy} healthy
          </Badge>
          {degraded > 0 && (
            <Badge className="bg-yellow-100 text-yellow-800 text-sm px-3 py-1">
              ⚠ {degraded} degraded
            </Badge>
          )}
          {down > 0 && (
            <Badge className="bg-red-100 text-red-800 text-sm px-3 py-1">✕ {down} down</Badge>
          )}
        </div>
      </div>

      {/* Platform Stats Row */}
      {stats && (
        <div className="grid grid-cols-4 gap-4">
          {[
            { label: 'Active Tenants', value: stats.activeTenants.toLocaleString() },
            { label: 'Daily Active Users', value: stats.dailyActiveUsers.toLocaleString() },
            { label: 'Avg Response Time', value: `${stats.avgResponseMs}ms` },
            { label: 'Error Rate', value: `${(stats.errorRate * 100).toFixed(2)}%` },
          ].map(({ label, value }) => (
            <Card key={label}>
              <CardContent className="pt-5">
                <p className="text-sm text-muted-foreground">{label}</p>
                <p className="text-2xl font-bold mt-1">{value}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Service Health Grid */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Activity className="h-4 w-4" />
            Service Status
          </CardTitle>
        </CardHeader>
        <CardContent>
          {loadingServices ? (
            <div className="flex justify-center py-10">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-3">
              {(services ?? []).map((svc) => (
                <div
                  key={svc.name}
                  className={`flex items-center gap-4 rounded-xl border p-4 ${
                    svc.status === 'down'
                      ? 'border-red-200 bg-red-50'
                      : svc.status === 'degraded'
                        ? 'border-yellow-200 bg-yellow-50'
                        : 'bg-card'
                  }`}
                >
                  <StatusIcon status={svc.status} />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between">
                      <p className="font-medium truncate">{svc.name}</p>
                      <StatusBadge status={svc.status} />
                    </div>
                    <div className="mt-1 flex items-center gap-4 text-xs text-muted-foreground">
                      <span className={latencyColor(svc.latencyMs)}>{svc.latencyMs}ms</span>
                      <span>{(svc.uptime * 100).toFixed(2)}% uptime</span>
                      <span>{(svc.errorRate * 100).toFixed(2)}% errors</span>
                    </div>
                    <p className="text-xs text-muted-foreground mt-0.5">v{svc.version}</p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Kafka Consumer Lag */}
      {kafkaLag && kafkaLag.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center justify-between">
              <span className="flex items-center gap-2">
                <Cpu className="h-4 w-4" />
                Kafka Consumer Lag
              </span>
              <span
                className={`text-sm font-normal ${totalLag > 10000 ? 'text-red-600' : totalLag > 1000 ? 'text-yellow-600' : 'text-green-600'}`}
              >
                Total: {totalLag.toLocaleString()} msgs
              </span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-muted-foreground">
                  <th className="pb-3 text-left font-medium">Topic</th>
                  <th className="pb-3 text-left font-medium">Consumer Group</th>
                  <th className="pb-3 text-right font-medium">Partition</th>
                  <th className="pb-3 text-right font-medium">Offset</th>
                  <th className="pb-3 text-right font-medium">Lag</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {kafkaLag.map((k, i) => (
                  <tr key={i}>
                    <td className="py-2 font-mono text-xs">{k.topic}</td>
                    <td className="py-2 text-muted-foreground">{k.consumerGroup}</td>
                    <td className="py-2 text-right text-muted-foreground">{k.partition}</td>
                    <td className="py-2 text-right text-muted-foreground">
                      {k.currentOffset.toLocaleString()}
                    </td>
                    <td
                      className={`py-2 text-right font-medium ${k.lag > 1000 ? 'text-red-600' : k.lag > 100 ? 'text-yellow-600' : 'text-green-600'}`}
                    >
                      {k.lag.toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

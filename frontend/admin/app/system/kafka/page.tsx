'use client'

import { useKafkaLag } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { RefreshCw, Cpu, TrendingDown, AlertTriangle, CheckCircle2 } from 'lucide-react'

function lagSeverity(lag: number): { color: string; label: string } {
  if (lag === 0) return { color: 'text-green-600', label: 'clear' }
  if (lag < 100) return { color: 'text-green-500', label: 'low' }
  if (lag < 1000) return { color: 'text-yellow-600', label: 'moderate' }
  if (lag < 10000) return { color: 'text-orange-600', label: 'high' }
  return { color: 'text-red-600', label: 'critical' }
}

function LagBar({ lag, max }: { lag: number; max: number }) {
  const pct = max > 0 ? Math.min(100, (lag / max) * 100) : 0
  const color = pct > 80 ? 'bg-red-500' : pct > 50 ? 'bg-yellow-500' : 'bg-green-500'
  return (
    <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
      <div className={`h-full rounded-full transition-all ${color}`} style={{ width: `${pct}%` }} />
    </div>
  )
}

export default function KafkaPage() {
  const { data: consumers, isLoading, dataUpdatedAt } = useKafkaLag()

  const rows = consumers ?? []
  const maxLag = Math.max(...rows.map((r) => r.lag), 1)
  const totalLag = rows.reduce((s, r) => s + r.lag, 0)
  const criticalCount = rows.filter((r) => r.lag > 10000).length
  const healthyCount = rows.filter((r) => r.lag < 100).length

  // Group by topic
  const byTopic = rows.reduce<Record<string, typeof rows>>((acc, row) => {
    ;(acc[row.topic] = acc[row.topic] ?? []).push(row)
    return acc
  }, {})

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Kafka Monitor</h1>
          <p className="mt-1 text-sm text-muted-foreground flex items-center gap-2">
            <RefreshCw className="h-3.5 w-3.5 animate-spin" />
            Auto-refreshes every 10s
            {dataUpdatedAt > 0 && (
              <span>· Last updated {new Date(dataUpdatedAt).toLocaleTimeString()}</span>
            )}
          </p>
        </div>
        <div className="flex gap-3">
          <Badge className="bg-green-100 text-green-800 text-sm px-3 py-1">
            <CheckCircle2 className="mr-1.5 h-3.5 w-3.5" />
            {healthyCount} clear
          </Badge>
          {criticalCount > 0 && (
            <Badge className="bg-red-100 text-red-800 text-sm px-3 py-1">
              <AlertTriangle className="mr-1.5 h-3.5 w-3.5" />
              {criticalCount} critical
            </Badge>
          )}
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-3 gap-4">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <div className="rounded-lg bg-primary/10 p-3">
                <Cpu className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Total Lag</p>
                <p className={`text-2xl font-bold ${lagSeverity(totalLag).color}`}>
                  {totalLag.toLocaleString()}
                </p>
                <p className="text-xs text-muted-foreground">messages behind</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <div className="rounded-lg bg-blue-100 p-3">
                <TrendingDown className="h-5 w-5 text-blue-600" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Topics Monitored</p>
                <p className="text-2xl font-bold">{Object.keys(byTopic).length}</p>
                <p className="text-xs text-muted-foreground">{rows.length} partitions total</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <div
                className={`rounded-lg p-3 ${criticalCount > 0 ? 'bg-red-100' : 'bg-green-100'}`}
              >
                {criticalCount > 0 ? (
                  <AlertTriangle className="h-5 w-5 text-red-600" />
                ) : (
                  <CheckCircle2 className="h-5 w-5 text-green-600" />
                )}
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Status</p>
                <p
                  className={`text-2xl font-bold ${criticalCount > 0 ? 'text-red-600' : 'text-green-600'}`}
                >
                  {criticalCount > 0 ? 'Backlog' : 'Healthy'}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Per-Topic Cards */}
      {isLoading ? (
        <div className="grid grid-cols-2 gap-4">
          {[0, 1, 2, 3].map((i) => (
            <Card key={i}>
              <CardContent className="pt-6">
                <div className="h-20 animate-pulse rounded-lg bg-muted" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : rows.length === 0 ? (
        <Card>
          <CardContent className="py-20 text-center text-muted-foreground">
            No consumer lag data available
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {Object.entries(byTopic).map(([topic, partitions]) => {
            const topicLag = partitions.reduce((s, p) => s + p.lag, 0)
            const { color, label } = lagSeverity(topicLag)
            return (
              <Card key={topic}>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base flex items-center justify-between">
                    <span className="font-mono text-sm">{topic}</span>
                    <div className="flex items-center gap-3">
                      <span className={`text-sm font-semibold ${color}`}>
                        {topicLag.toLocaleString()} lag
                      </span>
                      <Badge variant="outline" className={`text-xs ${color}`}>
                        {label}
                      </Badge>
                    </div>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    {partitions.map((p, i) => {
                      const { color: pc } = lagSeverity(p.lag)
                      return (
                        <div key={i} className="space-y-1">
                          <div className="flex items-center justify-between text-xs text-muted-foreground">
                            <span>
                              {p.consumerGroup} · partition {p.partition}
                            </span>
                            <span className={pc}>{p.lag.toLocaleString()} behind</span>
                          </div>
                          <LagBar lag={p.lag} max={maxLag} />
                          <p className="text-xs text-muted-foreground">
                            offset {p.currentOffset.toLocaleString()} /{' '}
                            {p.endOffset.toLocaleString()}
                          </p>
                        </div>
                      )
                    })}
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}

'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Bell, BellOff, Plus, Trash2, AlertTriangle, CheckCircle2, X } from 'lucide-react'

type AlertSeverity = 'critical' | 'warning' | 'info'
type AlertCondition = 'greater_than' | 'less_than' | 'equals'

interface AlertRule {
  id: string
  name: string
  metric: string
  condition: AlertCondition
  threshold: number
  severity: AlertSeverity
  enabled: boolean
  notifyEmail: string
  lastTriggered?: string
}

// Static alert rules (would come from API in production)
const DEFAULT_RULES: AlertRule[] = [
  {
    id: '1',
    name: 'High Error Rate',
    metric: 'error_rate',
    condition: 'greater_than',
    threshold: 0.05,
    severity: 'critical',
    enabled: true,
    notifyEmail: 'ops@slate.io',
    lastTriggered: '2026-04-15T10:32:00Z',
  },
  {
    id: '2',
    name: 'Service Down',
    metric: 'service_health',
    condition: 'equals',
    threshold: 0,
    severity: 'critical',
    enabled: true,
    notifyEmail: 'ops@slate.io',
  },
  {
    id: '3',
    name: 'Kafka Lag Critical',
    metric: 'kafka_consumer_lag',
    condition: 'greater_than',
    threshold: 10000,
    severity: 'warning',
    enabled: true,
    notifyEmail: 'ops@slate.io',
  },
  {
    id: '4',
    name: 'High Avg Latency',
    metric: 'avg_response_ms',
    condition: 'greater_than',
    threshold: 2000,
    severity: 'warning',
    enabled: false,
    notifyEmail: 'ops@slate.io',
  },
]

const METRICS = [
  'error_rate',
  'service_health',
  'kafka_consumer_lag',
  'avg_response_ms',
  'active_tenants',
  'cpu_usage',
  'memory_usage',
  'storage_used_pct',
]

function severityBadge(s: AlertSeverity) {
  if (s === 'critical') return 'bg-red-100 text-red-800'
  if (s === 'warning') return 'bg-yellow-100 text-yellow-800'
  return 'bg-blue-100 text-blue-800'
}

export default function AlertsPage() {
  const [rules, setRules] = useState<AlertRule[]>(DEFAULT_RULES)
  const [showNew, setShowNew] = useState(false)
  const [form, setForm] = useState<Partial<AlertRule>>({
    severity: 'warning',
    condition: 'greater_than',
    enabled: true,
  })

  function toggleRule(id: string) {
    setRules((rs) => rs.map((r) => (r.id === id ? { ...r, enabled: !r.enabled } : r)))
  }

  function deleteRule(id: string) {
    setRules((rs) => rs.filter((r) => r.id !== id))
  }

  function addRule() {
    if (!form.name || !form.metric || form.threshold === undefined || !form.notifyEmail) return
    setRules((rs) => [...rs, { ...form, id: Date.now().toString() } as AlertRule])
    setForm({ severity: 'warning', condition: 'greater_than', enabled: true })
    setShowNew(false)
  }

  const criticalCount = rules.filter((r) => r.enabled && r.severity === 'critical').length

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Alert Rules</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Configure threshold alerts for platform metrics.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Badge className="bg-red-100 text-red-800 text-sm px-3 py-1">
            {criticalCount} critical active
          </Badge>
          <Button onClick={() => setShowNew(true)}>
            <Plus className="mr-2 h-4 w-4" /> New Rule
          </Button>
        </div>
      </div>

      {/* Alert Rules List */}
      <div className="space-y-3">
        {rules.map((rule) => (
          <Card key={rule.id} className={!rule.enabled ? 'opacity-60' : ''}>
            <CardContent className="pt-4 pb-4">
              <div className="flex items-center gap-4">
                <div className={`rounded-lg p-2 ${rule.enabled ? 'bg-primary/10' : 'bg-muted'}`}>
                  {rule.enabled ? (
                    <Bell className="h-5 w-5 text-primary" />
                  ) : (
                    <BellOff className="h-5 w-5 text-muted-foreground" />
                  )}
                </div>
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    <p className="font-medium">{rule.name}</p>
                    <Badge className={`text-xs ${severityBadge(rule.severity)}`}>
                      {rule.severity}
                    </Badge>
                    {!rule.enabled && (
                      <Badge variant="outline" className="text-xs">
                        disabled
                      </Badge>
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground">
                    <span className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
                      {rule.metric}
                    </span>{' '}
                    {rule.condition.replace('_', ' ')}{' '}
                    <span className="font-semibold">{rule.threshold}</span>
                    {' → '}
                    {rule.notifyEmail}
                  </p>
                  {rule.lastTriggered && (
                    <p className="text-xs text-red-600 mt-0.5 flex items-center gap-1">
                      <AlertTriangle className="h-3 w-3" />
                      Last triggered {new Date(rule.lastTriggered).toLocaleString()}
                    </p>
                  )}
                  {!rule.lastTriggered && rule.enabled && (
                    <p className="text-xs text-green-600 mt-0.5 flex items-center gap-1">
                      <CheckCircle2 className="h-3 w-3" />
                      Never triggered
                    </p>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  <Button variant="outline" size="sm" onClick={() => toggleRule(rule.id)}>
                    {rule.enabled ? <BellOff className="h-4 w-4" /> : <Bell className="h-4 w-4" />}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-red-600 hover:text-red-700 hover:bg-red-50"
                    onClick={() => deleteRule(rule.id)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* New Rule Modal */}
      {showNew && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="w-full max-w-lg rounded-2xl bg-background p-6 shadow-xl">
            <div className="flex items-center justify-between mb-5">
              <h2 className="text-lg font-semibold">New Alert Rule</h2>
              <Button variant="ghost" size="icon" onClick={() => setShowNew(false)}>
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Rule Name</Label>
                <Input
                  placeholder="e.g. High Error Rate"
                  value={form.name ?? ''}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Metric</Label>
                  <select
                    className="w-full rounded-lg border bg-background px-3 py-2 text-sm"
                    value={form.metric ?? ''}
                    onChange={(e) => setForm({ ...form, metric: e.target.value })}
                  >
                    <option value="">Select metric…</option>
                    {METRICS.map((m) => (
                      <option key={m} value={m}>
                        {m}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="space-y-2">
                  <Label>Condition</Label>
                  <select
                    className="w-full rounded-lg border bg-background px-3 py-2 text-sm"
                    value={form.condition ?? 'greater_than'}
                    onChange={(e) =>
                      setForm({ ...form, condition: e.target.value as AlertCondition })
                    }
                  >
                    <option value="greater_than">Greater than</option>
                    <option value="less_than">Less than</option>
                    <option value="equals">Equals</option>
                  </select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Threshold</Label>
                  <Input
                    type="number"
                    placeholder="0"
                    value={form.threshold ?? ''}
                    onChange={(e) => setForm({ ...form, threshold: parseFloat(e.target.value) })}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Severity</Label>
                  <select
                    className="w-full rounded-lg border bg-background px-3 py-2 text-sm"
                    value={form.severity ?? 'warning'}
                    onChange={(e) =>
                      setForm({ ...form, severity: e.target.value as AlertSeverity })
                    }
                  >
                    <option value="critical">Critical</option>
                    <option value="warning">Warning</option>
                    <option value="info">Info</option>
                  </select>
                </div>
              </div>
              <div className="space-y-2">
                <Label>Notify Email</Label>
                <Input
                  type="email"
                  placeholder="ops@example.com"
                  value={form.notifyEmail ?? ''}
                  onChange={(e) => setForm({ ...form, notifyEmail: e.target.value })}
                />
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <Button variant="outline" onClick={() => setShowNew(false)}>
                  Cancel
                </Button>
                <Button onClick={addRule}>Create Rule</Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

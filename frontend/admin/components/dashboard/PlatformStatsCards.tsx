'use client'

import { usePlatformStats } from '@/lib/hooks/use-admin-queries'
import { Card, CardContent } from '@/components/ui/card'
import { Building2, Users, UserCheck, HardDrive, AlertTriangle, Timer, Loader2 } from 'lucide-react'

interface StatCardProps {
  title: string
  value: string | number
  icon: React.ElementType
  subtitle: string
  loading?: boolean
  valueColor?: string
}

function StatCard({ title, value, icon: Icon, subtitle, loading, valueColor }: StatCardProps) {
  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <p className="text-sm font-medium text-muted-foreground">{title}</p>
          <Icon className="h-4 w-4 text-muted-foreground" />
        </div>
        <div className="mt-2">
          {loading ? (
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          ) : (
            <p className={`text-2xl font-bold ${valueColor ?? 'text-foreground'}`}>{value}</p>
          )}
          <p className="mt-1 text-xs text-muted-foreground">{subtitle}</p>
        </div>
      </CardContent>
    </Card>
  )
}

export function PlatformStatsCards() {
  const { data: stats, isLoading } = usePlatformStats()

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
      <StatCard
        title="Active Tenants"
        icon={Building2}
        value={stats?.activeTenants ?? 0}
        subtitle="Registered institutions"
        loading={isLoading}
      />
      <StatCard
        title="Total Users"
        icon={Users}
        value={stats?.totalUsers?.toLocaleString() ?? '0'}
        subtitle="Across all tenants"
        loading={isLoading}
      />
      <StatCard
        title="Daily Active Users"
        icon={UserCheck}
        value={stats?.dailyActiveUsers?.toLocaleString() ?? '0'}
        subtitle="Last 24 hours"
        loading={isLoading}
      />
      <StatCard
        title="Storage Used"
        icon={HardDrive}
        value={`${stats?.storageUsedTb?.toFixed(1) ?? '0'} TB`}
        subtitle="Total platform storage"
        loading={isLoading}
      />
      <StatCard
        title="Error Rate"
        icon={AlertTriangle}
        value={`${(stats?.errorRate ?? 0).toFixed(2)}%`}
        subtitle="Last hour"
        loading={isLoading}
        valueColor={(stats?.errorRate ?? 0) > 1 ? 'text-red-600' : 'text-green-600'}
      />
      <StatCard
        title="Avg Response"
        icon={Timer}
        value={`${stats?.avgResponseMs ?? 0}ms`}
        subtitle="P50 latency"
        loading={isLoading}
      />
    </div>
  )
}

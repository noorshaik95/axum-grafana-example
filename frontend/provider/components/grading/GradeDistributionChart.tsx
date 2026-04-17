'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Loader2 } from 'lucide-react'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
  ResponsiveContainer,
} from 'recharts'
import { useGradeDistribution } from '@/lib/api/hooks'

interface GradeDistributionChartProps {
  courseId: string
  assignmentId?: string
}

export function GradeDistributionChart({ courseId, assignmentId }: GradeDistributionChartProps) {
  const { data: distribution, isLoading, error } = useGradeDistribution(courseId, assignmentId)

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
        </CardContent>
      </Card>
    )
  }

  if (error || !distribution) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <p className="text-sm text-slate-500">Grade distribution data unavailable.</p>
        </CardContent>
      </Card>
    )
  }

  const chartData = distribution.buckets.map((b) => ({
    range: b.range,
    count: b.count,
  }))

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Grade Distribution</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-4 gap-4 mb-4 text-center">
          <div className="rounded-lg bg-slate-50 p-2">
            <p className="text-lg font-bold text-slate-900">{distribution.mean.toFixed(1)}</p>
            <p className="text-xs text-slate-500">Mean</p>
          </div>
          <div className="rounded-lg bg-slate-50 p-2">
            <p className="text-lg font-bold text-slate-900">{distribution.median.toFixed(1)}</p>
            <p className="text-xs text-slate-500">Median</p>
          </div>
          <div className="rounded-lg bg-slate-50 p-2">
            <p className="text-lg font-bold text-slate-900">{distribution.p25.toFixed(1)}</p>
            <p className="text-xs text-slate-500">P25</p>
          </div>
          <div className="rounded-lg bg-slate-50 p-2">
            <p className="text-lg font-bold text-slate-900">{distribution.p75.toFixed(1)}</p>
            <p className="text-xs text-slate-500">P75</p>
          </div>
        </div>

        <ResponsiveContainer width="100%" height={250}>
          <BarChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
            <XAxis dataKey="range" tick={{ fontSize: 12, fill: '#64748b' }} />
            <YAxis tick={{ fontSize: 12, fill: '#64748b' }} allowDecimals={false} />
            <Tooltip
              contentStyle={{
                backgroundColor: '#fff',
                border: '1px solid #e2e8f0',
                borderRadius: '8px',
                fontSize: '13px',
              }}
            />
            <ReferenceLine
              y={distribution.mean}
              stroke="#6366f1"
              strokeDasharray="4 4"
              label="Mean"
            />
            <Bar dataKey="count" fill="#6366f1" radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}

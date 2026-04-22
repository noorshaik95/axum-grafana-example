'use client'

import { useMemo, useState } from 'react'
import Link from 'next/link'
import {
  TrendingUp,
  CheckSquare,
  BarChart2,
  AlertTriangle,
  ArrowRight,
  Loader2,
} from 'lucide-react'
import { useInstructorCourses, useCourseAnalytics, useGradeDistribution } from '@/lib/api/hooks'

function formatPct(n: number | null | undefined): string {
  if (n == null || Number.isNaN(n)) return '--'
  return `${Math.round(n)}%`
}

export default function AnalyticsPage() {
  const coursesQuery = useInstructorCourses({ pageSize: 50 })
  const courses = coursesQuery.data?.data ?? []
  const [courseId, setCourseId] = useState<string>('')

  const effectiveCourseId = courseId || courses[0]?.id || ''
  const selectedCourse = courses.find((c) => c.id === effectiveCourseId)

  const analyticsQuery = useCourseAnalytics(effectiveCourseId)
  const distributionQuery = useGradeDistribution(effectiveCourseId)

  // Derive a "where students struggle" list from the distribution buckets —
  // the closest real signal we have until a dedicated topic-struggle endpoint
  // is exposed. Each bucket below 70% is shown as a struggle row.
  const struggleBuckets = useMemo(() => {
    const buckets = distributionQuery.data?.buckets ?? []
    const total = buckets.reduce((sum, b) => sum + b.count, 0)
    return buckets
      .filter((b) => {
        const match = b.range.match(/(\d+)/)
        const low = match ? Number(match[1]) : 100
        return low < 70
      })
      .map((b) => ({
        topic: `Grades ${b.range}`,
        pct: total > 0 ? Math.round((b.count / total) * 100) : 0,
        students: b.count,
      }))
  }, [distributionQuery.data])

  const maxPct = Math.max(1, ...struggleBuckets.map((t) => t.pct))

  if (coursesQuery.isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
      </div>
    )
  }

  if (coursesQuery.error) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-red-700">
        Failed to load courses.
      </div>
    )
  }

  if (courses.length === 0) {
    return (
      <div className="space-y-4">
        <div>
          <h1 className="font-serif text-2xl text-[#12170f]">Analytics</h1>
        </div>
        <div
          className="rounded-xl border px-5 py-10 text-center"
          style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
        >
          <p className="text-sm" style={{ color: '#6a6e62' }}>
            No courses yet. Create a course to see analytics.
          </p>
        </div>
      </div>
    )
  }

  const analytics = analyticsQuery.data

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <h1 className="font-serif text-2xl text-[#12170f]">Analytics</h1>
          <p className="text-sm mt-1" style={{ color: '#6a6e62' }}>
            {selectedCourse
              ? `${selectedCourse.title} · ${selectedCourse.term}`
              : 'Select a course'}
          </p>
        </div>
        <select
          value={effectiveCourseId}
          onChange={(e) => setCourseId(e.target.value)}
          className="rounded-lg border px-3 py-2 text-sm"
          style={{ background: '#ffffff', borderColor: '#e4e0d4', color: '#12170f' }}
        >
          {courses.map((c) => (
            <option key={c.id} value={c.id}>
              {c.title}
            </option>
          ))}
        </select>
      </div>

      {analyticsQuery.isLoading ? (
        <div className="flex items-center justify-center py-10">
          <Loader2 className="h-5 w-5 animate-spin text-indigo-600" />
        </div>
      ) : analyticsQuery.error ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          Failed to load analytics.
        </div>
      ) : (
        <div className="grid grid-cols-3 gap-3">
          {[
            {
              label: 'Engagement',
              value: formatPct(analytics?.engagement_pct),
              sub: 'students active this week',
              Icon: TrendingUp,
            },
            {
              label: 'Completion',
              value: formatPct(analytics?.completion_pct),
              sub: 'modules finished avg',
              Icon: CheckSquare,
            },
            {
              label: 'Median grade',
              value: formatPct(analytics?.median_grade ?? null),
              sub: 'across all assignments',
              Icon: BarChart2,
            },
          ].map((stat) => (
            <div
              key={stat.label}
              className="rounded-xl border p-5"
              style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
            >
              <div
                className="flex h-10 w-10 items-center justify-center rounded-lg mb-3"
                style={{ background: '#f2f7f3' }}
              >
                <stat.Icon className="h-5 w-5" style={{ color: '#234e32' }} />
              </div>
              <p className="font-serif text-2xl font-bold" style={{ color: '#234e32' }}>
                {stat.value}
              </p>
              <p className="text-sm font-medium text-[#12170f] mt-0.5">{stat.label}</p>
              <p className="text-xs mt-0.5" style={{ color: '#6a6e62' }}>
                {stat.sub}
              </p>
            </div>
          ))}
        </div>
      )}

      {/* Where students struggle — derived from grade distribution */}
      <div
        className="rounded-xl border overflow-hidden"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        <div
          className="px-5 py-3 border-b"
          style={{ background: '#f6f3ec', borderColor: '#e4e0d4' }}
        >
          <h2 className="text-sm font-semibold text-[#12170f]">Grade distribution — low buckets</h2>
          <p className="text-xs mt-0.5" style={{ color: '#6a6e62' }}>
            Share of students in below-70% buckets
          </p>
        </div>
        <div className="p-5 space-y-3">
          {distributionQuery.isLoading ? (
            <div className="flex items-center justify-center py-6">
              <Loader2 className="h-5 w-5 animate-spin text-indigo-600" />
            </div>
          ) : struggleBuckets.length === 0 ? (
            <p className="text-sm" style={{ color: '#6a6e62' }}>
              No students in below-70% buckets.
            </p>
          ) : (
            struggleBuckets.map((topic) => {
              const barWidth = (topic.pct / maxPct) * 100
              const barColor = topic.pct >= 40 ? '#d97757' : topic.pct >= 20 ? '#ffb648' : '#3e7d4f'
              return (
                <div key={topic.topic}>
                  <div className="flex items-center justify-between mb-1.5">
                    <span className="text-sm text-[#12170f]">{topic.topic}</span>
                    <div className="flex items-center gap-2">
                      <span className="text-xs" style={{ color: '#6a6e62' }}>
                        {topic.students} students
                      </span>
                      <span className="font-mono text-sm font-semibold" style={{ color: barColor }}>
                        {topic.pct}%
                      </span>
                    </div>
                  </div>
                  <div
                    className="h-2 rounded-full overflow-hidden"
                    style={{ background: '#f6f3ec' }}
                  >
                    <div
                      className="h-full rounded-full transition-all"
                      style={{ width: `${barWidth}%`, background: barColor }}
                    />
                  </div>
                </div>
              )
            })
          )}
        </div>
      </div>

      {/* At-risk callout driven by analytics.at_risk_count */}
      {analytics && analytics.at_risk_count > 0 && (
        <div
          className="rounded-xl border px-5 py-4 flex items-center gap-4"
          style={{ background: 'rgba(217,119,87,0.06)', borderColor: 'rgba(217,119,87,0.3)' }}
        >
          <AlertTriangle className="h-5 w-5 shrink-0" style={{ color: '#d97757' }} />
          <div className="flex-1">
            <p className="text-sm font-semibold text-[#12170f]">
              {analytics.at_risk_count} student{analytics.at_risk_count === 1 ? '' : 's'} at risk
            </p>
            <p className="text-xs mt-0.5" style={{ color: '#6a6e62' }}>
              Review the roster for outreach suggestions.
            </p>
          </div>
          <Link
            href="/roster"
            className="shrink-0 rounded-xl px-3 py-1.5 text-xs font-semibold transition-all hover:opacity-80"
            style={{ background: '#d97757', color: '#ffffff' }}
          >
            View roster
            <ArrowRight className="inline h-3 w-3 ml-1" />
          </Link>
        </div>
      )}
    </div>
  )
}

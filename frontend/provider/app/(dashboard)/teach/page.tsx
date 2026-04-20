'use client'

import Link from 'next/link'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, ArrowRight, ChevronRight, Clock, MessageSquare } from 'lucide-react'
import { Spinner } from '../../../../shared/components/ui'
import { EmptyStateIllustrated, RiskBadge } from '../../../../shared/components'
import { listCourses } from '../../../lib/api/courses'
import * as gradingApi from '../../../lib/api/grading'
import * as teachApi from '../../../lib/api/teach'
import * as rosterApi from '../../../lib/api/roster'
import type { Assignment } from '../../../../shared/lib/api/types'

interface TeachCard {
  id: string
  href: string
  icon: typeof Clock
  label: string
  detail: string
  tone: 'default' | 'urgent'
}

function formatLectureTime(startsAt: string, endsAt: string): string {
  try {
    const start = new Date(startsAt)
    const end = new Date(endsAt)
    const fmt = new Intl.DateTimeFormat(undefined, { hour: 'numeric', minute: '2-digit' })
    return `${fmt.format(start)} – ${fmt.format(end)}`
  } catch {
    return ''
  }
}

export default function TeachPage() {
  // First taught course — used to scope the at-risk callout + quick stats.
  const coursesQuery = useQuery({
    queryKey: ['provider-taught-courses'],
    queryFn: () => listCourses({ pageSize: 1 }),
  })
  const firstCourseId = coursesQuery.data?.data?.[0]?.id

  // Grading pressure — computed client-side from past-due assignments in the
  // first taught course. The gateway exposes `/api/grading/queue/count` but
  // the underlying gRPC method isn't wired yet (CARRYOVER §1a HTTP-passthrough
  // gap). Until gateway-proxy-expert lands option A, we approximate queue size
  // from assignment dueDates. Flip to `gradingApi.getQueueCount` once the real
  // endpoint is reachable.
  const assignmentsQuery = useQuery({
    queryKey: ['provider-assignments-first-course', firstCourseId],
    queryFn: () => gradingApi.listAssignments(firstCourseId!),
    enabled: !!firstCourseId,
  })

  const nextLectureQuery = useQuery({
    queryKey: ['provider-next-lecture'],
    queryFn: () => teachApi.getNextLecture(),
  })

  const threadsQuery = useQuery({
    queryKey: ['provider-threads-needing-reply'],
    queryFn: () => teachApi.getDiscussionsNeedingReplyCount(),
  })

  const atRiskQuery = useQuery({
    queryKey: ['provider-at-risk', firstCourseId],
    queryFn: () => rosterApi.getRosterHealth({ course: firstCourseId!, risk: 'at_risk', limit: 5 }),
    enabled: !!firstCourseId,
  })

  const isInitialLoading =
    coursesQuery.isLoading || nextLectureQuery.isLoading || threadsQuery.isLoading

  if (isInitialLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Spinner />
      </div>
    )
  }

  if (coursesQuery.isError && !coursesQuery.data) {
    return (
      <div className="rounded-card border border-warm-200 bg-paper p-6 text-sm text-red-700">
        Could not load your teaching dashboard. Refresh and try again.
      </div>
    )
  }

  // Approximate "needs grading" as assignments whose due date has passed for
  // the first taught course. Real count arrives when CARRYOVER §1a lands.
  const now = Date.now()
  const assignments: Assignment[] = assignmentsQuery.data ?? []
  const queueCount = assignments.filter((a) => {
    const due = Date.parse(a.dueDate)
    return Number.isFinite(due) && due < now
  }).length
  const nextLecture = nextLectureQuery.data
  const threadsCount = threadsQuery.data?.count ?? 0
  const threadsOldest = threadsQuery.data?.oldest_hours ?? null
  const atRiskEntries = atRiskQuery.data?.entries ?? []

  const cards: TeachCard[] = []
  if (nextLecture) {
    cards.push({
      id: 'lecture',
      href: '/teach',
      icon: Clock,
      label: `Next lecture · ${nextLecture.course_title}`,
      detail: `${formatLectureTime(nextLecture.starts_at, nextLecture.ends_at)} · ${
        nextLecture.location ?? (nextLecture.live ? 'Live now' : 'Online')
      }`,
      tone: 'default',
    })
  }
  if (threadsCount > 0) {
    cards.push({
      id: 'threads',
      href: '/discussion',
      icon: MessageSquare,
      label: 'Threads needing reply',
      detail:
        threadsOldest != null
          ? `${threadsCount} unanswered · oldest ${threadsOldest}h ago`
          : `${threadsCount} unanswered`,
      tone: 'urgent',
    })
  }
  cards.push({
    id: 'office-hours',
    href: '/office-hours',
    icon: Clock,
    label: 'Office hours',
    detail: 'Open schedule',
    tone: 'default',
  })
  if (atRiskEntries.length > 0) {
    cards.push({
      id: 'at-risk',
      href: '/roster',
      icon: AlertTriangle,
      label: 'At-risk students',
      detail: `${atRiskEntries.length} student${atRiskEntries.length === 1 ? '' : 's'} flagged`,
      tone: 'urgent',
    })
  }

  return (
    <div className="space-y-6">
      {/* Action Hero */}
      <div className="relative overflow-hidden rounded-card bg-gradient-to-br from-forest-900 via-forest-800 to-forest-700 px-8 py-10 text-paper">
        <div className="absolute -right-20 -top-20 h-64 w-64 rounded-full bg-amber-400/20 blur-3xl" />
        <div className="relative">
          <div className="mb-3 flex items-center gap-2">
            <span className="inline-flex items-center gap-1.5 rounded-badge bg-amber-400/20 px-3 py-1 font-mono text-xs font-medium text-amber-400">
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-current" />
              {queueCount} in queue
            </span>
          </div>
          <h1 className="mb-2 font-serif text-3xl leading-tight text-paper">
            {queueCount > 0
              ? `Grade ${queueCount} submission${queueCount === 1 ? '' : 's'} today`
              : 'Nothing urgent waiting'}
          </h1>
          <p className="mb-6 text-base text-paper/70">
            {queueCount > 0
              ? 'Pattern-grouped queue ready — one click per group.'
              : 'Take a breath. Review discussion threads or prep your next lecture.'}
          </p>
          <Link
            href="/grade"
            className="inline-flex items-center gap-2 rounded-button bg-amber-400 px-5 py-3 text-sm font-semibold text-warm-900 transition-all hover:scale-105"
          >
            {queueCount > 0 ? 'Open queue' : 'Browse assignments'}
            <ArrowRight className="h-4 w-4" />
          </Link>
        </div>
      </div>

      {/* Today */}
      <div>
        <h2 className="mb-3 font-serif text-xl text-warm-900">Today</h2>
        {cards.length === 0 ? (
          <EmptyStateIllustrated
            illustration="all-done"
            title="Nothing urgent"
            description="No lectures, threads, or flagged students right now."
          />
        ) : (
          <div className="space-y-2">
            {cards.map((card) => {
              const Icon = card.icon
              const isUrgent = card.tone === 'urgent'
              return (
                <Link
                  key={card.id}
                  href={card.href}
                  className={`group flex items-center gap-4 rounded-card border px-4 py-3.5 transition-all hover:shadow-slate-sm ${
                    isUrgent ? 'border-red-500/30 bg-red-100/30' : 'border-warm-200 bg-paper'
                  }`}
                >
                  <div
                    className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-button ${
                      isUrgent ? 'bg-red-100 text-red-700' : 'bg-forest-50 text-forest-700'
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-semibold text-warm-900">{card.label}</p>
                    <p className="mt-0.5 text-xs text-warm-700">{card.detail}</p>
                  </div>
                  <ChevronRight className="h-4 w-4 shrink-0 text-warm-700 opacity-0 transition-opacity group-hover:opacity-100" />
                </Link>
              )
            })}
          </div>
        )}
      </div>

      {/* At-risk callout */}
      {firstCourseId && atRiskEntries.length > 0 ? (
        <div className="rounded-card border border-red-500/30 bg-red-100/20 p-5">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="font-serif text-lg text-warm-900">At-risk students</h3>
            <Link
              href="/roster"
              className="text-xs font-medium text-forest-700 hover:text-forest-800"
            >
              Open roster →
            </Link>
          </div>
          <div className="space-y-2">
            {atRiskEntries.slice(0, 3).map((entry) => (
              <div
                key={entry.user_id}
                className="flex items-center gap-3 rounded-button bg-paper px-3 py-2"
              >
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-red-100 text-xs font-semibold text-red-700">
                  {entry.display_name
                    .split(' ')
                    .map((n) => n[0])
                    .slice(0, 2)
                    .join('')}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium text-warm-900">{entry.display_name}</p>
                  <p className="text-xs text-warm-700">{entry.suggested_action}</p>
                </div>
                <RiskBadge risk={entry.risk_level} />
              </div>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  )
}

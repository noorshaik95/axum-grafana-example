'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import {
  ArrowLeft,
  Settings,
  TrendingUp,
  AlertTriangle,
  GripVertical,
  ClipboardCheck,
  Loader2,
} from 'lucide-react'
import { useInstructorCourse } from '@/lib/api/hooks'

type TabId = 'modules' | 'roster' | 'gradebook' | 'settings'

const tabs: { id: TabId; label: string }[] = [
  { id: 'modules', label: 'Modules' },
  { id: 'roster', label: 'Roster' },
  { id: 'gradebook', label: 'Gradebook' },
  { id: 'settings', label: 'Settings' },
]

const mockModules = [
  {
    id: 1,
    title: 'Introduction to Functional Programming',
    lessons: 4,
    status: 'live' as const,
  },
  {
    id: 2,
    title: 'Recursion & Pattern Matching',
    lessons: 6,
    status: 'live' as const,
  },
  {
    id: 3,
    title: 'Higher-Order Functions',
    lessons: 5,
    status: 'draft' as const,
  },
  {
    id: 4,
    title: 'Monads & Effects',
    lessons: 7,
    status: 'scheduled' as const,
  },
]

const statusConfig = {
  live: { label: 'Live', bg: '#dde9df', color: '#234e32' },
  draft: { label: 'Draft', bg: '#f6f3ec', color: '#6a6e62' },
  scheduled: { label: 'Scheduled', bg: 'rgba(255,182,72,0.15)', color: '#c48d1a' },
}

export default function CourseDetailPage() {
  const params = useParams()
  const courseId = params.id as string
  const [activeTab, setActiveTab] = useState<TabId>('modules')
  const { data: course, isLoading, error } = useInstructorCourse(courseId)
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin" style={{ color: '#234e32' }} />
      </div>
    )
  }

  if (error || !course) {
    return (
      <div className="space-y-4">
        <Link
          href="/courses"
          className="flex items-center gap-1.5 text-sm transition-colors hover:text-[#234e32]"
          style={{ color: '#6a6e62' }}
        >
          <ArrowLeft className="h-4 w-4" />
          Back to courses
        </Link>
        <div
          className="rounded-xl border p-6"
          style={{ borderColor: '#e4e0d4', background: '#ffffff' }}
        >
          <p style={{ color: '#d97757' }}>Failed to load course.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-5">
      {/* Back */}
      <Link
        href="/courses"
        className="flex items-center gap-1.5 text-sm transition-colors hover:text-[#234e32]"
        style={{ color: '#6a6e62' }}
      >
        <ArrowLeft className="h-4 w-4" />
        Courses
      </Link>

      {/* Course header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <h1 className="font-serif text-2xl text-[#12170f]">{course.title}</h1>
            <span
              className="rounded-full px-2.5 py-0.5 text-xs font-mono font-medium"
              style={
                course.isPublished
                  ? { background: '#dde9df', color: '#234e32' }
                  : { background: '#f6f3ec', color: '#6a6e62' }
              }
            >
              {course.isPublished ? 'Published' : 'Draft'}
            </span>
          </div>
          <p className="text-sm" style={{ color: '#6a6e62' }}>
            {course.term}
            {course.metadata?.courseCode ? ` · ${course.metadata.courseCode}` : ''}
          </p>
        </div>
        <Link
          href={`/courses/${courseId}/settings`}
          className="flex items-center gap-2 rounded-xl border px-4 py-2 text-sm font-medium transition-colors hover:bg-[#f6f3ec]"
          style={{ borderColor: '#e4e0d4', color: '#12170f' }}
        >
          <Settings className="h-4 w-4" />
          Settings
        </Link>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-3 gap-3">
        {[
          {
            label: 'Pulse',
            value: '82%',
            sub: 'engagement',
            Icon: TrendingUp,
            color: '#234e32',
            bg: '#f2f7f3',
          },
          {
            label: 'Median grade',
            value: '78%',
            sub: 'last assignment',
            Icon: ClipboardCheck,
            color: '#234e32',
            bg: '#f2f7f3',
          },
          {
            label: 'At-risk',
            value: '2',
            sub: 'students flagged',
            Icon: AlertTriangle,
            color: '#d97757',
            bg: 'rgba(217,119,87,0.08)',
          },
        ].map((stat) => (
          <div
            key={stat.label}
            className="rounded-xl border p-4 flex items-center gap-3"
            style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
          >
            <div
              className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg"
              style={{ background: stat.bg }}
            >
              <stat.Icon className="h-5 w-5" style={{ color: stat.color }} />
            </div>
            <div>
              <p className="font-serif text-xl font-bold" style={{ color: stat.color }}>
                {stat.value}
              </p>
              <p className="text-xs font-medium text-[#12170f]">{stat.label}</p>
              <p className="text-xs" style={{ color: '#6a6e62' }}>
                {stat.sub}
              </p>
            </div>
          </div>
        ))}
      </div>

      {/* Segment tabs */}
      <div
        className="inline-flex items-center rounded-xl p-1 gap-1"
        style={{ background: '#f6f3ec' }}
      >
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className="rounded-lg px-4 py-1.5 text-sm font-medium transition-all"
            style={
              activeTab === tab.id
                ? { background: '#234e32', color: '#ffffff' }
                : { color: '#6a6e62' }
            }
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {activeTab === 'modules' && (
        <div className="space-y-2">
          {mockModules.map((mod) => {
            const cfg = statusConfig[mod.status]
            return (
              <div
                key={mod.id}
                className="flex items-center gap-4 rounded-xl border px-4 py-3.5"
                style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
              >
                <GripVertical
                  className="h-4 w-4 shrink-0 cursor-grab"
                  style={{ color: '#e4e0d4' }}
                />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-[#12170f]">{mod.title}</p>
                  <p className="text-xs mt-0.5" style={{ color: '#6a6e62' }}>
                    {mod.lessons} lessons
                  </p>
                </div>
                <span
                  className="rounded-full px-2.5 py-0.5 text-xs font-mono font-medium shrink-0"
                  style={{ background: cfg.bg, color: cfg.color }}
                >
                  {cfg.label}
                </span>
                <Link
                  href={`/courses/${courseId}/content`}
                  className="shrink-0 rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors hover:bg-[#f6f3ec]"
                  style={{ borderColor: '#e4e0d4', color: '#12170f' }}
                >
                  Edit
                </Link>
              </div>
            )
          })}
        </div>
      )}

      {activeTab === 'roster' && (
        <div
          className="rounded-xl border overflow-hidden"
          style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
        >
          <div
            className="px-4 py-3 border-b text-xs font-mono font-medium"
            style={{ background: '#f6f3ec', borderColor: '#e4e0d4', color: '#6a6e62' }}
          >
            47 STUDENTS ENROLLED
          </div>
          <div className="p-5 flex items-center justify-center" style={{ color: '#6a6e62' }}>
            <Link
              href="/roster"
              className="text-sm font-medium transition-colors hover:text-[#234e32]"
              style={{ color: '#234e32' }}
            >
              View full roster with risk signals →
            </Link>
          </div>
        </div>
      )}

      {activeTab === 'gradebook' && (
        <div
          className="rounded-xl border overflow-hidden"
          style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
        >
          <div
            className="px-4 py-3 border-b text-xs font-mono font-medium"
            style={{ background: '#f6f3ec', borderColor: '#e4e0d4', color: '#6a6e62' }}
          >
            GRADEBOOK
          </div>
          <div className="p-5 flex items-center justify-center" style={{ color: '#6a6e62' }}>
            <Link
              href="/grading"
              className="text-sm font-medium transition-colors hover:text-[#234e32]"
              style={{ color: '#234e32' }}
            >
              Open full gradebook →
            </Link>
          </div>
        </div>
      )}

      {activeTab === 'settings' && (
        <div
          className="rounded-xl border overflow-hidden"
          style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
        >
          <div className="p-5 flex items-center justify-center" style={{ color: '#6a6e62' }}>
            <Link
              href={`/courses/${courseId}/settings`}
              className="text-sm font-medium transition-colors hover:text-[#234e32]"
              style={{ color: '#234e32' }}
            >
              Open course settings →
            </Link>
          </div>
        </div>
      )}
    </div>
  )
}

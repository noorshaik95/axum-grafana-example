'use client';

import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { BookOpen, Video, FileText, CheckCircle2, Clock, Zap } from 'lucide-react';

// Mock data
const WEEK_DAYS = ['M', 'T', 'W', 'Th', 'F', 'Sa', 'Su'];
const TODAY_IDX = new Date().getDay(); // 0=Sun
const WEEK_MAP = [6, 0, 1, 2, 3, 4, 5]; // Sun->index6, Mon->0, etc.
const todayDotIdx = WEEK_MAP[TODAY_IDX];

interface TodayTask {
  id: string;
  icon: 'assignment' | 'video' | 'reading';
  title: string;
  subtitle: string;
  timeUntil: string;
  urgency: 'overdue' | 'soon' | 'upcoming' | 'done';
  href: string;
}

const MOCK_TASKS: TodayTask[] = [
  {
    id: '1',
    icon: 'assignment',
    title: 'Problem Set 4 — Recursion',
    subtitle: 'CS 101 · Draft saved at Q3',
    timeUntil: '2h left',
    urgency: 'soon',
    href: '/assignments',
  },
  {
    id: '2',
    icon: 'video',
    title: 'Algorithms — Live Class',
    subtitle: 'CS 201 · Prof. Martinez',
    timeUntil: 'Today 3pm',
    urgency: 'upcoming',
    href: '/video',
  },
  {
    id: '3',
    icon: 'reading',
    title: 'Ch. 7 Reading — Data Structures',
    subtitle: 'CS 201 · 22 pages',
    timeUntil: 'Due tomorrow',
    urgency: 'upcoming',
    href: '/courses',
  },
  {
    id: '4',
    icon: 'assignment',
    title: 'Lab 3 — Binary Trees',
    subtitle: 'CS 201 · Submitted',
    timeUntil: 'Done',
    urgency: 'done',
    href: '/assignments',
  },
];

const urgencyBadge: Record<string, { bg: string; color: string; label?: string }> = {
  overdue: { bg: '#fde8e6', color: '#c0392b' },
  soon: { bg: '#fff3cd', color: '#7a4900' },
  upcoming: { bg: 'var(--forest-50)', color: 'var(--forest-700)' },
  done: { bg: '#f0f0f0', color: 'var(--muted)' },
};

const IconMap = {
  assignment: FileText,
  video: Video,
  reading: BookOpen,
};

function useTodayData() {
  return useQuery({
    queryKey: ['today-data'],
    queryFn: async () => {
      // In production, fetch real data from assignments + video APIs
      return {
        heroTask: MOCK_TASKS[0],
        tasks: MOCK_TASKS,
        streakDays: 7,
        doneToday: 1,
        totalToday: 4,
      };
    },
  });
}

export default function TodayPage() {
  const { data } = useTodayData();
  const tasks = data?.tasks ?? MOCK_TASKS;
  const hero = data?.heroTask ?? MOCK_TASKS[0];
  const doneCount = tasks.filter((t) => t.urgency === 'done').length;

  if (tasks.filter((t) => t.urgency !== 'done').length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-24 gap-6">
        <span style={{ fontSize: 56 }}>🌱</span>
        <div className="text-center">
          <h2 className="serif text-2xl" style={{ color: 'var(--ink)' }}>
            Nothing due today.
          </h2>
          <p className="mt-2 text-sm" style={{ color: 'var(--muted)' }}>
            You&apos;re all caught up. Keep the momentum going.
          </p>
        </div>
        <div className="flex gap-3">
          <Link href="/assignments" className="btn-primary">
            Get ahead
          </Link>
          <Link href="/courses" className="btn-secondary">
            Browse courses
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-2xl mx-auto">
      {/* Action hero */}
      <div
        className="rounded-2xl p-6 sm:p-8"
        style={{
          background: 'linear-gradient(135deg, var(--forest-700) 0%, var(--forest-900) 100%)',
          color: '#e9efe9',
        }}
      >
        <p
          className="mono text-xs font-semibold mb-2 tracking-wide"
          style={{ color: 'var(--amber)' }}
        >
          ● Due {hero.timeUntil} · CS 101
        </p>
        <h2 className="serif text-2xl sm:text-3xl mb-2 leading-snug" style={{ color: '#f2f7f3' }}>
          {hero.title}
        </h2>
        <p className="text-sm mb-5" style={{ color: 'var(--forest-300)' }}>
          {hero.subtitle}
        </p>
        <Link
          href={hero.href}
          className="btn-amber inline-flex items-center gap-2"
          style={{ fontSize: 14 }}
        >
          Resume <span aria-hidden>→</span>
        </Link>
      </div>

      {/* Week spine */}
      <div
        className="rounded-xl p-4 flex items-center justify-between"
        style={{
          background: '#fff',
          border: '1px solid var(--border)',
          boxShadow: 'var(--shadow-sm)',
        }}
      >
        <span className="text-xs font-semibold" style={{ color: 'var(--muted)' }}>
          This week
        </span>
        <div className="flex items-center gap-3">
          {WEEK_DAYS.map((day, idx) => {
            const isDone = idx < todayDotIdx;
            const isToday = idx === todayDotIdx;
            return (
              <div key={day} className="flex flex-col items-center gap-1">
                <span
                  style={{
                    fontSize: 10,
                    fontWeight: isToday ? 700 : 400,
                    color: isToday ? 'var(--forest-700)' : 'var(--muted)',
                  }}
                >
                  {day}
                </span>
                <div
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: '50%',
                    background: isDone
                      ? 'var(--forest-500)'
                      : isToday
                        ? 'var(--amber)'
                        : 'var(--border)',
                    border: isToday ? '2px solid var(--amber)' : 'none',
                    boxShadow: isToday ? '0 0 0 2px rgba(255,182,72,0.25)' : 'none',
                  }}
                />
              </div>
            );
          })}
        </div>
        <div className="flex items-center gap-1 text-xs" style={{ color: 'var(--muted)' }}>
          <CheckCircle2 style={{ width: 13, height: 13, color: 'var(--forest-500)' }} />
          {doneCount}/{tasks.length}
        </div>
      </div>

      {/* Task list */}
      <div className="space-y-2">
        <h3
          className="text-xs font-semibold uppercase tracking-wider px-1"
          style={{ color: 'var(--muted)' }}
        >
          Today&apos;s work
        </h3>
        {tasks.map((task) => {
          const Icon = IconMap[task.icon];
          const badge = urgencyBadge[task.urgency];
          const isDone = task.urgency === 'done';
          return (
            <Link
              key={task.id}
              href={task.href}
              className="flex items-center gap-4 p-4 rounded-xl transition-all hover:shadow-md"
              style={{
                background: '#fff',
                border: '1px solid var(--border)',
                opacity: isDone ? 0.6 : 1,
              }}
            >
              <div
                className="flex items-center justify-center rounded-lg shrink-0"
                style={{
                  width: 40,
                  height: 40,
                  background: isDone ? '#f0f0f0' : 'var(--forest-50)',
                  color: isDone ? 'var(--muted)' : 'var(--forest-600)',
                }}
              >
                {isDone ? (
                  <CheckCircle2 style={{ width: 20, height: 20, color: 'var(--forest-500)' }} />
                ) : (
                  <Icon style={{ width: 18, height: 18 }} />
                )}
              </div>
              <div className="flex-1 min-w-0">
                <p
                  className="text-sm font-medium truncate"
                  style={{
                    color: 'var(--ink)',
                    textDecoration: isDone ? 'line-through' : 'none',
                  }}
                >
                  {task.title}
                </p>
                <p className="text-xs mt-0.5 truncate" style={{ color: 'var(--muted)' }}>
                  {task.subtitle}
                </p>
              </div>
              <span
                className="shrink-0 px-2.5 py-1 rounded-full text-xs font-medium mono"
                style={{ background: badge.bg, color: badge.color }}
              >
                {task.timeUntil}
              </span>
            </Link>
          );
        })}
      </div>

      {/* Quick actions */}
      <div className="grid grid-cols-2 gap-3">
        <Link
          href="/plan"
          className="flex items-center gap-3 p-4 rounded-xl"
          style={{
            background: 'var(--forest-50)',
            border: '1px solid var(--forest-200)',
          }}
        >
          <Zap style={{ width: 16, height: 16, color: 'var(--forest-600)' }} />
          <span className="text-sm font-medium" style={{ color: 'var(--forest-700)' }}>
            16-week plan
          </span>
        </Link>
        <Link
          href="/office-hours"
          className="flex items-center gap-3 p-4 rounded-xl"
          style={{
            background: '#fff',
            border: '1px solid var(--border)',
          }}
        >
          <Clock style={{ width: 16, height: 16, color: 'var(--muted)' }} />
          <span className="text-sm font-medium" style={{ color: 'var(--ink)' }}>
            Office hours
          </span>
        </Link>
      </div>
    </div>
  );
}

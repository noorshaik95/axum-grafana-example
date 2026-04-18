'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { CheckCircle2, Circle, Clock } from 'lucide-react';

interface Week {
  num: number;
  label: string;
  status: 'done' | 'current' | 'upcoming';
  theme: string;
  deadlines: { title: string; course: string; date: string }[];
}

const WEEKS: Week[] = [
  { num: 1, label: 'W1', status: 'done', theme: 'Orientation & Setup', deadlines: [] },
  {
    num: 2,
    label: 'W2',
    status: 'done',
    theme: 'Foundations of CS',
    deadlines: [{ title: 'Quiz 1', course: 'CS 101', date: 'Mon' }],
  },
  {
    num: 3,
    label: 'W3',
    status: 'done',
    theme: 'Algorithms I',
    deadlines: [{ title: 'PS1', course: 'CS 101', date: 'Fri' }],
  },
  { num: 4, label: 'W4', status: 'done', theme: 'Data Structures I', deadlines: [] },
  {
    num: 5,
    label: 'W5',
    status: 'done',
    theme: 'Recursion',
    deadlines: [{ title: 'PS2', course: 'CS 101', date: 'Wed' }],
  },
  { num: 6, label: 'W6', status: 'done', theme: 'Sorting Algorithms', deadlines: [] },
  {
    num: 7,
    label: 'W7',
    status: 'done',
    theme: 'Midterm Review',
    deadlines: [{ title: 'Midterm', course: 'CS 101', date: 'Thu' }],
  },
  {
    num: 8,
    label: 'W8',
    status: 'current',
    theme: 'Trees & Graphs',
    deadlines: [
      { title: 'PS4 — Recursion', course: 'CS 101', date: 'Fri' },
      { title: 'Lab 3', course: 'CS 201', date: 'Mon' },
    ],
  },
  {
    num: 9,
    label: 'W9',
    status: 'upcoming',
    theme: 'Hash Tables',
    deadlines: [{ title: 'PS5', course: 'CS 101', date: 'Fri' }],
  },
  { num: 10, label: 'W10', status: 'upcoming', theme: 'Dynamic Programming I', deadlines: [] },
  {
    num: 11,
    label: 'W11',
    status: 'upcoming',
    theme: 'Dynamic Programming II',
    deadlines: [{ title: 'PS6', course: 'CS 101', date: 'Wed' }],
  },
  { num: 12, label: 'W12', status: 'upcoming', theme: 'Graph Algorithms', deadlines: [] },
  {
    num: 13,
    label: 'W13',
    status: 'upcoming',
    theme: 'Advanced Topics',
    deadlines: [{ title: 'Final Project', course: 'CS 201', date: 'Mon' }],
  },
  { num: 14, label: 'W14', status: 'upcoming', theme: 'Review & Prep', deadlines: [] },
  {
    num: 15,
    label: 'W15',
    status: 'upcoming',
    theme: 'Final Exams',
    deadlines: [{ title: 'Final Exam', course: 'CS 101', date: 'Thu' }],
  },
  { num: 16, label: 'W16', status: 'upcoming', theme: 'Wrap-up', deadlines: [] },
];

function usePlan() {
  return useQuery({
    queryKey: ['plan-mock'],
    queryFn: async () => WEEKS,
  });
}

export default function PlanPage() {
  const { data: weeks = WEEKS } = usePlan();
  const [selectedWeek, setSelectedWeek] = useState<number>(8);

  const currentWeek = weeks.find((w) => w.num === selectedWeek);

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="serif text-2xl sm:text-3xl" style={{ color: 'var(--ink)' }}>
          Your 16-week plan
        </h1>
        <p className="text-sm mt-1" style={{ color: 'var(--muted)' }}>
          Stay on track through the semester
        </p>
      </div>

      {/* Week spine */}
      <div
        className="rounded-xl p-4 overflow-x-auto"
        style={{ background: '#fff', border: '1px solid var(--border)' }}
      >
        <div className="flex gap-2 min-w-max">
          {weeks.map((w) => {
            const isSelected = w.num === selectedWeek;
            return (
              <button
                key={w.num}
                onClick={() => setSelectedWeek(w.num)}
                className="flex flex-col items-center gap-1.5 px-2 py-1.5 rounded-lg transition-all"
                style={{
                  background: isSelected
                    ? w.status === 'current'
                      ? 'var(--amber)'
                      : 'var(--forest-700)'
                    : 'transparent',
                  minWidth: 36,
                }}
              >
                <span
                  style={{
                    fontSize: 10,
                    fontWeight: isSelected ? 700 : 400,
                    color: isSelected
                      ? w.status === 'current'
                        ? 'var(--ink)'
                        : '#fff'
                      : 'var(--muted)',
                  }}
                >
                  {w.label}
                </span>
                <div
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: '50%',
                    background:
                      w.status === 'done'
                        ? 'var(--forest-500)'
                        : w.status === 'current'
                          ? 'var(--amber)'
                          : isSelected
                            ? '#fff'
                            : 'var(--border)',
                  }}
                />
              </button>
            );
          })}
        </div>
      </div>

      {/* Week detail cards */}
      {[selectedWeek - 1, selectedWeek, selectedWeek + 1]
        .filter((n) => n >= 1 && n <= 16)
        .map((num) => {
          const w = weeks.find((wk) => wk.num === num)!;
          const isCurrent = w.status === 'current';
          const isDone = w.status === 'done';
          return (
            <div
              key={w.num}
              className="rounded-xl overflow-hidden transition-all"
              style={{
                background: isCurrent
                  ? 'linear-gradient(135deg, var(--forest-700) 0%, var(--forest-900) 100%)'
                  : '#fff',
                border: `1px solid ${isCurrent ? 'var(--forest-700)' : 'var(--border)'}`,
                opacity: isDone ? 0.75 : 1,
              }}
            >
              <div className="flex items-center justify-between px-5 py-4">
                <div className="flex items-center gap-3">
                  {isDone ? (
                    <CheckCircle2
                      style={{ width: 18, height: 18, color: 'var(--forest-500)', flexShrink: 0 }}
                    />
                  ) : isCurrent ? (
                    <div
                      style={{
                        width: 18,
                        height: 18,
                        borderRadius: '50%',
                        background: 'var(--amber)',
                        flexShrink: 0,
                      }}
                    />
                  ) : (
                    <Circle
                      style={{ width: 18, height: 18, color: 'var(--muted)', flexShrink: 0 }}
                    />
                  )}
                  <div>
                    <span
                      className="mono text-xs font-semibold"
                      style={{ color: isCurrent ? 'var(--amber)' : 'var(--muted)' }}
                    >
                      Week {w.num}
                    </span>
                    <p
                      className="text-sm font-semibold"
                      style={{ color: isCurrent ? '#f2f7f3' : 'var(--ink)' }}
                    >
                      {w.theme}
                    </p>
                  </div>
                </div>
                {isCurrent && (
                  <span
                    className="text-xs font-bold px-2.5 py-1 rounded-full"
                    style={{ background: 'var(--amber)', color: 'var(--ink)' }}
                  >
                    Current
                  </span>
                )}
              </div>
              {w.deadlines.length > 0 && (
                <div
                  className="px-5 pb-4 space-y-2 border-t"
                  style={{ borderColor: isCurrent ? 'rgba(255,255,255,0.1)' : 'var(--border)' }}
                >
                  <p
                    className="text-xs font-semibold mt-3 uppercase tracking-wide"
                    style={{ color: isCurrent ? 'var(--forest-300)' : 'var(--muted)' }}
                  >
                    Deadlines
                  </p>
                  {w.deadlines.map((d, i) => (
                    <div key={i} className="flex items-center gap-2">
                      <Clock
                        style={{
                          width: 12,
                          height: 12,
                          color: isCurrent ? 'var(--amber)' : 'var(--muted)',
                          flexShrink: 0,
                        }}
                      />
                      <span
                        className="text-xs font-medium"
                        style={{ color: isCurrent ? '#e9efe9' : 'var(--ink)' }}
                      >
                        {d.title}
                      </span>
                      <span
                        className="text-xs"
                        style={{ color: isCurrent ? 'var(--forest-300)' : 'var(--muted)' }}
                      >
                        {d.course} · {d.date}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          );
        })}
    </div>
  );
}

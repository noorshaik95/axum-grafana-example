'use client';

import { useParams } from 'next/navigation';
import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, MessageSquare, Users } from 'lucide-react';

interface Person {
  id: string;
  name: string;
  initials: string;
  role: 'student' | 'professor';
  bio: string;
  sharedCourses: { id: string; code: string; title: string }[];
  recentThreads: { id: string; snippet: string; timeAgo: string }[];
}

const MOCK_PROFILES: Record<string, Person> = {
  '1': {
    id: '1',
    name: 'Maya Chen',
    initials: 'MC',
    role: 'student',
    bio: 'Junior · CS major · GPA 3.8',
    sharedCourses: [
      { id: 'cs101', code: 'CS 101', title: 'Intro to Computer Science' },
      { id: 'cs201', code: 'CS 201', title: 'Data Structures' },
    ],
    recentThreads: [
      { id: 't1', snippet: 'Study group for final exam?', timeAgo: '2d ago' },
      { id: 't2', snippet: 'Did you get the Lab 3 notes?', timeAgo: '4h ago' },
    ],
  },
  '5': {
    id: '5',
    name: 'Prof. Martinez',
    initials: 'PM',
    role: 'professor',
    bio: 'Associate Professor · CS dept · 12 years teaching',
    sharedCourses: [{ id: 'cs101', code: 'CS 101', title: 'Intro to Computer Science' }],
    recentThreads: [
      { id: 't3', snippet: 'Great work on PS3! A few notes...', timeAgo: '2h ago' },
      { id: 't4', snippet: 'Deadline extended by 24 hours.', timeAgo: '3d ago' },
    ],
  },
};

const DEFAULT_PERSON: Person = {
  id: '0',
  name: 'Classmate',
  initials: 'CL',
  role: 'student',
  bio: 'No bio available',
  sharedCourses: [{ id: 'cs101', code: 'CS 101', title: 'Intro to Computer Science' }],
  recentThreads: [],
};

function usePersonProfile(id: string) {
  return useQuery({
    queryKey: ['person-profile', id],
    queryFn: async () => MOCK_PROFILES[id] ?? { ...DEFAULT_PERSON, id },
  });
}

export default function PersonProfilePage() {
  const params = useParams();
  const id = params.id as string;
  const { data: person } = usePersonProfile(id);

  if (!person) return null;

  return (
    <div className="max-w-xl mx-auto space-y-5">
      <Link
        href="/people"
        className="inline-flex items-center gap-1.5 text-sm"
        style={{ color: 'var(--muted)' }}
      >
        <ArrowLeft style={{ width: 14, height: 14 }} />
        People
      </Link>

      {/* Profile header */}
      <div
        className="rounded-xl p-6 flex flex-col items-center text-center gap-4"
        style={{ background: '#fff', border: '1px solid var(--border)' }}
      >
        <div
          className="flex items-center justify-center rounded-full text-2xl font-bold"
          style={{
            width: 72,
            height: 72,
            background: person.role === 'professor' ? 'var(--forest-100)' : 'var(--paper)',
            color: person.role === 'professor' ? 'var(--forest-700)' : 'var(--muted)',
            border: '3px solid var(--border)',
          }}
        >
          {person.initials}
        </div>
        <div>
          <h1 className="serif text-2xl" style={{ color: 'var(--ink)' }}>
            {person.name}
          </h1>
          <p className="text-sm mt-1" style={{ color: 'var(--muted)' }}>
            {person.bio}
          </p>
        </div>
        <div className="flex gap-3">
          <Link
            href="/inbox"
            className="btn-primary flex items-center gap-2"
            style={{ fontSize: 13 }}
          >
            <MessageSquare style={{ width: 13, height: 13 }} />
            Message
          </Link>
          <button className="btn-secondary flex items-center gap-2" style={{ fontSize: 13 }}>
            <Users style={{ width: 13, height: 13 }} />
            Study group
          </button>
        </div>
      </div>

      {/* Shared courses */}
      <div
        className="rounded-xl overflow-hidden"
        style={{ background: '#fff', border: '1px solid var(--border)' }}
      >
        <div className="px-5 py-3 border-b" style={{ borderColor: 'var(--border)' }}>
          <h2 className="text-sm font-semibold" style={{ color: 'var(--ink)' }}>
            Shared courses
          </h2>
        </div>
        <div className="divide-y" style={{ borderColor: 'var(--border)' }}>
          {person.sharedCourses.map((c) => (
            <Link
              key={c.id}
              href={`/courses/${c.id}`}
              className="flex items-center gap-3 px-5 py-3 hover:bg-[var(--forest-50)]"
            >
              <span
                className="text-xs font-semibold px-2 py-0.5 rounded"
                style={{ background: 'var(--forest-100)', color: 'var(--forest-700)' }}
              >
                {c.code}
              </span>
              <span className="text-sm" style={{ color: 'var(--ink)' }}>
                {c.title}
              </span>
            </Link>
          ))}
        </div>
      </div>

      {/* Recent threads */}
      {person.recentThreads.length > 0 && (
        <div
          className="rounded-xl overflow-hidden"
          style={{ background: '#fff', border: '1px solid var(--border)' }}
        >
          <div className="px-5 py-3 border-b" style={{ borderColor: 'var(--border)' }}>
            <h2 className="text-sm font-semibold" style={{ color: 'var(--ink)' }}>
              Recent threads
            </h2>
          </div>
          <div className="divide-y" style={{ borderColor: 'var(--border)' }}>
            {person.recentThreads.map((t) => (
              <div key={t.id} className="flex items-center justify-between px-5 py-3">
                <p className="text-sm" style={{ color: 'var(--ink)' }}>
                  {t.snippet}
                </p>
                <span className="text-xs mono shrink-0 ml-3" style={{ color: 'var(--muted)' }}>
                  {t.timeAgo}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

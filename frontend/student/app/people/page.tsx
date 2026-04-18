'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { Users } from 'lucide-react';

type Segment = 'classmates' | 'professors' | 'groups';

interface Person {
  id: string;
  name: string;
  initials: string;
  role: 'student' | 'professor';
  sharedClass: string;
  bio?: string;
}

const MOCK_PEOPLE: Person[] = [
  {
    id: '1',
    name: 'Maya Chen',
    initials: 'MC',
    role: 'student',
    sharedClass: 'CS 101',
    bio: 'Junior · CS major',
  },
  {
    id: '2',
    name: 'Jordan Kim',
    initials: 'JK',
    role: 'student',
    sharedClass: 'CS 101',
    bio: 'Senior · CS major',
  },
  {
    id: '3',
    name: 'Priya Patel',
    initials: 'PP',
    role: 'student',
    sharedClass: 'MATH 202',
    bio: 'Sophomore · Math major',
  },
  {
    id: '4',
    name: 'Alex Torres',
    initials: 'AT',
    role: 'student',
    sharedClass: 'CS 201',
    bio: 'Junior · CS major',
  },
  {
    id: '5',
    name: 'Prof. Martinez',
    initials: 'PM',
    role: 'professor',
    sharedClass: 'CS 101',
    bio: 'Associate Professor · CS dept',
  },
  {
    id: '6',
    name: 'Prof. Williams',
    initials: 'PW',
    role: 'professor',
    sharedClass: 'MATH 202',
    bio: 'Professor · Mathematics dept',
  },
];

function usePeople() {
  return useQuery({
    queryKey: ['people-mock'],
    queryFn: async () => MOCK_PEOPLE,
  });
}

const SEGMENTS: { key: Segment; label: string }[] = [
  { key: 'classmates', label: 'Classmates' },
  { key: 'professors', label: 'Professors' },
  { key: 'groups', label: 'Groups' },
];

export default function PeoplePage() {
  const [segment, setSegment] = useState<Segment>('classmates');
  const { data: people = [] } = usePeople();

  const filtered = people.filter((p) => {
    if (segment === 'classmates') return p.role === 'student';
    if (segment === 'professors') return p.role === 'professor';
    return false; // groups - placeholder
  });

  return (
    <div className="max-w-2xl mx-auto space-y-4">
      <div>
        <h1 className="serif text-2xl" style={{ color: 'var(--ink)' }}>
          People
        </h1>
        <p className="text-sm mt-1" style={{ color: 'var(--muted)' }}>
          Your classmates and instructors
        </p>
      </div>

      {/* Segment tabs */}
      <div
        className="flex gap-1 p-1 rounded-xl"
        style={{ background: 'var(--paper)', border: '1px solid var(--border)' }}
      >
        {SEGMENTS.map((s) => {
          const isActive = segment === s.key;
          return (
            <button
              key={s.key}
              onClick={() => setSegment(s.key)}
              className="flex-1 py-1.5 rounded-lg text-sm font-medium transition-all"
              style={{
                background: isActive ? '#fff' : 'transparent',
                color: isActive ? 'var(--forest-700)' : 'var(--muted)',
                boxShadow: isActive ? 'var(--shadow-sm)' : 'none',
                border: isActive ? '1px solid var(--border)' : '1px solid transparent',
              }}
            >
              {s.label}
            </button>
          );
        })}
      </div>

      {segment === 'groups' ? (
        <div
          className="py-16 text-center rounded-xl"
          style={{ background: '#fff', border: '1px solid var(--border)' }}
        >
          <Users style={{ width: 32, height: 32, color: 'var(--muted)', margin: '0 auto 8px' }} />
          <p className="text-sm" style={{ color: 'var(--muted)' }}>
            Study groups coming soon.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-3">
          {filtered.map((person) => (
            <Link
              key={person.id}
              href={`/people/${person.id}`}
              className="flex flex-col items-center gap-3 p-5 rounded-xl transition-all hover:shadow-md"
              style={{ background: '#fff', border: '1px solid var(--border)' }}
            >
              <div
                className="flex items-center justify-center rounded-full text-base font-semibold"
                style={{
                  width: 48,
                  height: 48,
                  background: person.role === 'professor' ? 'var(--forest-100)' : 'var(--paper)',
                  color: person.role === 'professor' ? 'var(--forest-700)' : 'var(--muted)',
                  border: '2px solid var(--border)',
                }}
              >
                {person.initials}
              </div>
              <div className="text-center min-w-0 w-full">
                <p className="text-sm font-semibold truncate" style={{ color: 'var(--ink)' }}>
                  {person.name}
                </p>
                <p className="text-xs mt-0.5 truncate" style={{ color: 'var(--muted)' }}>
                  {person.sharedClass}
                </p>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

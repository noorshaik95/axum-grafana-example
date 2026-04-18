'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Inbox, AtSign, GraduationCap, Users } from 'lucide-react';

type Segment = 'you' | 'all' | 'profs' | 'peers';

interface Thread {
  id: string;
  sender: string;
  senderRole: 'prof' | 'peer' | 'system';
  initials: string;
  snippet: string;
  timeAgo: string;
  unread: boolean;
  courseTag?: string;
}

const MOCK_THREADS: Thread[] = [
  {
    id: '1',
    sender: 'Prof. Martinez',
    senderRole: 'prof',
    initials: 'PM',
    snippet: 'Great work on PS3! A few notes on your recursion approach...',
    timeAgo: '2h ago',
    unread: true,
    courseTag: 'CS 101',
  },
  {
    id: '2',
    sender: 'Maya Chen',
    senderRole: 'peer',
    initials: 'MC',
    snippet: 'Hey, did you get the study group notes from Tuesday?',
    timeAgo: '4h ago',
    unread: true,
  },
  {
    id: '3',
    sender: 'Prof. Williams',
    senderRole: 'prof',
    initials: 'PW',
    snippet: 'Office hours moved to Friday 2–4pm this week.',
    timeAgo: '1d ago',
    unread: false,
    courseTag: 'MATH 202',
  },
  {
    id: '4',
    sender: 'Jordan Kim',
    senderRole: 'peer',
    initials: 'JK',
    snippet: "Can you share your approach to Lab 3? I'm stuck on the tree traversal",
    timeAgo: '2d ago',
    unread: false,
  },
  {
    id: '5',
    sender: 'Prof. Martinez',
    senderRole: 'prof',
    initials: 'PM',
    snippet: 'The deadline for PS4 has been extended by 24 hours.',
    timeAgo: '3d ago',
    unread: false,
    courseTag: 'CS 101',
  },
];

function useInboxData() {
  return useQuery({
    queryKey: ['inbox-mock'],
    queryFn: async () => MOCK_THREADS,
  });
}

const SEGMENTS: { key: Segment; label: string; icon: React.ElementType }[] = [
  { key: 'you', label: '@you', icon: AtSign },
  { key: 'all', label: 'All', icon: Inbox },
  { key: 'profs', label: 'Profs', icon: GraduationCap },
  { key: 'peers', label: 'Peers', icon: Users },
];

export default function InboxPage() {
  const [segment, setSegment] = useState<Segment>('all');
  const [readIds, setReadIds] = useState<Set<string>>(new Set());
  const { data: threads = [] } = useInboxData();

  const filtered = threads.filter((t) => {
    if (segment === 'you') return t.unread && !readIds.has(t.id);
    if (segment === 'profs') return t.senderRole === 'prof';
    if (segment === 'peers') return t.senderRole === 'peer';
    return true;
  });

  const handleClick = (id: string) => {
    setReadIds((prev) => new Set([...prev, id]));
  };

  return (
    <div className="max-w-2xl mx-auto space-y-4">
      <div>
        <h1 className="serif text-2xl" style={{ color: 'var(--ink)' }}>
          Inbox
        </h1>
        <p className="text-sm mt-1" style={{ color: 'var(--muted)' }}>
          Messages from instructors and classmates
        </p>
      </div>

      {/* Segment tabs */}
      <div
        className="flex gap-1 p-1 rounded-xl"
        style={{ background: 'var(--paper)', border: '1px solid var(--border)' }}
      >
        {SEGMENTS.map((s) => {
          const Icon = s.icon;
          const isActive = segment === s.key;
          return (
            <button
              key={s.key}
              onClick={() => setSegment(s.key)}
              className="flex-1 flex items-center justify-center gap-1.5 py-1.5 rounded-lg text-sm font-medium transition-all"
              style={{
                background: isActive ? '#fff' : 'transparent',
                color: isActive ? 'var(--forest-700)' : 'var(--muted)',
                boxShadow: isActive ? 'var(--shadow-sm)' : 'none',
                border: isActive ? '1px solid var(--border)' : '1px solid transparent',
              }}
            >
              <Icon style={{ width: 13, height: 13 }} />
              {s.label}
            </button>
          );
        })}
      </div>

      {/* Thread list */}
      <div
        className="rounded-xl overflow-hidden"
        style={{ border: '1px solid var(--border)', background: '#fff' }}
      >
        {filtered.length === 0 ? (
          <div className="py-16 text-center">
            <Inbox style={{ width: 32, height: 32, color: 'var(--muted)', margin: '0 auto 8px' }} />
            <p className="text-sm" style={{ color: 'var(--muted)' }}>
              No messages here.
            </p>
          </div>
        ) : (
          <div className="divide-y" style={{ borderColor: 'var(--border)' }}>
            {filtered.map((thread) => {
              const isUnread = thread.unread && !readIds.has(thread.id);
              return (
                <button
                  key={thread.id}
                  onClick={() => handleClick(thread.id)}
                  className="w-full flex items-start gap-4 p-4 text-left transition-colors hover:bg-[var(--forest-50)]"
                >
                  <div
                    className="flex items-center justify-center rounded-full shrink-0 text-sm font-semibold"
                    style={{
                      width: 40,
                      height: 40,
                      background:
                        thread.senderRole === 'prof' ? 'var(--forest-100)' : 'var(--paper)',
                      color: thread.senderRole === 'prof' ? 'var(--forest-700)' : 'var(--muted)',
                      border: '1px solid var(--border)',
                    }}
                  >
                    {thread.initials}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-2 min-w-0">
                        <span
                          className="text-sm font-semibold truncate"
                          style={{ color: 'var(--ink)' }}
                        >
                          {thread.sender}
                        </span>
                        {thread.senderRole === 'prof' && (
                          <span
                            className="shrink-0 text-xs px-1.5 py-0.5 rounded"
                            style={{
                              background: 'var(--forest-100)',
                              color: 'var(--forest-700)',
                              fontWeight: 600,
                            }}
                          >
                            Prof
                          </span>
                        )}
                        {thread.courseTag && (
                          <span
                            className="shrink-0 text-xs px-1.5 py-0.5 rounded"
                            style={{ background: 'var(--paper)', color: 'var(--muted)' }}
                          >
                            {thread.courseTag}
                          </span>
                        )}
                      </div>
                      <span className="shrink-0 text-xs mono" style={{ color: 'var(--muted)' }}>
                        {thread.timeAgo}
                      </span>
                    </div>
                    <p
                      className="text-sm mt-0.5 truncate"
                      style={{
                        color: 'var(--muted)',
                        fontWeight: isUnread ? 500 : 400,
                      }}
                    >
                      {thread.snippet}
                    </p>
                  </div>
                  {isUnread && (
                    <div
                      className="shrink-0 mt-1.5 rounded-full"
                      style={{ width: 7, height: 7, background: 'var(--forest-500)' }}
                    />
                  )}
                </button>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

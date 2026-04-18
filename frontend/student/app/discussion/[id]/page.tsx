'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, Send } from 'lucide-react';

interface ThreadMessage {
  id: string;
  sender: string;
  initials: string;
  role: 'student' | 'prof';
  content: string;
  timeAgo: string;
}

interface DiscussionThread {
  id: string;
  title: string;
  course: string;
  startedBy: string;
  replyCount: number;
  messages: ThreadMessage[];
}

const MOCK_THREADS: Record<string, DiscussionThread> = {
  '1': {
    id: '1',
    title: 'Understanding base cases in recursion',
    course: 'CS 101',
    startedBy: 'Maya Chen',
    replyCount: 14,
    messages: [
      {
        id: 'm1',
        sender: 'Maya Chen',
        initials: 'MC',
        role: 'student',
        content:
          "I'm confused about when to stop recursion. The examples in the book always seem obvious, but when I'm solving PS4 problems I can't figure out the base case. Any tips?",
        timeAgo: '3h ago',
      },
      {
        id: 'm2',
        sender: 'Prof. Martinez',
        initials: 'PM',
        role: 'prof',
        content:
          'Great question, Maya! The key insight is: "what is the smallest version of this problem I can solve trivially?" For factorial, that\'s n=0 (answer is 1). For tree traversal, it\'s an empty node. Ask yourself: what input makes the problem immediately solvable without recursing?',
        timeAgo: '2h ago',
      },
      {
        id: 'm3',
        sender: 'Jordan Kim',
        initials: 'JK',
        role: 'student',
        content:
          'This clicked for me when I thought about it as: if I keep making the problem smaller, what\'s the "floor" I can\'t go below? That floor is your base case.',
        timeAgo: '1h ago',
      },
      {
        id: 'm4',
        sender: 'Maya Chen',
        initials: 'MC',
        role: 'student',
        content:
          "Oh that makes sense! So for PS4 problem 3 where we're reversing a string, the base case would be an empty string or single character?",
        timeAgo: '45m ago',
      },
      {
        id: 'm5',
        sender: 'Prof. Martinez',
        initials: 'PM',
        role: 'prof',
        content:
          'Exactly right! Either empty string OR single character works — both are trivially "reversed" already. You can pick either one. Single character is slightly more elegant because it makes the recursive case cleaner.',
        timeAgo: '30m ago',
      },
    ],
  },
};

const DEFAULT_THREAD: DiscussionThread = {
  id: '0',
  title: 'Discussion Thread',
  course: 'CS 101',
  startedBy: 'Student',
  replyCount: 0,
  messages: [],
};

function useDiscussionThread(id: string) {
  return useQuery({
    queryKey: ['discussion-thread', id],
    queryFn: async () => MOCK_THREADS[id] ?? { ...DEFAULT_THREAD, id },
  });
}

export default function DiscussionPage() {
  const params = useParams();
  const id = params.id as string;
  const { data: thread } = useDiscussionThread(id);
  const [reply, setReply] = useState('');
  const [localMessages, setLocalMessages] = useState<ThreadMessage[]>([]);

  const allMessages = [...(thread?.messages ?? []), ...localMessages];

  const handleSend = () => {
    if (!reply.trim()) return;
    setLocalMessages((prev) => [
      ...prev,
      {
        id: `local-${Date.now()}`,
        sender: 'You',
        initials: 'YO',
        role: 'student' as const,
        content: reply.trim(),
        timeAgo: 'just now',
      },
    ]);
    setReply('');
  };

  if (!thread) return null;

  return (
    <div className="max-w-2xl mx-auto space-y-5">
      <Link
        href="/courses"
        className="inline-flex items-center gap-1.5 text-sm"
        style={{ color: 'var(--muted)' }}
      >
        <ArrowLeft style={{ width: 14, height: 14 }} />
        Back
      </Link>

      {/* Thread header */}
      <div
        className="rounded-xl p-5"
        style={{ background: '#fff', border: '1px solid var(--border)' }}
      >
        <span
          className="text-xs font-semibold px-2 py-0.5 rounded"
          style={{ background: 'var(--forest-100)', color: 'var(--forest-700)' }}
        >
          {thread.course}
        </span>
        <h1 className="serif text-xl sm:text-2xl mt-3 leading-snug" style={{ color: 'var(--ink)' }}>
          {thread.title}
        </h1>
        <p className="text-xs mt-2" style={{ color: 'var(--muted)' }}>
          Started by {thread.startedBy} · {thread.replyCount + localMessages.length} replies
        </p>
      </div>

      {/* Messages */}
      <div className="space-y-3">
        {allMessages.map((msg) => {
          const isProf = msg.role === 'prof';
          return (
            <div
              key={msg.id}
              className="rounded-xl p-4"
              style={{
                background: isProf ? 'var(--forest-50)' : '#fff',
                border: `1px solid ${isProf ? 'var(--forest-200)' : 'var(--border)'}`,
              }}
            >
              <div className="flex items-start gap-3">
                <div
                  className="flex items-center justify-center rounded-full text-xs font-semibold shrink-0"
                  style={{
                    width: 36,
                    height: 36,
                    background: isProf ? 'var(--forest-200)' : 'var(--paper)',
                    color: isProf ? 'var(--forest-800)' : 'var(--muted)',
                    border: '1px solid var(--border)',
                  }}
                >
                  {msg.initials}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-sm font-semibold" style={{ color: 'var(--ink)' }}>
                      {msg.sender}
                    </span>
                    {isProf && (
                      <span
                        className="text-xs px-1.5 py-0.5 rounded font-semibold"
                        style={{ background: 'var(--forest-700)', color: '#fff' }}
                      >
                        prof
                      </span>
                    )}
                    <span className="text-xs mono ml-auto" style={{ color: 'var(--muted)' }}>
                      {msg.timeAgo}
                    </span>
                  </div>
                  <p className="text-sm leading-relaxed" style={{ color: 'var(--ink)' }}>
                    {msg.content}
                  </p>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Reply box */}
      <div
        className="rounded-xl p-4"
        style={{ background: '#fff', border: '1px solid var(--border)' }}
      >
        <p className="text-xs font-semibold mb-2" style={{ color: 'var(--muted)' }}>
          Reply to thread
        </p>
        <textarea
          value={reply}
          onChange={(e) => setReply(e.target.value)}
          placeholder="Share your thoughts or ask a question..."
          rows={3}
          className="w-full rounded-lg p-3 text-sm resize-none focus:outline-none"
          style={{
            border: '1px solid var(--border)',
            background: 'var(--paper)',
            color: 'var(--ink)',
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) handleSend();
          }}
        />
        <div className="flex items-center justify-between mt-3">
          <span className="text-xs" style={{ color: 'var(--muted)' }}>
            ⌘↵ to send
          </span>
          <button
            onClick={handleSend}
            disabled={!reply.trim()}
            className="btn-primary flex items-center gap-2 disabled:opacity-40"
            style={{ fontSize: 13 }}
          >
            <Send style={{ width: 13, height: 13 }} />
            Send
          </button>
        </div>
      </div>
    </div>
  );
}

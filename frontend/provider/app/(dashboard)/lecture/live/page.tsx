'use client'

import { useState } from 'react'
import { ThumbsUp, Send } from 'lucide-react'

const mockQuestions = [
  {
    id: 1,
    student: 'Alex M.',
    question: 'Why does tail recursion matter in Haskell specifically?',
    upvotes: 12,
    time: '2m ago',
    answered: false,
  },
  {
    id: 2,
    student: 'Jordan K.',
    question: 'Is the accumulator pattern always more efficient?',
    upvotes: 8,
    time: '5m ago',
    answered: false,
  },
  {
    id: 3,
    student: 'Sam T.',
    question: 'Can you show the difference in memory usage?',
    upvotes: 6,
    time: '7m ago',
    answered: false,
  },
  {
    id: 4,
    student: 'Riley P.',
    question: 'Does GHC automatically optimize non-tail recursive calls?',
    upvotes: 4,
    time: '11m ago',
    answered: true,
  },
]

export default function LectureLivePage() {
  const [questions, setQuestions] = useState(mockQuestions)
  const [newQuestion, setNewQuestion] = useState('')

  const handleUpvote = (id: number) => {
    setQuestions((prev) => prev.map((q) => (q.id === id ? { ...q, upvotes: q.upvotes + 1 } : q)))
  }

  const handleMarkAnswered = (id: number) => {
    setQuestions((prev) => prev.map((q) => (q.id === id ? { ...q, answered: !q.answered } : q)))
  }

  const unanswered = questions.filter((q) => !q.answered).sort((a, b) => b.upvotes - a.upvotes)
  const answered = questions.filter((q) => q.answered)

  return (
    <div
      className="grid grid-cols-1 gap-5 lg:grid-cols-3"
      style={{ minHeight: 'calc(100vh - 140px)' }}
    >
      {/* Left: Video area — 2/3 */}
      <div className="lg:col-span-2 flex flex-col gap-4">
        {/* Video container */}
        <div
          className="relative rounded-2xl overflow-hidden flex items-center justify-center"
          style={{ background: '#0f2617', aspectRatio: '16/9' }}
        >
          {/* Live indicator */}
          <div className="absolute top-4 left-4 flex items-center gap-2">
            <span
              className="flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-mono font-medium"
              style={{ background: 'rgba(217,119,87,0.9)', color: '#ffffff' }}
            >
              <span className="h-1.5 w-1.5 rounded-full bg-current animate-pulse" />
              LIVE
            </span>
          </div>

          {/* Student count */}
          <div className="absolute top-4 right-4">
            <span
              className="flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-mono"
              style={{ background: 'rgba(255,255,255,0.12)', color: 'rgba(255,255,255,0.85)' }}
            >
              34/47 here
            </span>
          </div>

          {/* Center placeholder */}
          <div className="text-center">
            <div className="text-6xl mb-3" style={{ opacity: 0.15 }}>
              🎥
            </div>
            <p className="font-serif text-xl" style={{ color: 'rgba(255,255,255,0.4)' }}>
              Camera feed
            </p>
          </div>

          {/* Controls */}
          <div className="absolute bottom-4 left-1/2 -translate-x-1/2 flex items-center gap-2">
            {['Mute', 'Video off', 'Share screen', 'End'].map((ctrl) => (
              <button
                key={ctrl}
                className="rounded-xl px-3 py-2 text-xs font-medium transition-all hover:opacity-80"
                style={
                  ctrl === 'End'
                    ? { background: '#d97757', color: '#ffffff' }
                    : { background: 'rgba(255,255,255,0.15)', color: 'rgba(255,255,255,0.85)' }
                }
              >
                {ctrl}
              </button>
            ))}
          </div>
        </div>

        {/* Current slide card */}
        <div
          className="rounded-xl border px-5 py-4"
          style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
        >
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-mono font-medium" style={{ color: '#6a6e62' }}>
              CURRENT SLIDE
            </span>
            <span className="text-xs font-mono" style={{ color: '#6a6e62' }}>
              Slide 14 / 32
            </span>
          </div>
          <h3 className="font-serif text-lg text-[#12170f]">
            Tail Recursion & Accumulator Pattern
          </h3>
          <p className="text-sm mt-1" style={{ color: '#6a6e62' }}>
            CS 3110 · Lecture 7 · Functional Programming
          </p>
          <div className="flex gap-2 mt-3">
            <button
              className="rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors hover:bg-[#f6f3ec]"
              style={{ borderColor: '#e4e0d4', color: '#12170f' }}
            >
              ← Prev
            </button>
            <button
              className="rounded-lg px-3 py-1.5 text-xs font-medium transition-colors hover:opacity-80"
              style={{ background: '#234e32', color: '#ffffff' }}
            >
              Next →
            </button>
          </div>
        </div>
      </div>

      {/* Right: Questions list — 1/3 */}
      <div
        className="flex flex-col rounded-xl border overflow-hidden"
        style={{ background: '#ffffff', borderColor: '#e4e0d4' }}
      >
        {/* Questions header */}
        <div
          className="flex items-center justify-between px-4 py-3 border-b"
          style={{ background: '#f6f3ec', borderColor: '#e4e0d4' }}
        >
          <span className="text-sm font-semibold text-[#12170f]">Questions</span>
          <div className="flex items-center gap-1.5">
            <span
              className="rounded-full px-2 py-0.5 text-xs font-mono"
              style={{ background: '#dde9df', color: '#234e32' }}
            >
              {unanswered.length} up
            </span>
          </div>
        </div>

        {/* Unanswered questions */}
        <div className="flex-1 overflow-y-auto divide-y" style={{ borderColor: '#e4e0d4' }}>
          {unanswered.map((q) => (
            <div key={q.id} className="px-4 py-3">
              <div className="flex items-start justify-between gap-2 mb-1.5">
                <div className="flex items-center gap-1.5">
                  <span className="text-xs font-medium text-[#12170f]">{q.student}</span>
                  <span className="text-xs" style={{ color: '#6a6e62' }}>
                    {q.time}
                  </span>
                </div>
                <button
                  onClick={() => handleMarkAnswered(q.id)}
                  className="text-xs px-2 py-0.5 rounded-full border transition-colors hover:bg-[#f2f7f3]"
                  style={{ borderColor: '#e4e0d4', color: '#6a6e62' }}
                >
                  Mark done
                </button>
              </div>
              <p className="text-sm text-[#12170f] leading-snug mb-2">{q.question}</p>
              <button
                onClick={() => handleUpvote(q.id)}
                className="flex items-center gap-1 text-xs transition-colors hover:text-[#234e32]"
                style={{ color: '#6a6e62' }}
              >
                <ThumbsUp className="h-3 w-3" />
                {q.upvotes}
              </button>
            </div>
          ))}

          {answered.length > 0 && (
            <>
              <div
                className="px-4 py-2 text-xs font-mono font-medium"
                style={{ background: '#f6f3ec', color: '#6a6e62' }}
              >
                ANSWERED
              </div>
              {answered.map((q) => (
                <div key={q.id} className="px-4 py-3 opacity-50">
                  <p className="text-sm text-[#12170f] line-through">{q.question}</p>
                </div>
              ))}
            </>
          )}
        </div>

        {/* Submit question (instructor use) */}
        <div className="border-t p-3" style={{ borderColor: '#e4e0d4' }}>
          <div
            className="flex items-center gap-2 rounded-lg border px-3 py-2"
            style={{ background: '#f6f3ec', borderColor: '#e4e0d4' }}
          >
            <input
              type="text"
              placeholder="Pin a question..."
              className="flex-1 text-sm bg-transparent outline-none"
              style={{ color: '#12170f' }}
              value={newQuestion}
              onChange={(e) => setNewQuestion(e.target.value)}
            />
            <button>
              <Send className="h-4 w-4" style={{ color: '#6a6e62' }} />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

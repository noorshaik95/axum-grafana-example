'use client'

import { useState } from 'react'
import { Calendar, Eye, Send } from 'lucide-react'

const AUDIENCE_OPTIONS = [
  { id: 'all', label: 'All schools' },
  { id: 'banner', label: 'In-app banner' },
  { id: 'email', label: 'Email admins' },
]

export default function BroadcastPage() {
  const [title, setTitle] = useState('')
  const [message, setMessage] = useState('')
  const [audience, setAudience] = useState<string[]>(['all', 'banner'])
  const [scheduled, setScheduled] = useState('')

  const toggleAudience = (id: string) => {
    setAudience((prev) => (prev.includes(id) ? prev.filter((a) => a !== id) : [...prev, id]))
  }

  return (
    <div className="max-w-xl space-y-6">
      <div>
        <h1
          className="text-2xl font-bold text-[#12170f]"
          style={{ fontFamily: 'Instrument Serif, Georgia, serif' }}
        >
          New broadcast
        </h1>
        <p className="text-sm text-[#6a6e62] mt-1">
          Send a platform-wide message to schools and users
        </p>
      </div>

      <div className="rounded-xl border border-[#e4e0d4] bg-white p-5 space-y-5">
        {/* Title */}
        <div>
          <label className="block text-xs font-medium text-[#6a6e62] uppercase tracking-wide mb-1.5">
            Title
          </label>
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="e.g. Scheduled maintenance — April 20"
            className="w-full px-3 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm text-[#12170f] placeholder-[#9a9e92] focus:outline-none focus:border-[#5d9a6c] transition-colors"
          />
        </div>

        {/* Message */}
        <div>
          <label className="block text-xs font-medium text-[#6a6e62] uppercase tracking-wide mb-1.5">
            Message
          </label>
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            rows={5}
            placeholder="Write your broadcast message here..."
            className="w-full px-3 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm text-[#12170f] placeholder-[#9a9e92] focus:outline-none focus:border-[#5d9a6c] transition-colors resize-none"
          />
          <p className="text-xs text-[#6a6e62] mt-1 text-right">{message.length} / 1,000</p>
        </div>

        {/* Audience chips */}
        <div>
          <label className="block text-xs font-medium text-[#6a6e62] uppercase tracking-wide mb-2">
            Audience
          </label>
          <div className="flex flex-wrap gap-2">
            {AUDIENCE_OPTIONS.map((opt) => (
              <button
                key={opt.id}
                onClick={() => toggleAudience(opt.id)}
                className={`px-3 py-1.5 rounded-full text-xs font-medium border transition-all ${
                  audience.includes(opt.id)
                    ? 'bg-[#234e32] text-white border-[#234e32]'
                    : 'bg-white text-[#6a6e62] border-[#e4e0d4] hover:border-[#8ab694]'
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>

        {/* Schedule */}
        <div>
          <label className="block text-xs font-medium text-[#6a6e62] uppercase tracking-wide mb-1.5">
            Schedule (optional)
          </label>
          <div className="relative">
            <Calendar className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[#6a6e62]" />
            <input
              type="datetime-local"
              value={scheduled}
              onChange={(e) => setScheduled(e.target.value)}
              className="w-full pl-9 pr-3 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm text-[#12170f] focus:outline-none focus:border-[#5d9a6c] transition-colors"
            />
          </div>
          <p className="text-xs text-[#6a6e62] mt-1">Leave blank to send immediately</p>
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-3">
        <button className="flex items-center gap-2 px-4 py-2 rounded-lg border border-[#e4e0d4] bg-white text-sm font-medium text-[#12170f] hover:bg-[#f6f3ec] transition-colors">
          <Eye className="h-4 w-4" />
          Preview
        </button>
        <button
          disabled={!title || !message || audience.length === 0}
          className="flex items-center gap-2 px-5 py-2 rounded-lg bg-[#234e32] text-white text-sm font-medium hover:bg-[#1a3a26] transition-colors disabled:opacity-40 disabled:cursor-not-allowed ml-auto"
        >
          <Send className="h-4 w-4" />
          {scheduled ? 'Schedule send' : 'Send now'}
        </button>
      </div>
    </div>
  )
}

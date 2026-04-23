'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Bell, Search, LogOut } from 'lucide-react'
import { useProfile } from '../../../shared/lib/api/hooks'

export function InstructorTopbar() {
  const router = useRouter()
  const { data: user } = useProfile()

  const initials = user
    ? `${user.firstName?.[0] ?? ''}${user.lastName?.[0] ?? ''}`.toUpperCase()
    : 'IN'

  const handleLogout = () => {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('slate_token')
      localStorage.removeItem('slate_refresh_token')
      document.cookie = 'slate_token=; path=/; max-age=0'
    }
    router.push('/login')
  }

  return (
    <header role="banner" className="sticky top-0 z-50 flex flex-col">
      {/* Main top bar */}
      <nav
        aria-label="Primary"
        className="flex h-14 items-center justify-between px-6 border-b"
        style={{
          background: 'rgba(251,250,245,0.85)',
          backdropFilter: 'blur(14px)',
          WebkitBackdropFilter: 'blur(14px)',
          borderBottomColor: '#e4e0d4',
        }}
      >
        {/* Left: Brand */}
        <Link href="/teach" className="flex items-center gap-2.5 shrink-0">
          <div
            className="flex h-7 w-7 items-center justify-center rounded-md text-white text-xs font-bold"
            style={{
              background: 'linear-gradient(135deg, #234e32 0%, #3e7d4f 100%)',
            }}
          >
            S
          </div>
          <span className="font-serif text-lg text-[#12170f] leading-none">Slate</span>
        </Link>

        {/* Center: ⌘K search */}
        <div className="flex-1 max-w-md mx-8">
          <div
            className="flex items-center gap-2 rounded-lg px-3 py-2 border text-sm text-[#6a6e62] cursor-pointer hover:border-[#8ab694] transition-colors"
            style={{ background: '#f6f3ec', borderColor: '#e4e0d4' }}
          >
            <Search className="h-4 w-4 shrink-0" />
            <span className="flex-1">Search courses, students...</span>
            <span
              className="font-mono text-xs px-1.5 py-0.5 rounded border"
              style={{ background: '#fbfaf5', borderColor: '#e4e0d4' }}
            >
              ⌘K
            </span>
          </div>
        </div>

        {/* Right: role pill + avatar + bell */}
        <div className="flex items-center gap-3 shrink-0">
          <span
            className="font-mono text-xs px-2.5 py-1 rounded-full font-medium"
            style={{ background: '#dde9df', color: '#234e32' }}
          >
            Instructor
          </span>
          <button className="relative p-2 rounded-lg hover:bg-[#f6f3ec] transition-colors">
            <Bell className="h-4 w-4 text-[#6a6e62]" />
            <span
              className="absolute top-1.5 right-1.5 h-2 w-2 rounded-full"
              style={{ background: '#ffb648' }}
            />
          </button>
          <div className="relative group">
            <button
              className="flex h-8 w-8 items-center justify-center rounded-full text-white text-xs font-semibold"
              style={{ background: 'linear-gradient(135deg, #234e32 0%, #3e7d4f 100%)' }}
            >
              {initials}
            </button>
            {/* Dropdown */}
            <div
              className="absolute right-0 top-10 w-48 rounded-xl border bg-white shadow-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all py-1"
              style={{ borderColor: '#e4e0d4' }}
            >
              <div className="px-3 py-2 border-b" style={{ borderColor: '#e4e0d4' }}>
                <p className="text-sm font-medium text-[#12170f]">
                  {user ? `${user.firstName} ${user.lastName}` : 'Instructor'}
                </p>
                <p className="text-xs text-[#6a6e62]">{user?.email ?? ''}</p>
              </div>
              <button
                onClick={handleLogout}
                className="flex w-full items-center gap-2 px-3 py-2 text-sm text-[#6a6e62] hover:bg-[#f6f3ec] transition-colors"
              >
                <LogOut className="h-4 w-4" />
                Sign out
              </button>
            </div>
          </div>
        </div>
      </nav>

      {/* Urgent items bar */}
      <div
        className="flex items-center gap-2 px-6 py-2 overflow-x-auto"
        style={{ background: '#234e32' }}
      >
        {/* Primary chip - amber */}
        <Link
          href="/grade/ps4"
          className="flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium font-mono whitespace-nowrap transition-opacity hover:opacity-80"
          style={{ background: '#ffb648', color: '#12170f' }}
        >
          <span className="font-bold">47 in queue · PS4</span>
          <span className="opacity-70">→ Grade now</span>
        </Link>

        <div className="w-px h-4 bg-white/20 shrink-0" />

        {/* Secondary chips */}
        <span
          className="flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-mono whitespace-nowrap"
          style={{ background: 'rgba(255,255,255,0.12)', color: 'rgba(255,255,255,0.85)' }}
        >
          Lecture 7 · 10:00
        </span>
        <Link
          href="/office-hours"
          className="flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-mono whitespace-nowrap transition-opacity hover:opacity-80"
          style={{ background: 'rgba(255,255,255,0.12)', color: 'rgba(255,255,255,0.85)' }}
        >
          OH · 4 booked · 2pm
        </Link>
      </div>
    </header>
  )
}

'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Bell, LogOut, Search } from 'lucide-react'
import { auth } from '../../../shared/lib/api'
import { useProfile } from '../../../shared/lib/api/hooks'

export function AdminTopbar() {
  const router = useRouter()
  const { data: profile } = useProfile()

  const initials = profile ? `${profile.firstName.charAt(0)}${profile.lastName.charAt(0)}` : 'AD'

  const handleLogout = async () => {
    await auth.logout()
    router.push('/login')
  }

  return (
    <div className="sticky top-0 z-50">
      {/* Main topbar */}
      <header className="bg-[rgba(251,250,245,0.85)] backdrop-blur-md border-b border-[#e4e0d4] flex h-14 items-center px-6 gap-4">
        {/* Left: Brand */}
        <Link href="/ops" className="flex items-center gap-2 shrink-0">
          <div className="h-7 w-7 rounded-md bg-gradient-to-br from-[#3e7d4f] to-[#0f2617] flex items-center justify-center text-white text-xs font-bold">
            S
          </div>
          <span
            className="text-[#12170f] text-base font-medium"
            style={{ fontFamily: 'Instrument Serif, Georgia, serif' }}
          >
            Slate Admin
          </span>
        </Link>

        {/* Center: Search */}
        <div className="flex-1 max-w-sm mx-auto">
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg border border-[#e4e0d4] bg-white/70 text-sm text-[#6a6e62] cursor-pointer hover:border-[#8ab694] transition-colors">
            <Search className="h-3.5 w-3.5 shrink-0" />
            <span className="flex-1 text-xs">Search&hellip;</span>
            <kbd className="text-[10px] font-mono bg-[#f6f3ec] px-1.5 py-0.5 rounded border border-[#e4e0d4]">
              ⌘K
            </kbd>
          </div>
        </div>

        {/* Right: Bell + Avatar + Logout */}
        <div className="flex items-center gap-2 ml-auto shrink-0">
          <button className="relative rounded-lg p-1.5 text-[#6a6e62] hover:bg-[#f6f3ec] transition-colors">
            <Bell className="h-4.5 w-4.5" />
            <span className="absolute right-1 top-1 h-2 w-2 rounded-full bg-[#d97757]" />
          </button>
          <div className="flex h-7 w-7 items-center justify-center rounded-full bg-[#dde9df] text-xs font-medium text-[#234e32]">
            {initials}
          </div>
          <button
            onClick={handleLogout}
            className="rounded-lg p-1.5 text-[#6a6e62] hover:bg-[#f6f3ec] transition-colors"
            title="Logout"
          >
            <LogOut className="h-4 w-4" />
          </button>
        </div>
      </header>

      {/* Now-bar */}
      <div className="bg-[#234e32] flex items-center gap-2 px-6 py-1.5 text-xs overflow-x-auto">
        <span className="shrink-0 flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-[#ffb648] text-[#12170f] font-medium">
          <span className="inline-block h-1.5 w-1.5 rounded-full bg-[#d97757] animate-pulse" />
          P1 · Stanford — grade sync backlog
        </span>
        <span className="shrink-0 flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-white/10 text-white/80">
          42 schools
        </span>
        <span className="shrink-0 flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-white/10 text-white/80">
          128k MAU
        </span>
        <span className="shrink-0 flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-white/10 text-white/80">
          99.97% uptime
        </span>
      </div>
    </div>
  )
}

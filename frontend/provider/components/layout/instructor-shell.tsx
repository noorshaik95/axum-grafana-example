'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import {
  LayoutDashboard,
  BookOpen,
  Users,
  ClipboardCheck,
  Video,
  Megaphone,
  Mail,
  Settings,
  LogOut,
} from 'lucide-react'
import { useProfile } from '../../../shared/lib/api/hooks'

const navItems = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Courses', href: '/courses', icon: BookOpen },
  { name: 'Students', href: '/students', icon: Users },
  { name: 'Grading', href: '/grading', icon: ClipboardCheck },
  { name: 'Video Sessions', href: '/video', icon: Video },
  { name: 'Announcements', href: '/announcements', icon: Megaphone },
  { name: 'Messages', href: '/messages', icon: Mail },
  { name: 'Settings', href: '/settings', icon: Settings },
]

export function InstructorShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
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
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <aside className="flex h-full w-64 flex-col bg-slate-900 text-white shrink-0">
        {/* Brand */}
        <div className="flex h-16 items-center gap-3 border-b border-white/10 px-5">
          <span className="text-lg font-bold tracking-tight">Slate</span>
          <span className="rounded-md bg-indigo-600 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider">
            Instructor
          </span>
        </div>

        {/* Nav */}
        <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-4">
          {navItems.map((item) => {
            const isActive = pathname === item.href || pathname.startsWith(`${item.href}/`)
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-indigo-600 text-white'
                    : 'text-slate-400 hover:bg-slate-800 hover:text-white'
                }`}
              >
                <item.icon className="h-5 w-5 shrink-0" />
                <span>{item.name}</span>
              </Link>
            )
          })}
        </nav>

        {/* User section */}
        <div className="border-t border-white/10 p-3">
          <div className="flex items-center gap-3 rounded-lg px-3 py-2">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-indigo-600 text-sm font-semibold">
              {initials}
            </div>
            <div className="flex-1 min-w-0">
              <p className="truncate text-sm font-medium text-white">
                {user ? `${user.firstName} ${user.lastName}` : 'Instructor'}
              </p>
              <p className="truncate text-xs text-slate-400">{user?.email ?? ''}</p>
            </div>
          </div>
          <button
            onClick={handleLogout}
            className="mt-1 flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-slate-400 hover:bg-slate-800 hover:text-white transition-colors"
          >
            <LogOut className="h-5 w-5 shrink-0" />
            <span>Logout</span>
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto bg-gray-50 px-8 py-6">{children}</main>
    </div>
  )
}

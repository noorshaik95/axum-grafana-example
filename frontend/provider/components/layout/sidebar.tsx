'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import { cn } from '@/lib/utils'
import {
  LayoutDashboard,
  BookOpen,
  Users,
  ClipboardCheck,
  BarChart2,
  Calendar,
  Settings,
  LogOut,
  GraduationCap,
} from 'lucide-react'
import { useProfile } from '../../../shared/lib/api/hooks'

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'My Courses', href: '/courses', icon: BookOpen },
  { name: 'Students', href: '/students', icon: Users },
  { name: 'Grading', href: '/grading', icon: ClipboardCheck },
  { name: 'Analytics', href: '/analytics', icon: BarChart2 },
  { name: 'Calendar', href: '/calendar', icon: Calendar },
  { name: 'Settings', href: '/settings', icon: Settings },
]

export function Sidebar() {
  const pathname = usePathname()
  const router = useRouter()
  const { data: profile } = useProfile()

  function handleLogout() {
    localStorage.removeItem('slate_token')
    localStorage.removeItem('slate_refresh_token')
    router.push('/login')
  }

  return (
    <aside className="flex flex-col w-64 min-h-screen bg-[#0f172a] text-white">
      <div className="flex h-16 items-center gap-3 px-6 border-b border-slate-700/50">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-600">
          <GraduationCap className="h-5 w-5" />
        </div>
        <div>
          <p className="text-sm font-semibold">Instructor Portal</p>
          <p className="text-xs text-slate-400">Slate LMS</p>
        </div>
      </div>

      <nav className="flex-1 py-4 px-3 space-y-1">
        {navigation.map((item) => {
          const isActive = pathname === item.href || pathname.startsWith(`${item.href}/`)
          return (
            <Link
              key={item.name}
              href={item.href}
              className={cn(
                'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-indigo-600/20 text-white'
                  : 'text-slate-400 hover:bg-slate-800 hover:text-white'
              )}
            >
              <item.icon className={cn('h-5 w-5 shrink-0', isActive ? 'text-indigo-400' : '')} />
              {item.name}
            </Link>
          )
        })}
      </nav>

      <div className="border-t border-slate-700/50 p-4">
        {profile && (
          <div className="flex items-center gap-3 mb-3">
            <div className="h-9 w-9 rounded-full bg-indigo-600/30 flex items-center justify-center text-sm font-medium text-indigo-300">
              {profile.firstName?.[0]}
              {profile.lastName?.[0]}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">
                {profile.firstName} {profile.lastName}
              </p>
              <p className="text-xs text-slate-400 truncate">{profile.email}</p>
            </div>
          </div>
        )}
        <button
          onClick={handleLogout}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm text-slate-400 hover:bg-slate-800 hover:text-white transition-colors"
        >
          <LogOut className="h-4 w-4" />
          Sign Out
        </button>
      </div>
    </aside>
  )
}

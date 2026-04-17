'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import {
  LayoutDashboard,
  Users,
  Building2,
  UserPlus,
  Shield,
  ScrollText,
  CreditCard,
  Activity,
  Radio,
  UserCog,
  Bell,
  LogOut,
  ChevronLeft,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useState } from 'react'
import { auth } from '../../../shared/lib/api'
import { useProfile } from '../../../shared/lib/api/hooks'
import { ImpersonationBanner } from '@/components/impersonation/ImpersonationBanner'

const navItems = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Onboarding', href: '/onboarding', icon: UserPlus },
  { name: 'Universities', href: '/universities', icon: Building2 },
  { name: 'Users', href: '/iam/users', icon: Users },
  { name: 'Roles', href: '/iam/roles', icon: Shield },
  { name: 'Audit Log', href: '/iam/audit', icon: ScrollText },
  { name: 'Billing', href: '/billing', icon: CreditCard },
  { name: 'System Health', href: '/system/health', icon: Activity },
  { name: 'Kafka', href: '/system/kafka', icon: Radio },
  { name: 'Impersonation', href: '/impersonation', icon: UserCog },
]

export function AdminShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const router = useRouter()
  const [collapsed, setCollapsed] = useState(false)
  const { data: profile } = useProfile()

  const handleLogout = async () => {
    await auth.logout()
    router.push('/login')
  }

  const currentNav = navItems.find((i) => pathname === i.href || pathname.startsWith(`${i.href}/`))

  const initials = profile ? `${profile.firstName.charAt(0)}${profile.lastName.charAt(0)}` : 'AD'

  const displayName = profile ? `${profile.firstName} ${profile.lastName}` : 'Admin'

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <aside
        className={cn(
          'flex h-full flex-col bg-[#0f172a] text-white transition-all duration-300',
          collapsed ? 'w-[68px]' : 'w-64'
        )}
      >
        {/* Brand */}
        <div className="flex h-16 items-center gap-3 border-b border-white/10 px-4">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-indigo-500 text-white font-bold text-sm">
            S
          </div>
          {!collapsed && <span className="text-sm font-semibold truncate">Slate Admin</span>}
          {!collapsed && (
            <button
              onClick={() => setCollapsed(true)}
              className="ml-auto rounded-md p-1 text-slate-400 hover:bg-slate-800 hover:text-white"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
          )}
        </div>

        {/* Nav */}
        <nav className="flex-1 space-y-1 overflow-y-auto p-3">
          {navItems.map((item) => {
            const isActive = pathname === item.href || pathname.startsWith(`${item.href}/`)
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-indigo-500/10 text-indigo-400'
                    : 'text-slate-400 hover:bg-slate-800 hover:text-white',
                  collapsed && 'justify-center px-2'
                )}
                title={collapsed ? item.name : undefined}
              >
                <item.icon className={cn('h-5 w-5 shrink-0', isActive && 'text-indigo-400')} />
                {!collapsed && <span>{item.name}</span>}
              </Link>
            )
          })}
        </nav>

        {/* User + Logout */}
        <div className="border-t border-white/10 p-3 space-y-1">
          {!collapsed && (
            <div className="flex items-center gap-3 px-3 py-2">
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-indigo-500/20 text-xs font-medium text-indigo-300">
                {initials}
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-slate-200">{displayName}</p>
                <p className="truncate text-xs text-slate-500">{profile?.email ?? ''}</p>
              </div>
            </div>
          )}
          <button
            onClick={handleLogout}
            className={cn(
              'flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-slate-400 hover:bg-slate-800 hover:text-white transition-colors',
              collapsed && 'justify-center px-2'
            )}
          >
            <LogOut className="h-5 w-5 shrink-0" />
            {!collapsed && <span>Logout</span>}
          </button>
        </div>

        {/* Expand */}
        {collapsed && (
          <div className="border-t border-white/10 p-3">
            <button
              onClick={() => setCollapsed(false)}
              className="flex w-full items-center justify-center rounded-lg px-2 py-2.5 text-slate-400 hover:bg-slate-800 hover:text-white"
            >
              <ChevronLeft className="h-5 w-5 rotate-180" />
            </button>
          </div>
        )}
      </aside>

      {/* Main area */}
      <div className="flex flex-1 flex-col overflow-hidden">
        <ImpersonationBanner />
        {/* Top header */}
        <header className="flex h-16 items-center justify-between border-b border-[var(--color-border)] bg-white px-6">
          <div className="text-sm font-medium text-[var(--color-text)]">
            {currentNav?.name || 'Dashboard'}
          </div>
          <div className="flex items-center gap-3">
            <button className="relative rounded-lg p-2 text-[var(--color-text-muted)] hover:bg-[var(--color-bg-muted)]">
              <Bell className="h-5 w-5" />
              <span className="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-[var(--color-error)]" />
            </button>
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-indigo-100 text-sm font-medium text-indigo-600">
              {initials}
            </div>
          </div>
        </header>

        {/* Content */}
        <main className="flex-1 overflow-y-auto bg-[var(--color-bg-muted)] p-6">{children}</main>
      </div>
    </div>
  )
}

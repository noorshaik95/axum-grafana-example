'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import {
  LayoutDashboard,
  BookOpen,
  ClipboardList,
  GraduationCap,
  Video,
  Megaphone,
  MessageSquare,
  User,
  Bell,
  LogOut,
  ChevronLeft,
  Menu,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { useState } from 'react';
import { useStudentProfile } from '../../lib/api/profile';
import { auth } from '../../../shared/lib/api';

const navItems = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Courses', href: '/courses', icon: BookOpen },
  { name: 'Assignments', href: '/assignments', icon: ClipboardList },
  { name: 'Grades', href: '/grades', icon: GraduationCap },
  { name: 'Live Classes', href: '/video', icon: Video },
  { name: 'Announcements', href: '/announcements', icon: Megaphone },
  { name: 'Messages', href: '/messages', icon: MessageSquare },
  { name: 'Profile', href: '/profile', icon: User },
];

export function StudentShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const { data: profile } = useStudentProfile();

  const initials = profile ? `${profile.firstName.charAt(0)}${profile.lastName.charAt(0)}` : 'ST';

  const displayName = profile ? `${profile.firstName} ${profile.lastName}` : 'Student';

  const handleLogout = async () => {
    await auth.logout();
    document.cookie = 'slate_token=; path=/; max-age=0';
    router.push('/login');
  };

  const currentNav = navItems.find((i) => pathname === i.href || pathname.startsWith(`${i.href}/`));

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Mobile overlay */}
      {mobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={() => setMobileOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 flex h-full flex-col bg-slate-900 text-white transition-all duration-300 lg:relative lg:z-auto',
          collapsed ? 'lg:w-[68px]' : 'lg:w-64',
          mobileOpen ? 'w-64 translate-x-0' : '-translate-x-full lg:translate-x-0'
        )}
      >
        {/* Brand */}
        <div className="flex h-16 items-center gap-3 border-b border-white/10 px-4">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-indigo-500 text-white font-bold text-sm">
            S
          </div>
          {!collapsed && <span className="text-sm font-semibold truncate">Student Portal</span>}
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
            const isActive = pathname === item.href || pathname.startsWith(`${item.href}/`);
            return (
              <Link
                key={item.href}
                href={item.href}
                onClick={() => setMobileOpen(false)}
                className={cn(
                  'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-indigo-600 text-white'
                    : 'text-slate-400 hover:bg-slate-800 hover:text-white',
                  collapsed && 'justify-center px-2'
                )}
                title={collapsed ? item.name : undefined}
              >
                <item.icon className="h-5 w-5 shrink-0" />
                {!collapsed && <span>{item.name}</span>}
              </Link>
            );
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
        {/* Top header */}
        <header className="flex h-16 items-center justify-between border-b border-[var(--color-border)] bg-white px-4 sm:px-6">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setMobileOpen(true)}
              className="rounded-lg p-2 text-[var(--color-text-muted)] hover:bg-[var(--color-bg-muted)] lg:hidden"
              aria-label="Open menu"
            >
              <Menu className="h-5 w-5" />
            </button>
            <span className="text-sm font-medium text-[var(--color-text)]">
              {currentNav?.name || 'Dashboard'}
            </span>
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
  );
}

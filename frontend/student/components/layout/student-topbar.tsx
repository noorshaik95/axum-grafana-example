'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useState } from 'react';
import { BookOpen, GraduationCap, Home, Inbox, User, LogOut } from 'lucide-react';
import { useStudentProfile } from '../../lib/api/profile';
import { auth } from '../../../shared/lib/api';

interface NowChip {
  label: string;
  href: string;
  primary?: boolean;
}

const NOW_CHIPS: NowChip[] = [
  { label: 'PS4 due 2h · CS 101', href: '/today', primary: true },
  { label: 'Live: Algorithms 3pm', href: '/video' },
  { label: 'Office hours open', href: '/office-hours' },
];

const BOTTOM_NAV = [
  { name: 'Today', href: '/today', icon: Home },
  { name: 'Courses', href: '/courses', icon: BookOpen },
  { name: 'Inbox', href: '/inbox', icon: Inbox },
  { name: 'Grades', href: '/grades', icon: GraduationCap },
  { name: 'Profile', href: '/profile', icon: User },
];

export function StudentTopbar({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { data: profile } = useStudentProfile();
  const [userMenuOpen, setUserMenuOpen] = useState(false);

  const initials = profile ? `${profile.firstName.charAt(0)}${profile.lastName.charAt(0)}` : 'ST';

  const handleLogout = async () => {
    await auth.logout();
    document.cookie = 'slate_token=; path=/; max-age=0';
    router.push('/login');
  };

  return (
    <div className="flex flex-col min-h-screen" style={{ background: 'var(--cream)' }}>
      {/* Top bar */}
      <header
        style={{
          position: 'sticky',
          top: 0,
          zIndex: 50,
          background: 'rgba(251,250,245,0.85)',
          backdropFilter: 'blur(14px)',
          WebkitBackdropFilter: 'blur(14px)',
          borderBottom: '1px solid var(--border)',
        }}
      >
        <div className="flex h-14 items-center gap-4 px-4 sm:px-6 max-w-7xl mx-auto w-full">
          {/* Brand */}
          <Link href="/today" className="flex items-center gap-2 shrink-0">
            <div
              style={{
                width: 28,
                height: 28,
                background: 'linear-gradient(135deg, var(--forest-500) 0%, var(--forest-800) 100%)',
                borderRadius: 9,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <span className="text-white font-bold text-xs">S</span>
            </div>
            <span
              className="serif italic hidden sm:block"
              style={{ fontSize: 17, color: 'var(--ink)', letterSpacing: '-0.01em' }}
            >
              Slate
            </span>
          </Link>

          {/* Command bar */}
          <div className="flex-1 flex justify-center">
            <button
              className="hidden sm:flex items-center gap-2 px-4 py-1.5 text-sm"
              style={{
                borderRadius: 'var(--r-pill)',
                border: '1px solid var(--border)',
                background: 'rgba(255,255,255,0.6)',
                color: 'var(--muted)',
                maxWidth: 320,
                width: '100%',
                cursor: 'text',
              }}
              aria-label="Open command palette"
            >
              <kbd className="mono" style={{ fontSize: 11, color: 'var(--muted)', opacity: 0.7 }}>
                ⌘K
              </kbd>
              <span style={{ fontSize: 13 }}>Jump anywhere…</span>
            </button>
          </div>

          {/* Right: role pill + avatar */}
          <div className="flex items-center gap-3 shrink-0">
            <div
              className="hidden sm:flex items-center gap-1 px-2 py-1"
              style={{
                borderRadius: 'var(--r-pill)',
                border: '1px solid var(--border)',
                background: 'var(--paper)',
                fontSize: 12,
              }}
            >
              <span
                style={{
                  background: 'var(--forest-700)',
                  color: '#fff',
                  borderRadius: 'var(--r-pill)',
                  padding: '1px 8px',
                  fontSize: 11,
                  fontWeight: 600,
                }}
              >
                Student
              </span>
              <Link
                href="http://localhost:3001"
                style={{ color: 'var(--muted)', padding: '1px 8px', fontSize: 11 }}
              >
                Instructor
              </Link>
            </div>

            {/* Avatar */}
            <div className="relative">
              <button
                onClick={() => setUserMenuOpen((v) => !v)}
                className="flex items-center justify-center rounded-full font-semibold text-sm"
                style={{
                  width: 32,
                  height: 32,
                  background: 'var(--forest-100)',
                  color: 'var(--forest-700)',
                  border: '2px solid var(--forest-200)',
                }}
                aria-label="User menu"
              >
                {initials}
              </button>
              {userMenuOpen && (
                <div
                  className="absolute right-0 mt-2 w-48 py-1"
                  style={{
                    background: '#fff',
                    border: '1px solid var(--border)',
                    borderRadius: 'var(--r-md)',
                    boxShadow: 'var(--shadow-md)',
                    zIndex: 100,
                  }}
                >
                  <div
                    className="px-4 py-2 border-b"
                    style={{ borderColor: 'var(--border)', fontSize: 13 }}
                  >
                    <p className="font-medium" style={{ color: 'var(--ink)' }}>
                      {profile?.firstName} {profile?.lastName}
                    </p>
                    <p style={{ color: 'var(--muted)', fontSize: 11 }}>{profile?.email}</p>
                  </div>
                  <Link
                    href="/profile"
                    className="flex items-center gap-2 px-4 py-2 text-sm hover:bg-[var(--forest-50)]"
                    style={{ color: 'var(--ink)' }}
                    onClick={() => setUserMenuOpen(false)}
                  >
                    <User className="h-3.5 w-3.5" />
                    Profile
                  </Link>
                  <button
                    onClick={handleLogout}
                    className="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-[var(--forest-50)]"
                    style={{ color: 'var(--ink)' }}
                  >
                    <LogOut className="h-3.5 w-3.5" />
                    Sign out
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Now bar */}
        <div
          style={{
            background: 'var(--forest-700)',
            color: '#e9efe9',
            fontSize: 12,
            padding: '6px 24px',
          }}
          className="hidden sm:flex items-center gap-2 overflow-x-auto"
        >
          <span style={{ color: 'var(--forest-300)', marginRight: 4 }}>Today →</span>
          {NOW_CHIPS.map((chip) => (
            <Link
              key={chip.href}
              href={chip.href}
              className="shrink-0 px-3 py-1 rounded-full font-medium transition-opacity hover:opacity-80"
              style={
                chip.primary
                  ? {
                      background: 'var(--amber)',
                      color: 'var(--ink)',
                      fontFamily: 'JetBrains Mono, monospace',
                      fontWeight: 700,
                    }
                  : {
                      background: 'rgba(255,255,255,0.12)',
                      color: '#d0e4d3',
                    }
              }
            >
              {chip.primary && '● '}
              {chip.label}
            </Link>
          ))}
        </div>
      </header>

      {/* Main content */}
      <main className="flex-1 w-full max-w-7xl mx-auto px-4 sm:px-6 py-6 pb-24 sm:pb-6">
        {children}
      </main>

      {/* Mobile bottom tab bar */}
      <nav
        className="sm:hidden fixed bottom-0 inset-x-0 z-50 flex items-center"
        style={{
          background: 'rgba(251,250,245,0.95)',
          backdropFilter: 'blur(12px)',
          borderTop: '1px solid var(--border)',
          height: 64,
        }}
      >
        {BOTTOM_NAV.map((item) => {
          const isActive = pathname === item.href || pathname.startsWith(`${item.href}/`);
          return (
            <Link
              key={item.href}
              href={item.href}
              className="flex-1 flex flex-col items-center justify-center gap-0.5 py-2"
            >
              <item.icon
                style={{
                  width: 22,
                  height: 22,
                  color: isActive ? 'var(--forest-600)' : 'var(--muted)',
                  strokeWidth: isActive ? 2.5 : 1.75,
                }}
              />
              <span
                style={{
                  fontSize: 10,
                  fontWeight: isActive ? 600 : 400,
                  color: isActive ? 'var(--forest-700)' : 'var(--muted)',
                }}
              >
                {item.name}
              </span>
            </Link>
          );
        })}
      </nav>
    </div>
  );
}

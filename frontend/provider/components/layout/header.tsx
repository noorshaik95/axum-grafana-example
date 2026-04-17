'use client'

import { Bell, Settings as SettingsIcon, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { mockInstructor } from '@/lib/mock-data'
import Link from 'next/link'

interface HeaderProps {
  className?: string
}

export function Header({ className }: HeaderProps) {
  return (
    <header className={className}>
      {/* Top Bar with glassmorphism styling (Requirement 12.1) */}
      <div className="flex h-16 items-center justify-between border-b border-white/10 px-6 glass-panel">
        {/* Welcome Message */}
        <div className="flex-1">
          <h2 className="text-2xl font-bold text-slate-50">
            Welcome back, Prof. {mockInstructor.lastName}!
          </h2>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-3">
          {/* Create Course button with aurora glow */}
          <Button
            asChild
            className="bg-gradient-to-r from-aurora-violet to-aurora-magenta hover:from-aurora-violet/90 hover:to-aurora-magenta/90 text-white transition-all duration-300 hover:shadow-[0_0_20px_hsl(270_100%_60%_/_0.4)]"
          >
            <Link href="/courses/create">
              <Plus className="h-4 w-4 mr-1" />
              Create Course
            </Link>
          </Button>

          {/* Notification button with hover glow (Requirement 12.3) */}
          <Button
            variant="ghost"
            size="icon"
            aria-label="Notifications"
            className="relative text-slate-400 hover:text-slate-50 hover:bg-white/5 transition-all duration-300 group"
          >
            <Bell className="h-5 w-5 transition-all duration-300 group-hover:text-aurora-cyan group-hover:drop-shadow-[0_0_8px_hsl(186_100%_50%_/_0.5)]" />
            <Badge
              variant="destructive"
              className="absolute -right-1 -top-1 h-5 w-5 rounded-full p-0 flex items-center justify-center text-xs bg-aurora-magenta border-0"
            >
              5
            </Badge>
          </Button>

          {/* Settings button with hover glow (Requirement 12.3) */}
          <Button
            variant="ghost"
            size="icon"
            asChild
            aria-label="Settings"
            className="text-slate-400 hover:text-slate-50 hover:bg-white/5 transition-all duration-300 group"
          >
            <Link href="/settings">
              <SettingsIcon className="h-5 w-5 transition-all duration-300 group-hover:text-aurora-violet group-hover:drop-shadow-[0_0_8px_hsl(270_100%_60%_/_0.5)]" />
            </Link>
          </Button>

          {/* User Avatar with glass border */}
          <div className="flex items-center gap-2 pl-2 border-l border-white/10">
            <img
              src={mockInstructor.avatar}
              alt={`${mockInstructor.firstName} ${mockInstructor.lastName}`}
              className="h-8 w-8 rounded-full ring-2 ring-aurora-violet/30"
            />
            <div className="hidden md:block text-sm">
              <p className="font-medium text-slate-50">
                {mockInstructor.firstName} {mockInstructor.lastName}
              </p>
              <p className="text-xs text-slate-400">{mockInstructor.department}</p>
            </div>
          </div>
        </div>
      </div>
    </header>
  )
}

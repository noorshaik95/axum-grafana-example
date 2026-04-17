'use client'

import * as React from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { cn } from '../../utils/index'
import { ChevronLeft, LogOut } from 'lucide-react'

interface NavItem {
  name: string
  href: string
  icon: React.ComponentType<{ className?: string }>
}

interface SidebarNavProps {
  items: NavItem[]
  brand: {
    name: string
    subtitle?: string
    icon: React.ReactNode
  }
  onLogout?: () => void
  collapsible?: boolean
}

function SidebarNav({ items, brand, onLogout, collapsible = true }: SidebarNavProps) {
  const pathname = usePathname()
  const [collapsed, setCollapsed] = React.useState(false)

  return (
    <aside
      className={cn(
        'flex h-screen flex-col bg-[var(--color-sidebar)] text-white transition-all duration-300',
        collapsed ? 'w-[68px]' : 'w-64'
      )}
      role="navigation"
      aria-label="Main navigation"
    >
      {/* Brand */}
      <div className="flex h-16 items-center gap-3 border-b border-white/10 px-4">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-[var(--color-primary)] text-white">
          {brand.icon}
        </div>
        {!collapsed && (
          <div className="flex flex-col overflow-hidden">
            <span className="truncate text-sm font-semibold">{brand.name}</span>
            {brand.subtitle && (
              <span className="truncate text-xs text-slate-400">{brand.subtitle}</span>
            )}
          </div>
        )}
        {collapsible && !collapsed && (
          <button
            onClick={() => setCollapsed(true)}
            className="ml-auto rounded-md p-1 text-slate-400 hover:bg-[var(--color-sidebar-hover)] hover:text-white"
            aria-label="Collapse sidebar"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
        )}
      </div>

      {/* Nav items */}
      <nav className="flex-1 space-y-1 overflow-y-auto p-3">
        {items.map((item) => {
          const isActive = pathname === item.href || pathname.startsWith(`${item.href}/`)
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-[var(--color-sidebar-active)] text-white'
                  : 'text-slate-400 hover:bg-[var(--color-sidebar-hover)] hover:text-white',
                collapsed && 'justify-center px-2'
              )}
              aria-current={isActive ? 'page' : undefined}
              title={collapsed ? item.name : undefined}
            >
              <item.icon className="h-5 w-5 shrink-0" aria-hidden="true" />
              {!collapsed && <span>{item.name}</span>}
            </Link>
          )
        })}
      </nav>

      {/* Footer */}
      {onLogout && (
        <div className="border-t border-white/10 p-3">
          <button
            onClick={onLogout}
            className={cn(
              'flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-slate-400 transition-colors hover:bg-[var(--color-sidebar-hover)] hover:text-white',
              collapsed && 'justify-center px-2'
            )}
          >
            <LogOut className="h-5 w-5 shrink-0" aria-hidden="true" />
            {!collapsed && <span>Logout</span>}
          </button>
        </div>
      )}

      {/* Expand button when collapsed */}
      {collapsible && collapsed && (
        <div className="border-t border-white/10 p-3">
          <button
            onClick={() => setCollapsed(false)}
            className="flex w-full items-center justify-center rounded-lg px-2 py-2.5 text-slate-400 hover:bg-[var(--color-sidebar-hover)] hover:text-white"
            aria-label="Expand sidebar"
          >
            <ChevronLeft className="h-5 w-5 rotate-180" />
          </button>
        </div>
      )}
    </aside>
  )
}

export { SidebarNav }
export type { NavItem, SidebarNavProps }

'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import { cn } from '@/lib/utils'
import {
  LayoutDashboard,
  Users,
  Building2,
  Activity,
  CreditCard,
  Shield,
  LogOut,
  Menu,
  ChevronDown,
  ChevronRight,
  UserCog,
  FileText,
  Cpu,
  Bell,
  BookOpen,
  Heart,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { auth } from '../../../shared/lib/api'
import { useState } from 'react'

type NavItem = {
  name: string
  href: string
  icon: React.ElementType
  children?: { name: string; href: string }[]
}

const navigation: NavItem[] = [
  {
    name: 'Dashboard',
    href: '/dashboard',
    icon: LayoutDashboard,
  },
  {
    name: 'Universities',
    href: '/universities',
    icon: Building2,
  },
  {
    name: 'Onboarding',
    href: '/onboarding',
    icon: BookOpen,
  },
  {
    name: 'IAM',
    href: '/iam',
    icon: Shield,
    children: [
      { name: 'Users', href: '/iam/users' },
      { name: 'Roles', href: '/iam/roles' },
      { name: 'Audit Log', href: '/iam/audit' },
    ],
  },
  {
    name: 'Billing',
    href: '/billing',
    icon: CreditCard,
  },
  {
    name: 'System',
    href: '/system',
    icon: Activity,
    children: [
      { name: 'Health', href: '/system/health' },
      { name: 'Kafka', href: '/system/kafka' },
      { name: 'Alerts', href: '/system/alerts' },
    ],
  },
  {
    name: 'Impersonation',
    href: '/impersonation',
    icon: UserCog,
  },
]

function NavGroup({ item, pathname }: { item: NavItem; pathname: string }) {
  const isActive =
    pathname === item.href || (item.href !== '/dashboard' && pathname.startsWith(item.href))
  const [open, setOpen] = useState(isActive)

  if (!item.children) {
    return (
      <Link
        href={item.href}
        className={cn(
          'group flex items-center rounded-lg px-3 py-2 text-sm font-medium transition-colors',
          isActive
            ? 'bg-primary/10 text-primary'
            : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
        )}
      >
        <item.icon
          className={cn(
            'mr-3 h-4 w-4',
            isActive ? 'text-primary' : 'text-muted-foreground group-hover:text-accent-foreground'
          )}
        />
        {item.name}
      </Link>
    )
  }

  return (
    <div>
      <button
        onClick={() => setOpen((v) => !v)}
        className={cn(
          'group flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm font-medium transition-colors',
          isActive
            ? 'bg-primary/10 text-primary'
            : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
        )}
      >
        <span className="flex items-center">
          <item.icon
            className={cn(
              'mr-3 h-4 w-4',
              isActive ? 'text-primary' : 'text-muted-foreground group-hover:text-accent-foreground'
            )}
          />
          {item.name}
        </span>
        {open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
      </button>
      {open && (
        <div className="ml-7 mt-1 space-y-0.5 border-l border-border pl-3">
          {item.children.map((child) => {
            const childActive = pathname === child.href || pathname.startsWith(child.href)
            return (
              <Link
                key={child.href}
                href={child.href}
                className={cn(
                  'block rounded-md px-2 py-1.5 text-sm transition-colors',
                  childActive
                    ? 'text-primary font-medium'
                    : 'text-muted-foreground hover:text-foreground'
                )}
              >
                {child.name}
              </Link>
            )
          })}
        </div>
      )}
    </div>
  )
}

export function Sidebar() {
  const pathname = usePathname()
  const router = useRouter()
  const [isOpen, setIsOpen] = useState(true)

  const handleLogout = async () => {
    await auth.logout()
    router.push('/login')
  }

  return (
    <>
      <Button
        variant="ghost"
        size="icon"
        className="fixed top-4 left-4 z-50 lg:hidden rounded-lg"
        onClick={() => setIsOpen(!isOpen)}
      >
        <Menu className="h-6 w-6" />
      </Button>

      <aside
        className={cn(
          'fixed left-0 top-0 z-40 h-screen w-64 border-r border-border bg-card transition-transform lg:translate-x-0',
          isOpen ? 'translate-x-0' : '-translate-x-full'
        )}
      >
        <div className="flex h-full flex-col">
          <div className="flex h-16 items-center border-b border-border px-6">
            <Shield className="h-6 w-6 text-primary" />
            <span className="ml-2 text-lg font-semibold">Slate Admin</span>
          </div>

          <nav className="flex-1 space-y-1 overflow-y-auto p-4">
            {navigation.map((item) => (
              <NavGroup key={item.href} item={item} pathname={pathname} />
            ))}
          </nav>

          <div className="border-t border-border p-4">
            <Button
              variant="ghost"
              className="w-full justify-start text-muted-foreground hover:text-foreground"
              onClick={handleLogout}
            >
              <LogOut className="mr-3 h-5 w-5" />
              Logout
            </Button>
          </div>
        </div>
      </aside>
    </>
  )
}

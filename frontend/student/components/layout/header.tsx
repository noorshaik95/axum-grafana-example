'use client';

import { Bell, Search } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ThemeToggle } from '@/components/layout/theme-toggle';
import { Breadcrumbs } from '@/components/layout/breadcrumbs';

interface HeaderProps {
  className?: string;
}

export function Header({ className }: HeaderProps) {
  return (
    <header className={className}>
      {/* Top Bar */}
      <div className="flex h-16 items-center justify-between border-b px-6">
        {/* Search */}
        <div className="flex flex-1 items-center gap-4">
          <div className="relative w-full max-w-md">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="search"
              placeholder="Search courses, assignments..."
              className="pl-10"
              aria-label="Search"
            />
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="icon" aria-label="Notifications" className="relative">
            <Bell className="h-5 w-5" />
            <span className="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-red-600" />
          </Button>

          <ThemeToggle />
        </div>
      </div>

      {/* Breadcrumbs */}
      <div className="border-b px-6 py-3">
        <Breadcrumbs />
      </div>
    </header>
  );
}

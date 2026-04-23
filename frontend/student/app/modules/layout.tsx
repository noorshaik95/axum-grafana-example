'use client';

import { AppShellWrapper } from '@/components/layout/app-shell-wrapper';

export default function ModulesLayout({ children }: { children: React.ReactNode }) {
  return <AppShellWrapper>{children}</AppShellWrapper>;
}

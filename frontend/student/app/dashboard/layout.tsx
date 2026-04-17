'use client';

import { StudentShell } from '@/components/layout/student-shell';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return <StudentShell>{children}</StudentShell>;
}

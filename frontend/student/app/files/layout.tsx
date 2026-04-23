'use client';

import { StudentTopbar } from '@/components/layout/student-topbar';

export default function ShellLayout({ children }: { children: React.ReactNode }) {
  return <StudentTopbar>{children}</StudentTopbar>;
}

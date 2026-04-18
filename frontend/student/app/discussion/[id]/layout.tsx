'use client';

import { StudentTopbar } from '@/components/layout/student-topbar';

export default function DiscussionLayout({ children }: { children: React.ReactNode }) {
  return <StudentTopbar>{children}</StudentTopbar>;
}

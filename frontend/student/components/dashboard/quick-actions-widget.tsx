'use client';

import { Upload, MessageSquare, Calendar, BookOpen, FileText, Video } from 'lucide-react';
import { QuickActionButton } from '@/components/common/quick-action-button';
const quickActions = [
  {
    name: 'Submit Work',
    icon: <Upload className="h-6 w-6" />,
    href: '/assignments',
  },
  {
    name: 'Discussions',
    icon: <MessageSquare className="h-6 w-6" />,
    href: '/discussions',
  },
  {
    name: 'Schedule',
    icon: <Calendar className="h-6 w-6" />,
    href: '/calendar',
  },
  {
    name: 'My Courses',
    icon: <BookOpen className="h-6 w-6" />,
    href: '/courses',
  },
  {
    name: 'View Grades',
    icon: <FileText className="h-6 w-6" />,
    href: '/grades',
  },
  {
    name: 'Live Classes',
    icon: <Video className="h-6 w-6" />,
    href: '/video',
  },
];

export function QuickActionsWidget() {
  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-6">
      {quickActions.map((action) => (
        <QuickActionButton
          key={action.name}
          icon={action.icon}
          label={action.name}
          href={action.href}
        />
      ))}
    </div>
  );
}

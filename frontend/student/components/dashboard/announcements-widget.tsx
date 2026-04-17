'use client';

import Link from 'next/link';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ArrowRight, Bell, AlertCircle, Info } from 'lucide-react';
import { formatRelativeTime } from '@/lib/utils';

const announcements = [
  {
    id: '1',
    title: 'Midterm Schedule Updated',
    course: 'CS 101',
    content:
      'The midterm exam has been rescheduled to next Friday. Please check the updated syllabus.',
    priority: 'high' as const,
    publishedAt: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
    isRead: false,
  },
  {
    id: '2',
    title: 'Office Hours This Week',
    course: 'MATH 201',
    content: 'I will be holding extra office hours on Thursday 2-4 PM for help with problem sets.',
    priority: 'medium' as const,
    publishedAt: new Date(Date.now() - 5 * 60 * 60 * 1000).toISOString(),
    isRead: false,
  },
  {
    id: '3',
    title: 'New Reading Material',
    course: 'ENG 102',
    content: 'Chapter 7 materials have been uploaded to the course page.',
    priority: 'low' as const,
    publishedAt: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
    isRead: true,
  },
];

export function AnnouncementsWidget() {
  // Semantic colors: red for urgent, yellow for warning, blue for info
  const getPriorityIcon = (priority: 'high' | 'medium' | 'low') => {
    switch (priority) {
      case 'high':
        return <AlertCircle className="h-4 w-4 text-red-500" />;
      case 'medium':
        return <Info className="h-4 w-4 text-yellow-500" />;
      default:
        return <Bell className="h-4 w-4 text-blue-500" />;
    }
  };

  const getPriorityVariant = (priority: 'high' | 'medium' | 'low') => {
    switch (priority) {
      case 'high':
        return 'destructive' as const; // Red
      case 'medium':
        return 'warning' as const; // Amber
      default:
        return 'default' as const; // Blue
    }
  };

  const unreadCount = announcements.filter((a) => !a.isRead).length;

  return (
    <Card className="h-full glass-card">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CardTitle>Announcements</CardTitle>
            {unreadCount > 0 && (
              <Badge variant="destructive" className="h-5">
                {unreadCount}
              </Badge>
            )}
          </div>
          <Button asChild variant="ghost" size="sm">
            <Link href="/announcements">
              View All
              <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
          </Button>
        </div>
        <CardDescription>Latest updates from your courses</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {announcements.map((announcement) => (
          <Link
            key={announcement.id}
            href={`/announcements/${announcement.id}`}
            className={`block rounded-xl glass-card p-3 transition-colors hover:bg-white/10 focus-ring`}
          >
            <div className="space-y-2 p-2">
              <div className="flex items-start justify-between gap-2">
                <div className="flex items-start gap-2">
                  {getPriorityIcon(announcement.priority)}
                  <div className="flex-1 space-y-1">
                    <h4 className="text-sm font-medium leading-none">{announcement.title}</h4>
                    <p className="text-xs text-muted-foreground">{announcement.course}</p>
                  </div>
                </div>
                <Badge
                  variant={getPriorityVariant(announcement.priority)}
                  className="text-xs capitalize"
                >
                  {announcement.priority}
                </Badge>
              </div>
              <p className="line-clamp-2 text-xs text-muted-foreground">{announcement.content}</p>
              <p className="text-xs text-muted-foreground">
                {formatRelativeTime(announcement.publishedAt)}
              </p>
            </div>
          </Link>
        ))}
      </CardContent>
    </Card>
  );
}

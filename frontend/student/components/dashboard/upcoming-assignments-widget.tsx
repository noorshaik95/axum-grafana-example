'use client';

import Link from 'next/link';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Clock, ArrowRight, FileText } from 'lucide-react';
import { formatRelativeTime, getDueDateStatus } from '@/lib/utils';

const upcomingAssignments = [
  {
    id: '1',
    title: 'Data Structures Project',
    course: 'CS 101',
    dueDate: new Date(Date.now() + 2 * 24 * 60 * 60 * 1000).toISOString(),
    points: 100,
    status: 'pending' as const,
  },
  {
    id: '2',
    title: 'Calculus Problem Set 5',
    course: 'MATH 201',
    dueDate: new Date(Date.now() + 5 * 24 * 60 * 60 * 1000).toISOString(),
    points: 50,
    status: 'in-progress' as const,
  },
  {
    id: '3',
    title: 'Essay: Modern Literature',
    course: 'ENG 102',
    dueDate: new Date(Date.now() + 1 * 24 * 60 * 60 * 1000).toISOString(),
    points: 75,
    status: 'pending' as const,
  },
  {
    id: '4',
    title: 'Lab Report: Kinematics',
    course: 'PHY 201',
    dueDate: new Date(Date.now() + 3 * 24 * 60 * 60 * 1000).toISOString(),
    points: 60,
    status: 'pending' as const,
  },
];

export function UpcomingAssignmentsWidget() {
  // Semantic colors: red for urgent (<=1 day), yellow for warning (<=3 days), slate for normal
  const getDueDateColor = (dueDate: string) => {
    const status = getDueDateStatus(dueDate);
    switch (status) {
      case 'error':
        return 'text-red-500 font-semibold';
      case 'warning':
        return 'text-yellow-500 font-medium';
      default:
        return 'text-muted-foreground';
    }
  };

  return (
    <Card className="h-full">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>Upcoming Assignments</CardTitle>
            <CardDescription>Next 7 days</CardDescription>
          </div>
          <Button asChild variant="ghost" size="sm">
            <Link href="/assignments">
              View All
              <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {upcomingAssignments.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <FileText className="mb-2 h-12 w-12 text-muted-foreground/50" />
            <p className="text-sm text-muted-foreground">No upcoming assignments</p>
          </div>
        ) : (
          upcomingAssignments.map((assignment) => (
            <Link
              key={assignment.id}
              href={`/assignments/${assignment.id}`}
              className="block rounded-xl glass-card p-3 transition-colors hover:bg-white/10 focus-ring"
            >
              <div className="flex items-start justify-between gap-2 p-2">
                <div className="flex-1 space-y-1">
                  <h4 className="text-sm font-medium leading-none">{assignment.title}</h4>
                  <p className="text-xs text-muted-foreground">{assignment.course}</p>
                  <div className="flex items-center gap-1 text-xs">
                    <Clock className={`h-3 w-3 ${getDueDateColor(assignment.dueDate)}`} />
                    <span className={getDueDateColor(assignment.dueDate)}>
                      Due {formatRelativeTime(assignment.dueDate)}
                    </span>
                  </div>
                </div>
                <div className="flex flex-col items-end gap-1">
                  <Badge
                    variant={assignment.status === 'pending' ? 'secondary' : 'default'}
                    className="text-xs"
                  >
                    {assignment.points} pts
                  </Badge>
                  {assignment.status === 'in-progress' && (
                    <Badge variant="outline" className="text-xs">
                      Draft saved
                    </Badge>
                  )}
                </div>
              </div>
            </Link>
          ))
        )}
      </CardContent>
    </Card>
  );
}

'use client';

import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Calendar as CalendarIcon } from 'lucide-react';
import { ScheduleEvent } from '@/components/common/schedule-event';

const todayEvents = [
  {
    id: '1',
    title: 'CS 101 Lecture',
    type: 'class' as const,
    time: '09:00 AM',
    endTime: '10:30 AM',
    location: 'Room 301',
    course: 'CS 101',
  },
  {
    id: '2',
    title: 'MATH 201 Tutorial',
    type: 'class' as const,
    time: '02:00 PM',
    endTime: '03:00 PM',
    location: 'Room 205',
    course: 'MATH 201',
  },
  {
    id: '3',
    title: 'Assignment Due: Physics Lab',
    type: 'deadline' as const,
    time: '11:59 PM',
    endTime: '11:59 PM',
    location: 'Online',
    course: 'PHY 201',
  },
];

const formatTodayDate = () => {
  const today = new Date();
  const options: Intl.DateTimeFormatOptions = {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  };
  return today.toLocaleDateString('en-US', options);
};

export function CalendarWidget() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-foreground">Today&apos;s Schedule</h2>
          <p className="text-sm text-muted-foreground mt-1">{formatTodayDate()}</p>
        </div>
        <Button asChild variant="ghost" size="icon" className="hover:bg-muted">
          <Link href="/calendar" aria-label="View full calendar">
            <CalendarIcon className="h-5 w-5" />
          </Link>
        </Button>
      </div>

      {/* Today's Events */}
      <div className="space-y-4">
        {todayEvents.length === 0 ? (
          <p className="text-sm text-muted-foreground">No events scheduled</p>
        ) : (
          todayEvents.map((event) => (
            <ScheduleEvent key={event.id} event={event} variant="detailed" />
          ))
        )}
      </div>
    </div>
  );
}

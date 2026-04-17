'use client';

import { useMemo } from 'react';
import { useActivityCalendar } from '@/lib/api/hooks';
import { Loader2 } from 'lucide-react';

interface ActivityCalendarProps {
  days?: number;
}

function getIntensity(count: number, maxCount: number): string {
  if (count === 0) return 'bg-gray-100';
  const ratio = count / maxCount;
  if (ratio <= 0.25) return 'bg-green-200';
  if (ratio <= 0.5) return 'bg-green-400';
  if (ratio <= 0.75) return 'bg-green-500';
  return 'bg-green-700';
}

export function ActivityCalendar({ days = 365 }: ActivityCalendarProps) {
  const { data: activityData, isLoading } = useActivityCalendar(days);

  const { grid, months, maxCount } = useMemo(() => {
    const activityMap = new Map<string, number>();
    if (activityData) {
      activityData.forEach((d) => activityMap.set(d.date, d.count));
    }

    const today = new Date();
    const totalDays = Math.min(days, 365);
    const cells: { date: string; count: number; dayOfWeek: number }[] = [];
    let max = 1;

    for (let i = totalDays - 1; i >= 0; i--) {
      const date = new Date(today);
      date.setDate(date.getDate() - i);
      const dateStr = date.toISOString().split('T')[0];
      const count = activityMap.get(dateStr) ?? 0;
      if (count > max) max = count;
      cells.push({ date: dateStr, count, dayOfWeek: date.getDay() });
    }

    // Group into weeks (columns)
    const weeks: (typeof cells)[] = [];
    let currentWeek: typeof cells = [];
    cells.forEach((cell) => {
      if (cell.dayOfWeek === 0 && currentWeek.length > 0) {
        weeks.push(currentWeek);
        currentWeek = [];
      }
      currentWeek.push(cell);
    });
    if (currentWeek.length > 0) weeks.push(currentWeek);

    // Calculate month labels
    const monthLabels: { label: string; weekIdx: number }[] = [];
    let lastMonth = -1;
    weeks.forEach((week, idx) => {
      const firstCell = week[0];
      const month = new Date(firstCell.date).getMonth();
      if (month !== lastMonth) {
        monthLabels.push({
          label: new Date(firstCell.date).toLocaleDateString('en-US', {
            month: 'short',
          }),
          weekIdx: idx,
        });
        lastMonth = month;
      }
    });

    return { grid: weeks, months: monthLabels, maxCount: max };
  }, [activityData, days]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-[var(--color-text-muted)]" />
      </div>
    );
  }

  const totalActivity = activityData?.reduce((sum, d) => sum + d.count, 0) ?? 0;

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-sm font-medium text-[var(--color-text)]">Activity Calendar</p>
        <span className="text-xs text-[var(--color-text-muted)]">
          {totalActivity} total activities
        </span>
      </div>

      <div className="overflow-x-auto">
        {/* Month labels */}
        <div className="flex gap-0 mb-1 ml-8" style={{ minWidth: grid.length * 14 }}>
          {months.map((m, i) => (
            <span
              key={i}
              className="text-[10px] text-[var(--color-text-muted)]"
              style={{
                position: 'relative',
                left: m.weekIdx * 14,
                width: 0,
                whiteSpace: 'nowrap',
              }}
            >
              {m.label}
            </span>
          ))}
        </div>

        <div className="flex gap-[2px]">
          {/* Day labels */}
          <div className="flex flex-col gap-[2px] shrink-0 pr-1">
            {['', 'Mon', '', 'Wed', '', 'Fri', ''].map((label, i) => (
              <div key={i} className="h-[12px] flex items-center justify-end">
                <span className="text-[10px] text-[var(--color-text-muted)]">{label}</span>
              </div>
            ))}
          </div>

          {/* Grid */}
          {grid.map((week, weekIdx) => (
            <div key={weekIdx} className="flex flex-col gap-[2px]">
              {/* Pad first week */}
              {weekIdx === 0 &&
                Array.from({ length: week[0].dayOfWeek }).map((_, i) => (
                  <div key={`pad-${i}`} className="h-[12px] w-[12px]" />
                ))}
              {week.map((cell) => (
                <div
                  key={cell.date}
                  className={`h-[12px] w-[12px] rounded-[2px] ${getIntensity(
                    cell.count,
                    maxCount
                  )}`}
                  title={`${cell.date}: ${cell.count} activities`}
                />
              ))}
            </div>
          ))}
        </div>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-1 justify-end">
        <span className="text-[10px] text-[var(--color-text-muted)] mr-1">Less</span>
        {['bg-gray-100', 'bg-green-200', 'bg-green-400', 'bg-green-500', 'bg-green-700'].map(
          (cls) => (
            <div key={cls} className={`h-[10px] w-[10px] rounded-[2px] ${cls}`} />
          )
        )}
        <span className="text-[10px] text-[var(--color-text-muted)] ml-1">More</span>
      </div>
    </div>
  );
}

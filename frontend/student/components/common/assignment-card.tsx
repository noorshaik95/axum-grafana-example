import * as React from 'react';
import { cn } from '@/lib/utils';
import { GradientType } from './gradient-card';
import { Button } from '@/components/ui/button';
import { FileText, Type, ClipboardList, FileCheck, Clock, Calendar } from 'lucide-react';

interface AssignmentCardProps {
  assignment: {
    id: string;
    title: string;
    description: string;
    course: {
      code: string;
      gradient: GradientType;
    };
    points: number;
    dueDate: Date;
    type: 'file' | 'text' | 'quiz' | 'exam';
    status: 'pending' | 'submitted' | 'graded';
  };
  onSubmit?: () => void;
  className?: string;
}

const gradientBadgeClasses: Record<GradientType, string> = {
  'blue-cyan': 'gradient-blue-cyan',
  'purple-pink': 'gradient-purple-pink',
  'emerald-teal': 'gradient-emerald-teal',
  'orange-red': 'gradient-orange-red',
  'amber-yellow': 'gradient-amber-yellow',
  'violet-indigo': 'gradient-violet-indigo',
  'indigo-purple': 'gradient-indigo-purple',
};

const typeIcons = {
  file: FileText,
  text: Type,
  quiz: ClipboardList,
  exam: FileCheck,
};

const getPointsBadgeColor = (points: number): string => {
  if (points >= 100) return 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300';
  if (points >= 50) return 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300';
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300';
};

const isDueSoon = (dueDate: Date): boolean => {
  const now = new Date();
  const hoursUntilDue = (dueDate.getTime() - now.getTime()) / (1000 * 60 * 60);
  return hoursUntilDue <= 48 && hoursUntilDue > 0;
};

const formatDueDate = (dueDate: Date): string => {
  const options: Intl.DateTimeFormatOptions = {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  };
  return dueDate.toLocaleDateString('en-US', options);
};

export const AssignmentCard = React.forwardRef<HTMLDivElement, AssignmentCardProps>(
  ({ assignment, onSubmit, className }, ref) => {
    const TypeIcon = typeIcons[assignment.type];
    const dueSoon = isDueSoon(assignment.dueDate);
    const pointsBadgeColor = getPointsBadgeColor(assignment.points);

    return (
      <div
        ref={ref}
        className={cn(
          'bg-white dark:bg-gray-800 rounded-2xl border border-gray-200 dark:border-gray-700',
          'shadow-sm hover:shadow-md transition-all duration-300 p-6',
          className
        )}
      >
        {/* Header with Course Badge and Points */}
        <div className="flex items-start justify-between mb-3">
          <div
            className={cn(
              'px-3 py-1 rounded-lg text-white font-semibold text-sm',
              gradientBadgeClasses[assignment.course.gradient]
            )}
          >
            {assignment.course.code}
          </div>
          <div className={cn('px-3 py-1 rounded-lg font-semibold text-sm', pointsBadgeColor)}>
            {assignment.points} pts
          </div>
        </div>

        {/* Assignment Title and Type */}
        <div className="flex items-start gap-3 mb-2">
          <div className="mt-1">
            <TypeIcon className="w-5 h-5 text-gray-600 dark:text-gray-400" />
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-bold text-gray-900 dark:text-white mb-1">
              {assignment.title}
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400 line-clamp-2">
              {assignment.description}
            </p>
          </div>
        </div>

        {/* Due Date */}
        <div
          className={cn(
            'flex items-center gap-2 mb-4 text-sm font-medium',
            dueSoon ? 'text-red-600 dark:text-red-400' : 'text-gray-600 dark:text-gray-400'
          )}
        >
          {dueSoon ? <Clock className="w-4 h-4" /> : <Calendar className="w-4 h-4" />}
          <span>Due {formatDueDate(assignment.dueDate)}</span>
          {dueSoon && <span className="text-xs">(Due Soon!)</span>}
        </div>

        {/* Submit Button */}
        {assignment.status === 'pending' && (
          <Button
            onClick={onSubmit}
            className={cn('w-full font-semibold', gradientBadgeClasses[assignment.course.gradient])}
          >
            Submit Assignment
          </Button>
        )}

        {assignment.status === 'submitted' && (
          <div className="w-full py-2 px-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 text-center font-medium text-sm">
            Submitted - Awaiting Grade
          </div>
        )}

        {assignment.status === 'graded' && (
          <div className="w-full py-2 px-4 rounded-lg bg-emerald-50 dark:bg-emerald-900/20 text-emerald-700 dark:text-emerald-300 text-center font-medium text-sm">
            Graded
          </div>
        )}
      </div>
    );
  }
);

AssignmentCard.displayName = 'AssignmentCard';

import * as React from 'react';
import { cn } from '@/lib/utils';
import { GradientType } from './gradient-card';
import { Progress } from '@/components/ui/progress';
import { Button } from '@/components/ui/button';
import { ArrowRight, Users, BookOpen, Calendar } from 'lucide-react';

// Semantic colors for deadline text (parses "Next deadline in X days" format)
const getDeadlineColor = (deadlineText: string): string => {
  const match = deadlineText.match(/(\d+)\s*day/i);
  if (!match) return 'text-muted-foreground';
  const days = parseInt(match[1], 10);
  if (days <= 1) return 'text-red-500 font-semibold'; // Urgent
  if (days <= 3) return 'text-yellow-500 font-medium'; // Warning
  return 'text-muted-foreground'; // Normal
};

interface CourseCardProps {
  course: {
    id: string;
    code: string;
    name: string;
    instructor: string;
    progress: number;
    gradient: GradientType;
    studentCount?: number;
    credits?: number;
    nextDeadline?: string;
  };
  variant?: 'compact' | 'detailed';
  showProgress?: boolean;
  showMetadata?: boolean;
  className?: string;
  onContinue?: () => void;
}

export const CourseCard = React.forwardRef<HTMLDivElement, CourseCardProps>(
  (
    {
      course,
      variant = 'detailed',
      showProgress = true,
      showMetadata = true,
      className,
      onContinue,
    },
    ref
  ) => {
    return (
      <div
        ref={ref}
        className={cn(
          'glass-card rounded-xl',
          'transform-gpu transition-all duration-300 ease-out',
          'hover:-translate-y-1 hover:shadow-lg',
          className
        )}
      >
        {/* Responsive padding: smaller on mobile (Requirement 13.2, 13.3) */}
        <div className="p-4 sm:p-6">
          {/* Course Code Badge - slate themed */}
          <div className="inline-block px-2.5 sm:px-3 py-1 sm:py-1.5 rounded-full text-xs sm:text-sm font-semibold mb-3 sm:mb-4 bg-blue-500 text-white">
            {course.code}
          </div>

          {/* Course Name - responsive text size (Requirement 13.3) */}
          <h3 className="text-xl sm:text-2xl font-semibold text-foreground mb-2 tracking-tight leading-tight">
            {course.name}
          </h3>

          {/* Instructor - minimum 14px font size (Requirement 13.3) */}
          <p className="text-sm text-muted-foreground mb-3 sm:mb-4">{course.instructor}</p>

          {/* Metadata - responsive layout for mobile (Requirement 13.2) */}
          {showMetadata && variant === 'detailed' && (
            <div className="flex flex-wrap items-center gap-2 sm:gap-4 mb-3 sm:mb-4 text-xs sm:text-sm text-muted-foreground">
              {course.studentCount !== undefined && (
                <div className="flex items-center gap-1 sm:gap-1.5">
                  <Users className="w-3.5 h-3.5 sm:w-4 sm:h-4" />
                  <span>{course.studentCount} students</span>
                </div>
              )}
              {course.credits !== undefined && (
                <div className="flex items-center gap-1 sm:gap-1.5">
                  <BookOpen className="w-3.5 h-3.5 sm:w-4 sm:h-4" />
                  <span>{course.credits} credits</span>
                </div>
              )}
              {course.nextDeadline && (
                <div
                  className={`flex items-center gap-1 sm:gap-1.5 ${getDeadlineColor(course.nextDeadline)}`}
                >
                  <Calendar className="w-3.5 h-3.5 sm:w-4 sm:h-4" />
                  <span>{course.nextDeadline}</span>
                </div>
              )}
            </div>
          )}

          {/* Progress Section - slate themed */}
          {showProgress && (
            <div className="mb-3 sm:mb-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs sm:text-sm font-medium text-muted-foreground">
                  Progress
                </span>
                <span className="text-xl sm:text-2xl font-bold text-foreground">
                  {course.progress}%
                </span>
              </div>
              <Progress value={course.progress} className="h-2" />
            </div>
          )}

          {/* Continue Button - blue accent */}
          <Button
            onClick={onContinue}
            className="w-full mt-2 min-h-[44px] bg-blue-500 hover:bg-blue-600 text-white"
          >
            Continue
            <ArrowRight className="w-4 h-4 ml-2" />
          </Button>
        </div>
      </div>
    );
  }
);

CourseCard.displayName = 'CourseCard';

import * as React from 'react';
import { cn, getGradeColor } from '@/lib/utils';
import { GradientType } from './gradient-card';
import { AnimatedProgress } from './animated-progress';
import { FileText, Calendar } from 'lucide-react';

interface GradeCardProps {
  course: {
    id: string;
    code: string;
    name: string;
    instructor: string;
    gradient: GradientType;
    grade: number;
    progress: number;
  };
  assignments: Array<{
    id: string;
    name: string;
    type: string;
    grade: number;
    maxPoints: number;
    date: Date;
    feedback?: string;
  }>;
  className?: string;
}

const formatDate = (date: Date): string => {
  const options: Intl.DateTimeFormatOptions = {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  };
  return date.toLocaleDateString('en-US', options);
};

export const GradeCard = React.forwardRef<HTMLDivElement, GradeCardProps>(
  ({ course, assignments, className }, ref) => {
    return (
      <div ref={ref} className={cn('glass-card rounded-2xl p-6', className)}>
        {/* Course Header */}
        <div className="flex items-start justify-between mb-4">
          <div className="flex-1">
            <div className="inline-block px-3 py-1 rounded-lg bg-blue-500 text-white font-semibold text-sm mb-2">
              {course.code}
            </div>
            <h3 className="text-xl font-bold text-foreground mb-1">{course.name}</h3>
            <p className="text-sm text-muted-foreground">{course.instructor}</p>
          </div>
          <div className="text-right">
            <div className={`text-4xl font-bold ${getGradeColor(course.grade)}`}>
              {course.grade}%
            </div>
            <div className="text-sm text-muted-foreground mt-1">Current Grade</div>
          </div>
        </div>

        {/* Course Progress */}
        <div className="mb-6">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-muted-foreground">Course Progress</span>
            <span className="text-sm font-semibold text-foreground">{course.progress}%</span>
          </div>
          <AnimatedProgress
            value={course.progress}
            gradient={course.gradient}
            size="md"
            animated={true}
          />
        </div>

        {/* Graded Assignments */}
        <div>
          <h4 className="text-sm font-semibold text-foreground mb-3">Graded Assignments</h4>
          <div className="space-y-3">
            {assignments.map((assignment) => (
              <div key={assignment.id} className="rounded-lg border border-border bg-muted/30 p-4">
                <div className="flex items-start justify-between mb-2">
                  <div className="flex items-start gap-2 flex-1">
                    <FileText className="w-4 h-4 mt-0.5 text-muted-foreground" />
                    <div>
                      <h5 className="font-semibold text-foreground text-sm">{assignment.name}</h5>
                      <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
                        <Calendar className="w-3 h-3" />
                        <span>{formatDate(assignment.date)}</span>
                        <span>•</span>
                        <span className="capitalize">{assignment.type}</span>
                      </div>
                    </div>
                  </div>
                  <div className="text-right">
                    <div
                      className={`text-lg font-bold ${getGradeColor(Math.round((assignment.grade / assignment.maxPoints) * 100))}`}
                    >
                      {assignment.grade}/{assignment.maxPoints}
                    </div>
                    <div
                      className={`text-xs ${getGradeColor(Math.round((assignment.grade / assignment.maxPoints) * 100))}`}
                    >
                      {Math.round((assignment.grade / assignment.maxPoints) * 100)}%
                    </div>
                  </div>
                </div>
                {assignment.feedback && (
                  <div className="mt-2 pt-2 border-t border-border">
                    <p className="text-sm text-blue-600 dark:text-blue-400 italic">
                      &quot;{assignment.feedback}&quot;
                    </p>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  }
);

GradeCard.displayName = 'GradeCard';

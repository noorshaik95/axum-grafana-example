'use client';

import Link from 'next/link';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ArrowRight, TrendingUp, TrendingDown, Minus } from 'lucide-react';
import { getGradeColor, getGradeLetter, getGradeBadgeClass } from '@/lib/utils';

const recentGrades = [
  {
    id: '1',
    assignmentName: 'Midterm Exam',
    course: 'CS 101',
    grade: 92,
    maxGrade: 100,
    gradedAt: '2 days ago',
    trend: 'up' as const,
  },
  {
    id: '2',
    assignmentName: 'Problem Set 4',
    course: 'MATH 201',
    grade: 85,
    maxGrade: 100,
    gradedAt: '3 days ago',
    trend: 'neutral' as const,
  },
  {
    id: '3',
    assignmentName: 'Research Paper',
    course: 'ENG 102',
    grade: 88,
    maxGrade: 100,
    gradedAt: '5 days ago',
    trend: 'down' as const,
  },
  {
    id: '4',
    assignmentName: 'Lab Practical',
    course: 'PHY 201',
    grade: 95,
    maxGrade: 100,
    gradedAt: '1 week ago',
    trend: 'up' as const,
  },
];

export function RecentGradesWidget() {
  // Semantic colors: blue for up, red for down, slate for neutral
  const getTrendIcon = (trend: 'up' | 'down' | 'neutral') => {
    switch (trend) {
      case 'up':
        return <TrendingUp className="h-3 w-3 text-blue-500" />;
      case 'down':
        return <TrendingDown className="h-3 w-3 text-red-500" />;
      default:
        return <Minus className="h-3 w-3 text-slate-500" />;
    }
  };

  return (
    <Card className="h-full">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>Recent Grades</CardTitle>
            <CardDescription>Latest feedback and scores</CardDescription>
          </div>
          <Button asChild variant="ghost" size="sm">
            <Link href="/grades">
              View All
              <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {recentGrades.map((grade) => {
          const percentage = Math.round((grade.grade / grade.maxGrade) * 100);
          return (
            <Link
              key={grade.id}
              href={`/grades/${grade.id}`}
              className="block rounded-xl glass-card p-3 transition-colors hover:bg-white/10 focus-ring"
            >
              <div className="flex items-start justify-between gap-2 p-2">
                <div className="flex-1 space-y-1">
                  <h4 className="text-sm font-medium leading-none">{grade.assignmentName}</h4>
                  <p className="text-xs text-muted-foreground">{grade.course}</p>
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <span>Graded {grade.gradedAt}</span>
                    {getTrendIcon(grade.trend)}
                  </div>
                </div>
                <div className="flex flex-col items-end gap-1">
                  <div className="flex items-baseline gap-1">
                    <span className={`text-lg font-bold ${getGradeColor(percentage)}`}>
                      {percentage}%
                    </span>
                  </div>
                  <Badge className={getGradeBadgeClass(percentage)}>
                    {getGradeLetter(percentage)}
                  </Badge>
                </div>
              </div>
            </Link>
          );
        })}
      </CardContent>
    </Card>
  );
}

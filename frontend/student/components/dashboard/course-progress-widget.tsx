'use client';

import Link from 'next/link';
import { ArrowRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { CourseCard } from '@/components/common/course-card';
import { GradientType } from '@/components/common/gradient-card';
import { useRouter } from 'next/navigation';

const enrolledCourses = [
  {
    id: '1',
    code: 'CS 101',
    name: 'Introduction to Computer Science',
    progress: 75,
    instructor: 'Dr. Smith',
    nextDeadline: 'Next deadline in 2 days',
    gradient: 'blue-cyan' as GradientType,
    studentCount: 45,
    credits: 4,
  },
  {
    id: '2',
    code: 'MATH 201',
    name: 'Calculus II',
    progress: 60,
    instructor: 'Prof. Johnson',
    nextDeadline: 'Next deadline in 5 days',
    gradient: 'purple-pink' as GradientType,
    studentCount: 38,
    credits: 4,
  },
  {
    id: '3',
    code: 'ENG 102',
    name: 'English Composition',
    progress: 90,
    instructor: 'Dr. Williams',
    nextDeadline: 'Next deadline in 1 day',
    gradient: 'emerald-teal' as GradientType,
    studentCount: 32,
    credits: 3,
  },
  {
    id: '4',
    code: 'PHY 201',
    name: 'Physics I',
    progress: 45,
    instructor: 'Dr. Brown',
    nextDeadline: 'Next deadline in 3 days',
    gradient: 'orange-red' as GradientType,
    studentCount: 41,
    credits: 4,
  },
];

export function CourseProgressWidget() {
  const router = useRouter();

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-foreground">My Courses</h2>
          <p className="text-sm text-muted-foreground mt-1">
            Track your progress across all enrolled courses
          </p>
        </div>
        <Button asChild variant="outline" size="sm" className="glass-card hover:bg-muted">
          <Link href="/courses">
            View All
            <ArrowRight className="ml-2 h-4 w-4" />
          </Link>
        </Button>
      </div>

      {/* Course Cards Grid */}
      <div className="grid gap-6 md:grid-cols-2">
        {enrolledCourses.map((course) => (
          <CourseCard
            key={course.id}
            course={course}
            variant="detailed"
            showProgress={true}
            showMetadata={true}
            onContinue={() => router.push(`/courses/${course.id}`)}
          />
        ))}
      </div>
    </div>
  );
}

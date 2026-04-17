'use client'

import * as React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ArrowRight, Users, TrendingUp, FileText } from 'lucide-react'
import { mockCourses } from '@/lib/mock-data'
import { cn } from '@/lib/utils'

const gradientClasses: Record<string, string> = {
  'blue-cyan': 'from-blue-500 to-cyan-500',
  'purple-pink': 'from-purple-500 to-pink-500',
  'emerald-teal': 'from-emerald-500 to-teal-500',
  'orange-red': 'from-orange-500 to-red-500',
  'amber-yellow': 'from-amber-500 to-yellow-500',
  'violet-indigo': 'from-violet-500 to-indigo-500',
  'indigo-purple': 'from-indigo-500 to-purple-500',
}

export function CourseOverviewWidget() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Active Courses</CardTitle>
        <CardDescription>Manage your courses and track progress</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {mockCourses.map((course) => (
            <div
              key={course.id}
              className="border rounded-lg p-4 hover:shadow-md transition-shadow"
            >
              <div className="flex items-start justify-between mb-3">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    <div
                      className={cn(
                        'px-3 py-1 rounded-md text-white text-sm font-semibold bg-gradient-to-r',
                        gradientClasses[course.gradient]
                      )}
                    >
                      {course.code}
                    </div>
                    <Badge variant="outline">{course.section}</Badge>
                  </div>
                  <h4 className="font-semibold text-gray-900 dark:text-white">{course.name}</h4>
                </div>
              </div>

              <div className="grid grid-cols-3 gap-4 mb-4">
                <div className="flex items-center gap-2 text-sm">
                  <Users className="w-4 h-4 text-gray-500" />
                  <div>
                    <p className="font-semibold text-gray-900 dark:text-white">
                      {course.studentCount}
                    </p>
                    <p className="text-xs text-gray-500">students</p>
                  </div>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <FileText className="w-4 h-4 text-gray-500" />
                  <div>
                    <p className="font-semibold text-gray-900 dark:text-white">
                      {course.pendingGrading}
                    </p>
                    <p className="text-xs text-gray-500">to grade</p>
                  </div>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <TrendingUp className="w-4 h-4 text-gray-500" />
                  <div>
                    <p className="font-semibold text-gray-900 dark:text-white">
                      {course.averageGrade}%
                    </p>
                    <p className="text-xs text-gray-500">avg grade</p>
                  </div>
                </div>
              </div>

              <div className="mb-3">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm text-gray-600 dark:text-gray-400">Course Progress</span>
                  <span className="text-sm font-semibold text-gray-900 dark:text-white">
                    {course.progress}%
                  </span>
                </div>
                <Progress value={course.progress} className="h-2" />
              </div>

              <Button variant="outline" size="sm" className="w-full">
                Manage Course
                <ArrowRight className="w-4 h-4" />
              </Button>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

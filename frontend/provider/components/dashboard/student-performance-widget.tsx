'use client'

import * as React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { AlertTriangle, TrendingUp, Users, Award } from 'lucide-react'
import { mockStudentPerformance } from '@/lib/mock-data'

export function StudentPerformanceWidget() {
  const { overall, gradeDistribution, atRiskStudents } = mockStudentPerformance

  return (
    <Card>
      <CardHeader>
        <CardTitle>Student Performance</CardTitle>
        <CardDescription>Overview of student progress and grades</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-6">
          {/* Key Metrics */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
                <Award className="w-4 h-4" />
                <span>Average Grade</span>
              </div>
              <div className="text-3xl font-bold text-gray-900 dark:text-white">
                {overall.averageGrade}%
              </div>
            </div>
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
                <Users className="w-4 h-4" />
                <span>Active Students</span>
              </div>
              <div className="text-3xl font-bold text-gray-900 dark:text-white">
                {overall.activeStudents}
                <span className="text-sm font-normal text-gray-500">/{overall.totalStudents}</span>
              </div>
            </div>
          </div>

          {/* Grade Distribution */}
          <div>
            <h4 className="text-sm font-semibold mb-3 text-gray-900 dark:text-white">
              Grade Distribution
            </h4>
            <div className="space-y-2">
              {gradeDistribution.map((grade) => (
                <div key={grade.grade} className="space-y-1">
                  <div className="flex items-center justify-between text-sm">
                    <span className="font-medium text-gray-700 dark:text-gray-300">
                      Grade {grade.grade}
                    </span>
                    <span className="text-gray-600 dark:text-gray-400">
                      {grade.count} students ({grade.percentage}%)
                    </span>
                  </div>
                  <Progress value={grade.percentage} className="h-2" />
                </div>
              ))}
            </div>
          </div>

          {/* At-Risk Students */}
          {atRiskStudents.length > 0 && (
            <div>
              <div className="flex items-center justify-between mb-3">
                <h4 className="text-sm font-semibold text-gray-900 dark:text-white">
                  At-Risk Students
                </h4>
                <Badge variant="destructive">{atRiskStudents.length}</Badge>
              </div>
              <div className="space-y-2">
                {atRiskStudents.slice(0, 3).map((student) => (
                  <div
                    key={student.id}
                    className="border border-red-200 dark:border-red-800 rounded-lg p-3 bg-red-50 dark:bg-red-950/20"
                  >
                    <div className="flex items-center gap-3">
                      <img
                        src={student.avatar}
                        alt={student.name}
                        className="w-8 h-8 rounded-full"
                      />
                      <div className="flex-1">
                        <div className="flex items-center justify-between">
                          <h5 className="text-sm font-semibold text-gray-900 dark:text-white">
                            {student.name}
                          </h5>
                          <Badge variant="outline" className="text-xs">
                            {student.currentGrade}%
                          </Badge>
                        </div>
                        <div className="flex items-center gap-2 mt-1">
                          <Badge variant="outline" className="text-xs">
                            {student.courseName}
                          </Badge>
                          <span className="text-xs text-gray-600 dark:text-gray-400 flex items-center gap-1">
                            <AlertTriangle className="w-3 h-3" />
                            {student.missedAssignments} missed
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Active Rate */}
          <div className="p-4 bg-gradient-to-r from-blue-50 to-cyan-50 dark:from-blue-950/20 dark:to-cyan-950/20 rounded-lg">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <TrendingUp className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                <span className="font-semibold text-gray-900 dark:text-white">
                  Student Engagement
                </span>
              </div>
              <span className="text-2xl font-bold text-blue-600 dark:text-blue-400">
                {Math.round((overall.activeStudents / overall.totalStudents) * 100)}%
              </span>
            </div>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              {overall.activeStudents} of {overall.totalStudents} students are actively
              participating
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

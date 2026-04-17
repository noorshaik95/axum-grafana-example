'use client'

import * as React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { FileText, Clock, AlertCircle } from 'lucide-react'
import { mockGradingQueue } from '@/lib/mock-data'
import { formatRelativeTime } from '@/lib/utils'
import { cn } from '@/lib/utils'

const priorityColors = {
  low: 'default' as const,
  medium: 'warning' as const,
  high: 'destructive' as const,
}

const typeIcons = {
  essay: FileText,
  quiz: FileText,
  project: FileText,
  assignment: FileText,
}

export function GradingQueueWidget() {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span>Grading Queue</span>
          <Badge variant="secondary">{mockGradingQueue.length} items</Badge>
        </CardTitle>
        <CardDescription>Assignments waiting to be graded</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {mockGradingQueue.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              <FileText className="w-12 h-12 mx-auto mb-2 opacity-50" />
              <p>No assignments to grade</p>
            </div>
          ) : (
            mockGradingQueue.map((item) => {
              const Icon = typeIcons[item.assignmentType]
              const isOverdue = new Date(item.dueDate) < new Date()

              return (
                <div
                  key={item.id}
                  className={cn(
                    'border rounded-lg p-3 hover:shadow-md transition-shadow',
                    isOverdue && 'border-red-300 bg-red-50 dark:bg-red-950/20'
                  )}
                >
                  <div className="flex items-start gap-3">
                    <img
                      src={item.studentAvatar}
                      alt={item.studentName}
                      className="w-10 h-10 rounded-full"
                    />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between mb-1">
                        <h4 className="font-semibold text-sm text-gray-900 dark:text-white truncate">
                          {item.studentName}
                        </h4>
                        <Badge variant={priorityColors[item.priority]} className="text-xs">
                          {item.priority}
                        </Badge>
                      </div>
                      <div className="flex items-center gap-1 mb-1">
                        <Icon className="w-3 h-3 text-gray-500" />
                        <p className="text-sm text-gray-700 dark:text-gray-300 truncate">
                          {item.assignmentName}
                        </p>
                      </div>
                      <div className="flex items-center gap-3 text-xs text-gray-500">
                        <span>{item.courseName}</span>
                        <span>•</span>
                        <span className="flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          {formatRelativeTime(item.submittedAt)}
                        </span>
                        {isOverdue && (
                          <>
                            <span>•</span>
                            <span className="flex items-center gap-1 text-red-600">
                              <AlertCircle className="w-3 h-3" />
                              Overdue
                            </span>
                          </>
                        )}
                      </div>
                      <div className="mt-2">
                        <Button size="sm" className="w-full">
                          Grade Now
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>
              )
            })
          )}
        </div>
      </CardContent>
    </Card>
  )
}

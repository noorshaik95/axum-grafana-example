'use client'

import * as React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { FileText, MessageCircle, CheckCircle, MessageSquare } from 'lucide-react'
import { mockRecentActivity } from '@/lib/mock-data'
import { formatRelativeTime } from '@/lib/utils'

const activityIcons = {
  submission: FileText,
  question: MessageCircle,
  completion: CheckCircle,
  discussion: MessageSquare,
}

const activityColors = {
  submission: 'bg-blue-100 dark:bg-blue-950 text-blue-600 dark:text-blue-400',
  question: 'bg-purple-100 dark:bg-purple-950 text-purple-600 dark:text-purple-400',
  completion: 'bg-green-100 dark:bg-green-950 text-green-600 dark:text-green-400',
  discussion: 'bg-orange-100 dark:bg-orange-950 text-orange-600 dark:text-orange-400',
}

export function RecentActivityWidget() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Recent Activity</CardTitle>
        <CardDescription>Latest updates from your students</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {mockRecentActivity.map((activity) => {
            const Icon = activityIcons[activity.type]

            return (
              <div key={activity.id} className="flex gap-3">
                <div className="flex-shrink-0">
                  <div
                    className={`w-8 h-8 rounded-full flex items-center justify-center ${activityColors[activity.type]}`}
                  >
                    <Icon className="w-4 h-4" />
                  </div>
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <p className="text-sm text-gray-900 dark:text-white">
                        <span className="font-semibold">{activity.studentName}</span>{' '}
                        <span className="text-gray-600 dark:text-gray-400">{activity.action}</span>{' '}
                        <span className="font-medium">{activity.item}</span>
                      </p>
                      <div className="flex items-center gap-2 mt-1">
                        <Badge variant="outline" className="text-xs">
                          {activity.courseName}
                        </Badge>
                        <span className="text-xs text-gray-500">
                          {formatRelativeTime(activity.timestamp)}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      </CardContent>
    </Card>
  )
}

'use client'

import * as React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Calendar, Clock, MapPin, BookOpen, FlaskConical, Users } from 'lucide-react'
import { mockUpcomingClasses } from '@/lib/mock-data'
import { formatDateTime } from '@/lib/utils'

const typeIcons = {
  lecture: BookOpen,
  lab: FlaskConical,
  'office-hours': Users,
  exam: BookOpen,
}

const typeColors = {
  lecture: 'default' as const,
  lab: 'info' as const,
  'office-hours': 'success' as const,
  exam: 'destructive' as const,
}

export function UpcomingClassesWidget() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Upcoming Classes</CardTitle>
        <CardDescription>Your schedule for the next few days</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {mockUpcomingClasses.map((classItem) => {
            const Icon = typeIcons[classItem.type]
            const now = new Date()
            const classDate = new Date(classItem.date)
            const isToday = classDate.toDateString() === now.toDateString()
            const isSoon = classDate.getTime() - now.getTime() < 3 * 60 * 60 * 1000 // Less than 3 hours

            return (
              <div
                key={classItem.id}
                className="border rounded-lg p-3 hover:shadow-md transition-shadow"
              >
                <div className="flex items-start gap-3">
                  <div className="mt-1">
                    <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
                      <Icon className="w-5 h-5 text-primary" />
                    </div>
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center justify-between mb-1">
                      <Badge variant={typeColors[classItem.type]} className="text-xs">
                        {classItem.type.replace('-', ' ')}
                      </Badge>
                      {isSoon && (
                        <Badge variant="warning" className="text-xs">
                          Starting Soon
                        </Badge>
                      )}
                    </div>
                    <h4 className="font-semibold text-sm text-gray-900 dark:text-white mb-1">
                      {classItem.courseCode}: {classItem.topic}
                    </h4>
                    <div className="space-y-1 text-xs text-gray-600 dark:text-gray-400">
                      <div className="flex items-center gap-1">
                        <Calendar className="w-3 h-3" />
                        <span>{formatDateTime(classItem.date)}</span>
                        {isToday && (
                          <Badge variant="info" className="text-xs ml-2">
                            Today
                          </Badge>
                        )}
                      </div>
                      <div className="flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        <span>
                          {classItem.startTime} - {classItem.endTime}
                        </span>
                      </div>
                      <div className="flex items-center gap-1">
                        <MapPin className="w-3 h-3" />
                        <span>{classItem.location}</span>
                      </div>
                      {classItem.prepNotes && (
                        <div className="mt-2 p-2 bg-blue-50 dark:bg-blue-950/20 rounded text-xs">
                          <span className="font-semibold">Note: </span>
                          {classItem.prepNotes}
                        </div>
                      )}
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

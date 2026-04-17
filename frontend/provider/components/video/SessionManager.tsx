'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Loader2, Video, ExternalLink, XCircle } from 'lucide-react'
import { useVideoSessions, useCancelVideoSession } from '@/lib/api/hooks'
import { formatDateTime } from '@/lib/utils'

export function SessionManager() {
  const { data: sessions, isLoading } = useVideoSessions()
  const cancelSession = useCancelVideoSession()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
      </div>
    )
  }

  const allSessions = sessions ?? []
  const upcoming = allSessions.filter((s) => s.status === 'scheduled' || s.status === 'live')
  const past = allSessions.filter((s) => s.status === 'ended' || s.status === 'cancelled')

  function isJoinable(session: (typeof allSessions)[0]): boolean {
    if (session.status === 'live') return true
    if (session.status !== 'scheduled') return false
    const diff = new Date(session.scheduledAt).getTime() - Date.now()
    return diff <= 10 * 60 * 1000 // 10 minutes before
  }

  if (allSessions.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12 text-center">
          <Video className="h-10 w-10 text-slate-300 mb-3" />
          <p className="text-slate-500">No video sessions scheduled.</p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Tabs defaultValue="upcoming">
      <TabsList>
        <TabsTrigger value="upcoming">Upcoming ({upcoming.length})</TabsTrigger>
        <TabsTrigger value="past">Past ({past.length})</TabsTrigger>
      </TabsList>

      <TabsContent value="upcoming" className="mt-4">
        {upcoming.length === 0 ? (
          <Card>
            <CardContent className="text-center py-8 text-slate-500">
              No upcoming sessions.
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-3">
            {upcoming.map((session) => (
              <Card key={session.id} className="hover:shadow-sm transition-shadow">
                <CardContent className="flex items-center justify-between p-4">
                  <div>
                    <div className="flex items-center gap-2 mb-1">
                      <p className="text-sm font-medium text-slate-900">{session.title}</p>
                      <Badge variant={session.status === 'live' ? 'destructive' : 'default'}>
                        {session.status === 'live' ? 'LIVE' : 'Scheduled'}
                      </Badge>
                    </div>
                    <div className="flex items-center gap-3 text-xs text-slate-500">
                      {session.courseName && <span>{session.courseName}</span>}
                      <span>{formatDateTime(session.scheduledAt)}</span>
                      <span>{session.duration} min</span>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {isJoinable(session) && session.joinUrl && (
                      <Button size="sm" asChild>
                        <a href={session.joinUrl} target="_blank" rel="noopener noreferrer">
                          <ExternalLink className="h-4 w-4 mr-1" />
                          Join
                        </a>
                      </Button>
                    )}
                    {session.status === 'scheduled' && (
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => cancelSession.mutate(session.id)}
                        disabled={cancelSession.isPending}
                        className="text-red-500 hover:text-red-700"
                      >
                        <XCircle className="h-4 w-4" />
                      </Button>
                    )}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </TabsContent>

      <TabsContent value="past" className="mt-4">
        {past.length === 0 ? (
          <Card>
            <CardContent className="text-center py-8 text-slate-500">No past sessions.</CardContent>
          </Card>
        ) : (
          <div className="space-y-3">
            {past.map((session) => (
              <Card key={session.id} className="hover:shadow-sm transition-shadow">
                <CardContent className="flex items-center justify-between p-4">
                  <div>
                    <div className="flex items-center gap-2 mb-1">
                      <p className="text-sm font-medium text-slate-900">{session.title}</p>
                      <Badge variant={session.status === 'cancelled' ? 'destructive' : 'secondary'}>
                        {session.status}
                      </Badge>
                    </div>
                    <div className="flex items-center gap-3 text-xs text-slate-500">
                      {session.courseName && <span>{session.courseName}</span>}
                      <span>{formatDateTime(session.scheduledAt)}</span>
                      <span>{session.duration} min</span>
                    </div>
                  </div>
                  {session.recordingUrl && (
                    <Button size="sm" variant="outline" asChild>
                      <a href={session.recordingUrl} target="_blank" rel="noopener noreferrer">
                        <Video className="h-4 w-4 mr-1" />
                        Recording
                      </a>
                    </Button>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </TabsContent>
    </Tabs>
  )
}

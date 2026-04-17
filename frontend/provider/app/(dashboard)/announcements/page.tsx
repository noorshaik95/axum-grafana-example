'use client'

import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Plus, Megaphone, Loader2, Trash2 } from 'lucide-react'
import Link from 'next/link'
import { useAnnouncements, usePublishAnnouncement, useDeleteAnnouncement } from '@/lib/api/hooks'
import { formatDateTime } from '@/lib/utils'

export default function AnnouncementsPage() {
  const { data: announcements, isLoading } = useAnnouncements()
  const publishAnnouncement = usePublishAnnouncement()
  const deleteAnnouncement = useDeleteAnnouncement()

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Announcements</h1>
          <p className="text-slate-500 mt-1">Create and manage announcements for your courses.</p>
        </div>
        <Button asChild>
          <Link href="/announcements/new">
            <Plus className="h-4 w-4 mr-2" />
            New Announcement
          </Link>
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
        </div>
      ) : !announcements || announcements.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <Megaphone className="h-10 w-10 text-slate-300 mb-3" />
            <p className="text-slate-500 mb-4">No announcements yet.</p>
            <Button asChild>
              <Link href="/announcements/new">
                <Plus className="h-4 w-4 mr-2" />
                Create Announcement
              </Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {announcements.map((announcement) => (
            <Card key={announcement.id} className="hover:shadow-sm transition-shadow">
              <CardContent className="p-4">
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="text-sm font-medium text-slate-900 truncate">
                        {announcement.title}
                      </h3>
                      <Badge
                        variant={
                          announcement.status === 'published'
                            ? 'default'
                            : announcement.status === 'scheduled'
                              ? 'secondary'
                              : 'outline'
                        }
                      >
                        {announcement.status}
                      </Badge>
                    </div>
                    <p className="text-sm text-slate-600 line-clamp-2 mb-2">
                      {announcement.content.replace(/<[^>]*>/g, '').slice(0, 200)}
                    </p>
                    <div className="flex items-center gap-3 text-xs text-slate-400">
                      {announcement.courseName && <span>{announcement.courseName}</span>}
                      <span>
                        {announcement.publishedAt
                          ? `Published ${formatDateTime(announcement.publishedAt)}`
                          : announcement.scheduledAt
                            ? `Scheduled for ${formatDateTime(announcement.scheduledAt)}`
                            : `Created ${formatDateTime(announcement.createdAt)}`}
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 ml-4 shrink-0">
                    {announcement.status === 'draft' && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => publishAnnouncement.mutate(announcement.id)}
                        disabled={publishAnnouncement.isPending}
                      >
                        Publish
                      </Button>
                    )}
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => deleteAnnouncement.mutate(announcement.id)}
                      disabled={deleteAnnouncement.isPending}
                    >
                      <Trash2 className="h-4 w-4 text-red-500" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}

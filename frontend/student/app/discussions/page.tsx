import Link from 'next/link';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { MessageSquare, Pin, Lock, Eye, MessageCircle, Search, Plus } from 'lucide-react';
import { mockDiscussions } from '@/lib/mock-data';
import { formatRelativeTime } from '@/lib/utils';

export const metadata = {
  title: 'Discussions | Student Portal',
  description: 'Participate in course discussions',
};

export default function DiscussionsPage() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Discussions</h1>
          <p className="text-muted-foreground">Engage with your classmates and instructors</p>
        </div>
        <Button>
          <Plus className="mr-2 h-4 w-4" />
          New Discussion
        </Button>
      </div>

      {/* Search */}
      <Card>
        <CardContent className="pt-6">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input placeholder="Search discussions..." className="pl-10" />
          </div>
        </CardContent>
      </Card>

      {/* Discussion List */}
      <div className="space-y-4">
        {mockDiscussions.map((discussion) => (
          <Link key={discussion.id} href={`/discussions/${discussion.id}`}>
            <Card className="transition-all hover:shadow-md">
              <CardHeader>
                <div className="flex items-start gap-4">
                  {/* Avatar */}
                  <img
                    src={discussion.authorAvatar}
                    alt={discussion.authorName}
                    className="h-10 w-10 rounded-full"
                  />

                  {/* Content */}
                  <div className="flex-1 space-y-2">
                    <div className="flex items-start justify-between">
                      <div className="space-y-1">
                        <div className="flex items-center gap-2">
                          {discussion.isPinned && <Pin className="h-4 w-4 text-primary" />}
                          {discussion.isLocked && (
                            <Lock className="h-4 w-4 text-muted-foreground" />
                          )}
                          <Badge variant="secondary">{discussion.courseName}</Badge>
                        </div>
                        <CardTitle className="hover:text-primary">{discussion.title}</CardTitle>
                        <p className="text-sm text-muted-foreground">
                          by {discussion.authorName} • {formatRelativeTime(discussion.createdAt)}
                        </p>
                      </div>
                    </div>

                    <CardDescription className="line-clamp-2">{discussion.content}</CardDescription>

                    {/* Stats */}
                    <div className="flex items-center gap-4 text-sm text-muted-foreground">
                      <div className="flex items-center gap-1">
                        <Eye className="h-4 w-4" />
                        <span>{discussion.views} views</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <MessageCircle className="h-4 w-4" />
                        <span>{discussion.replyCount} replies</span>
                      </div>
                      <span>Last updated {formatRelativeTime(discussion.updatedAt)}</span>
                    </div>
                  </div>
                </div>
              </CardHeader>
            </Card>
          </Link>
        ))}
      </div>

      {/* Empty State */}
      {mockDiscussions.length === 0 && (
        <Card className="p-12">
          <div className="flex flex-col items-center text-center space-y-4">
            <MessageSquare className="h-16 w-16 text-muted-foreground/50" />
            <div className="space-y-2">
              <h3 className="text-xl font-semibold">No Discussions</h3>
              <p className="text-muted-foreground">
                Be the first to start a discussion in your courses.
              </p>
            </div>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Start a Discussion
            </Button>
          </div>
        </Card>
      )}
    </div>
  );
}

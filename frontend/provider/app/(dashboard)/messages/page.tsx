'use client'

import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Mail, Loader2, Plus } from 'lucide-react'
import Link from 'next/link'
import { useInbox, useSentThreads } from '@/lib/api/hooks'
import { formatRelativeTime, getInitials } from '@/lib/utils'
import { ComposeDialog } from '@/components/messages/ComposeDialog'
import { useState } from 'react'

export default function MessagesPage() {
  const { data: inbox, isLoading: inboxLoading } = useInbox()
  const { data: sent, isLoading: sentLoading } = useSentThreads()
  const [showCompose, setShowCompose] = useState(false)

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Messages</h1>
          <p className="text-slate-500 mt-1">Communicate with students and colleagues.</p>
        </div>
        <Button onClick={() => setShowCompose(true)}>
          <Plus className="h-4 w-4 mr-2" />
          New Message
        </Button>
      </div>

      <Tabs defaultValue="inbox">
        <TabsList>
          <TabsTrigger value="inbox">
            Inbox {inbox && inbox.length > 0 && `(${inbox.length})`}
          </TabsTrigger>
          <TabsTrigger value="sent">Sent</TabsTrigger>
        </TabsList>

        <TabsContent value="inbox" className="mt-4">
          {inboxLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
            </div>
          ) : !inbox || inbox.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12 text-center">
                <Mail className="h-10 w-10 text-slate-300 mb-3" />
                <p className="text-slate-500">No messages in your inbox.</p>
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-2">
              {inbox.map((thread) => (
                <Link key={thread.id} href={`/messages/${thread.id}`}>
                  <Card className="hover:shadow-sm transition-shadow cursor-pointer">
                    <CardContent className="flex items-center gap-4 p-4">
                      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-indigo-100 text-sm font-medium text-indigo-700">
                        {thread.participants[0] ? getInitials(thread.participants[0].name) : '?'}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between">
                          <p
                            className={`text-sm truncate ${thread.unreadCount > 0 ? 'font-semibold text-slate-900' : 'text-slate-700'}`}
                          >
                            {thread.subject}
                          </p>
                          <span className="text-xs text-slate-400 shrink-0 ml-2">
                            {formatRelativeTime(thread.updatedAt)}
                          </span>
                        </div>
                        <p className="text-sm text-slate-500 truncate mt-0.5">
                          {thread.lastMessage?.content ?? 'No messages'}
                        </p>
                      </div>
                      {thread.unreadCount > 0 && (
                        <Badge className="shrink-0">{thread.unreadCount}</Badge>
                      )}
                    </CardContent>
                  </Card>
                </Link>
              ))}
            </div>
          )}
        </TabsContent>

        <TabsContent value="sent" className="mt-4">
          {sentLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
            </div>
          ) : !sent || sent.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12 text-center">
                <Mail className="h-10 w-10 text-slate-300 mb-3" />
                <p className="text-slate-500">No sent messages.</p>
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-2">
              {sent.map((thread) => (
                <Link key={thread.id} href={`/messages/${thread.id}`}>
                  <Card className="hover:shadow-sm transition-shadow cursor-pointer">
                    <CardContent className="flex items-center gap-4 p-4">
                      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-slate-100 text-sm font-medium text-slate-600">
                        {thread.participants[0] ? getInitials(thread.participants[0].name) : '?'}
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm text-slate-700 truncate">{thread.subject}</p>
                        <p className="text-sm text-slate-500 truncate mt-0.5">
                          {thread.lastMessage?.content ?? 'No messages'}
                        </p>
                      </div>
                      <span className="text-xs text-slate-400 shrink-0">
                        {formatRelativeTime(thread.updatedAt)}
                      </span>
                    </CardContent>
                  </Card>
                </Link>
              ))}
            </div>
          )}
        </TabsContent>
      </Tabs>

      <ComposeDialog open={showCompose} onOpenChange={setShowCompose} />
    </div>
  )
}

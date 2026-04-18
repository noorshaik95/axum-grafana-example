'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { ArrowLeft, Loader2, Send } from 'lucide-react'
import Link from 'next/link'
import { useThread, useSendMessage } from '@/lib/api/hooks'
import { formatDateTime, getInitials } from '@/lib/utils'

export default function ThreadPage() {
  const params = useParams()
  const threadId = params.threadId as string
  const { data, isLoading } = useThread(threadId)
  const sendMessage = useSendMessage()
  const [reply, setReply] = useState('')

  async function handleSend(e: React.FormEvent) {
    e.preventDefault()
    if (!reply.trim()) return
    await sendMessage.mutateAsync({
      threadId,
      data: { content: reply.trim() },
    })
    setReply('')
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    )
  }

  const thread = data?.thread
  const messages = data?.messages ?? []

  return (
    <div className="space-y-6 max-w-3xl mx-auto">
      <div>
        <Button variant="ghost" size="sm" asChild className="mb-2 -ml-2">
          <Link href="/messages">
            <ArrowLeft className="h-4 w-4 mr-1" />
            Back to Messages
          </Link>
        </Button>
        <h1 className="text-xl font-bold text-slate-900">{thread?.subject ?? 'Thread'}</h1>
        {thread?.participants && (
          <p className="text-sm text-slate-500 mt-1">
            {thread.participants.map((p) => p.name).join(', ')}
          </p>
        )}
      </div>

      {/* Messages */}
      <div className="space-y-4">
        {messages.map((msg) => (
          <Card key={msg.id}>
            <CardContent className="p-4">
              <div className="flex items-start gap-3">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-indigo-100 text-sm font-medium text-indigo-700">
                  {msg.senderName ? getInitials(msg.senderName) : '?'}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <p className="text-sm font-medium text-slate-900">
                      {msg.senderName ?? 'Unknown'}
                    </p>
                    <span className="text-xs text-slate-400">{formatDateTime(msg.createdAt)}</span>
                  </div>
                  <p className="text-sm text-slate-600 mt-1 whitespace-pre-wrap">{msg.content}</p>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Reply */}
      <Card>
        <CardContent className="p-4">
          <form onSubmit={handleSend} className="space-y-3">
            <Textarea
              value={reply}
              onChange={(e) => setReply(e.target.value)}
              placeholder="Type your reply..."
              rows={3}
            />
            <div className="flex justify-end">
              <Button type="submit" disabled={sendMessage.isPending || !reply.trim()}>
                {sendMessage.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Send className="h-4 w-4 mr-2" />
                )}
                Send
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

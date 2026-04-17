'use client'

import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Loader2 } from 'lucide-react'
import { useCreateThread } from '@/lib/api/hooks'

interface ComposeDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ComposeDialog({ open, onOpenChange }: ComposeDialogProps) {
  const createThread = useCreateThread()
  const [subject, setSubject] = useState('')
  const [recipients, setRecipients] = useState('')
  const [content, setContent] = useState('')

  async function handleSend(e: React.FormEvent) {
    e.preventDefault()
    if (!subject.trim() || !recipients.trim() || !content.trim()) return
    const recipientIds = recipients
      .split(',')
      .map((r) => r.trim())
      .filter(Boolean)
    await createThread.mutateAsync({
      subject: subject.trim(),
      recipientIds,
      content: content.trim(),
    })
    setSubject('')
    setRecipients('')
    setContent('')
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>New Message</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSend} className="space-y-4 mt-4">
          <div>
            <Label htmlFor="to">To (user IDs, comma-separated)</Label>
            <Input
              id="to"
              value={recipients}
              onChange={(e) => setRecipients(e.target.value)}
              placeholder="user-id-1, user-id-2"
              className="mt-1.5"
            />
          </div>
          <div>
            <Label htmlFor="subject">Subject</Label>
            <Input
              id="subject"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="Message subject"
              className="mt-1.5"
            />
          </div>
          <div>
            <Label htmlFor="body">Message</Label>
            <Textarea
              id="body"
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder="Write your message..."
              rows={5}
              className="mt-1.5"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={createThread.isPending || !subject.trim() || !content.trim()}
            >
              {createThread.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
              Send
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

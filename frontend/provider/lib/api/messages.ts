import { apiClient } from '../../../shared/lib/api/client'

export interface Message {
  id: string
  threadId: string
  senderId: string
  senderName?: string
  content: string
  readAt: string | null
  createdAt: string
}

export interface Thread {
  id: string
  subject: string
  participants: ThreadParticipant[]
  lastMessage?: Message
  unreadCount: number
  createdAt: string
  updatedAt: string
}

export interface ThreadParticipant {
  userId: string
  name: string
  avatarUrl?: string
}

export interface CreateThreadDto {
  subject: string
  recipientIds: string[]
  content: string
}

export interface SendMessageDto {
  content: string
}

export async function getInbox(): Promise<Thread[]> {
  return apiClient.get<Thread[]>('/api/messages/inbox')
}

export async function getSentThreads(): Promise<Thread[]> {
  // TODO: no /api/messages/sent gateway route in email-service
  return []
}

export async function getThread(
  threadId: string
): Promise<{ thread: Thread; messages: Message[] }> {
  return apiClient.get<{ thread: Thread; messages: Message[] }>(`/api/messages/threads/${threadId}`)
}

export async function createThread(data: CreateThreadDto): Promise<Thread> {
  return apiClient.post<Thread>('/api/messages', {
    subject: data.subject,
    recipient_ids: data.recipientIds,
    content: data.content,
  }) as Promise<Thread>
}

export async function sendMessage(threadId: string, data: SendMessageDto): Promise<Message> {
  return apiClient.post<Message>('/api/messages', {
    thread_id: threadId,
    content: data.content,
  }) as Promise<Message>
}

export async function markThreadRead(threadId: string): Promise<void> {
  return apiClient.post(`/api/messages/${threadId}/read`)
}

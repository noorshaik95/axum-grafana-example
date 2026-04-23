import { get, post, getCurrentUserId } from './client';

export interface MessageThread {
  id: string;
  subject: string;
  participants: ThreadParticipant[];
  lastMessage: string;
  lastMessageAt: string;
  unreadCount: number;
  courseId: string | null;
  courseTitle: string | null;
}

export interface ThreadParticipant {
  id: string;
  name: string;
  role: string;
  avatarUrl: string | null;
}

export interface Message {
  id: string;
  threadId: string;
  senderId: string;
  senderName: string;
  senderRole: string;
  content: string;
  createdAt: string;
  readAt: string | null;
}

// email.MessagingService.GetInbox requires tenant_id + user_id per proto
// (services/email-service/api/proto/email.proto GetInboxRequest).
export function getInbox(): Promise<MessageThread[]> {
  const userId = getCurrentUserId();
  if (!userId) return Promise.resolve([]);
  const p = new URLSearchParams();
  p.set('user_id', userId);
  return get<MessageThread[]>(`/api/messages/inbox?${p.toString()}`);
}

export function getSent(): Promise<MessageThread[]> {
  const userId = getCurrentUserId();
  if (!userId) return Promise.resolve([]);
  const p = new URLSearchParams();
  p.set('user_id', userId);
  return get<MessageThread[]>(`/api/messages/sent?${p.toString()}`);
}

export function getThread(threadId: string): Promise<{
  thread: MessageThread;
  messages: Message[];
}> {
  return get(`/api/messages/threads/${threadId}`);
}

export function sendMessage(data: {
  threadId?: string;
  recipientIds: string[];
  subject: string;
  content: string;
  courseId?: string;
}): Promise<Message> {
  return post<Message>('/api/messages', data);
}

export function replyToThread(threadId: string, content: string): Promise<Message> {
  return post<Message>(`/api/messages/threads/${threadId}/reply`, { content });
}

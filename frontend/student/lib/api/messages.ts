import { get, post } from './client';

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

export function getInbox(): Promise<MessageThread[]> {
  return get<MessageThread[]>('/api/messages/inbox');
}

export function getSent(): Promise<MessageThread[]> {
  return get<MessageThread[]>('/api/messages/sent');
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

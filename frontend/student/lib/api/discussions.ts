import { get, getCurrentUserId } from './client';

export type InboxAuthorRole = 'prof' | 'peer' | 'system' | 'ta';
export type InboxFilter = 'mentions' | 'replies' | 'all' | 'profs' | 'peers';

export interface InboxItem {
  id: string;
  threadId: string;
  authorId: string;
  authorName: string;
  authorRole: InboxAuthorRole;
  authorInitials: string;
  courseId?: string;
  courseCode?: string;
  snippet: string;
  kind: 'mention' | 'reply' | 'announcement';
  createdAt: string;
  read: boolean;
}

// Proto discussion.DiscussionService.GetInbox requires tenant_id + user_id
// (services/discussion-service/internal/grpc/server.go GetInbox handler).
// `filter` is a client-side grouping today; the proto has no such field so
// we pass it as an extra query param for future-compat and filter locally.
// Empty responses — e.g. when FE can't derive user_id — yield [].
export function getInbox(filter?: InboxFilter): Promise<InboxItem[]> {
  const userId = getCurrentUserId();
  if (!userId) return Promise.resolve([]);
  const p = new URLSearchParams();
  p.set('user_id', userId);
  // unseen_only aligns with proto GetInboxRequest.unseen_only
  if (filter === 'mentions') p.set('unseen_only', 'true');
  return get<InboxItem[]>(`/api/discussions/inbox?${p.toString()}`);
}

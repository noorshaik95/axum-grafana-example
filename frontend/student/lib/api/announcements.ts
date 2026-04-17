import { get } from './client';

export interface Announcement {
  id: string;
  courseId: string | null;
  courseTitle: string | null;
  title: string;
  content: string;
  authorName: string;
  priority: 'low' | 'normal' | 'high' | 'urgent';
  createdAt: string;
  readAt: string | null;
}

export function getAnnouncements(params?: {
  courseId?: string;
  unreadOnly?: boolean;
}): Promise<Announcement[]> {
  const searchParams = new URLSearchParams();
  if (params?.courseId) searchParams.set('courseId', params.courseId);
  if (params?.unreadOnly) searchParams.set('unreadOnly', 'true');
  const query = searchParams.toString();
  return get<Announcement[]>(`/api/announcements${query ? `?${query}` : ''}`);
}

export function markAnnouncementRead(id: string): Promise<void> {
  return get<void>(`/api/announcements/${id}/read`);
}

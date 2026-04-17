import { get } from './client';

export interface VideoSession {
  id: string;
  courseId: string;
  courseTitle: string;
  title: string;
  description: string | null;
  hostName: string;
  startTime: string;
  endTime: string;
  duration: number;
  isActive: boolean;
  isRecording: boolean;
  participantCount: number;
  maxParticipants: number;
  joinUrl: string | null;
  recordingUrl: string | null;
}

export function getSessions(params?: {
  upcoming?: boolean;
  courseId?: string;
}): Promise<VideoSession[]> {
  const searchParams = new URLSearchParams();
  if (params?.upcoming) searchParams.set('upcoming', 'true');
  if (params?.courseId) searchParams.set('courseId', params.courseId);
  const query = searchParams.toString();
  return get<VideoSession[]>(`/api/video/sessions${query ? `?${query}` : ''}`);
}

export function getSession(sessionId: string): Promise<VideoSession> {
  return get<VideoSession>(`/api/video/sessions/${sessionId}`);
}

export function getJoinUrl(sessionId: string): Promise<{ url: string }> {
  return get<{ url: string }>(`/api/video/sessions/${sessionId}/join`);
}

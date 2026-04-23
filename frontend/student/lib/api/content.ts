import { get, put } from './client';

export interface ContentItem {
  id: string;
  lessonId: string;
  moduleId: string;
  courseId: string;
  name: string;
  description: string | null;
  contentType: string;
  storageKey: string;
  manifestUrl: string | null;
  posterUrl: string | null;
  durationSeconds: number | null;
  resumePositionSeconds: number;
  published: boolean;
  tracks?: Array<{
    src: string;
    kind?: string;
    srcLang?: string;
    label?: string;
    default?: boolean;
  }>;
}

export function getContent(contentId: string): Promise<ContentItem> {
  return get<ContentItem>(`/api/content/${contentId}`);
}

export function putPosition(contentId: string, positionSeconds: number): Promise<void> {
  return put<void>(`/api/content/${contentId}/position`, { positionSeconds });
}

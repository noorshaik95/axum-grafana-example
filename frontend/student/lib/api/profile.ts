'use client';

import { useQuery } from '@tanstack/react-query';
import type { User } from '../../../shared/lib/api/types';
import { getCurrentUserId } from './client';

/**
 * Student profile hook — reads the user object from localStorage
 * (written at login by /app/(auth)/login/page.tsx). Avoids a network
 * call to /api/users/profile on every layout mount, which was 404-ing
 * in production and breaking every student page via the breaker
 * (T3-R1 student parity with #43 admin fix).
 *
 * Happy path: zero network. Missing-cache fallback synthesises a
 * minimal User from the JWT's user_id so initials/avatar-menu still
 * render; the middleware auth guard already kicks unauthed users to
 * /login before this hook fires, so returning null is safe.
 */
export type StudentProfile = User;

function readCachedUser(): StudentProfile | null {
  if (typeof window === 'undefined') return null;
  const raw = window.localStorage.getItem('student_user');
  if (!raw) return null;
  try {
    return JSON.parse(raw) as StudentProfile;
  } catch {
    return null;
  }
}

function synthesiseFromJwt(): StudentProfile | null {
  const userId = getCurrentUserId();
  if (!userId) return null;
  const now = new Date().toISOString();
  return {
    id: userId,
    email: '',
    firstName: '',
    lastName: '',
    phone: null,
    isActive: true,
    authMethod: 'normal',
    timezone: 'UTC',
    avatarUrl: null,
    bio: null,
    organizationId: null,
    roles: [
      {
        id: '',
        name: 'student',
        description: null,
        permissions: [],
        createdAt: now,
        updatedAt: now,
      },
    ],
    createdAt: now,
    updatedAt: now,
  };
}

export function useStudentProfile() {
  return useQuery<StudentProfile | null>({
    queryKey: ['student', 'profile'],
    queryFn: () => {
      const cached = readCachedUser();
      if (cached) return cached;
      return synthesiseFromJwt();
    },
    staleTime: Infinity,
    gcTime: Infinity,
    retry: false,
  });
}

export function clearStudentProfile() {
  if (typeof window === 'undefined') return;
  window.localStorage.removeItem('student_user');
}

import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatDate(date: Date | string): string {
  return new Date(date).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

export function formatDateTime(date: Date | string): string {
  return new Date(date).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function formatRelativeTime(date: Date | string): string {
  const now = new Date();
  const then = new Date(date);
  const diffInSeconds = Math.floor((now.getTime() - then.getTime()) / 1000);

  if (diffInSeconds < 60) {
    return 'just now';
  } else if (diffInSeconds < 3600) {
    const minutes = Math.floor(diffInSeconds / 60);
    return `${minutes} minute${minutes > 1 ? 's' : ''} ago`;
  } else if (diffInSeconds < 86400) {
    const hours = Math.floor(diffInSeconds / 3600);
    return `${hours} hour${hours > 1 ? 's' : ''} ago`;
  } else if (diffInSeconds < 604800) {
    const days = Math.floor(diffInSeconds / 86400);
    return `${days} day${days > 1 ? 's' : ''} ago`;
  } else {
    return formatDate(date);
  }
}

export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 Bytes';

  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

export function truncate(str: string, length: number): string {
  if (str.length <= length) return str;
  return str.slice(0, length) + '...';
}

export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout | null = null;

  return function executedFunction(...args: Parameters<T>) {
    const later = () => {
      timeout = null;
      func(...args);
    };

    if (timeout) {
      clearTimeout(timeout);
    }
    timeout = setTimeout(later, wait);
  };
}

export function throttle<T extends (...args: any[]) => any>(
  func: T,
  limit: number
): (...args: Parameters<T>) => void {
  let inThrottle: boolean;

  return function executedFunction(...args: Parameters<T>) {
    if (!inThrottle) {
      func(...args);
      inThrottle = true;
      setTimeout(() => (inThrottle = false), limit);
    }
  };
}

export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export function isServer(): boolean {
  return typeof window === 'undefined';
}

export function getInitials(name: string): string {
  return name
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);
}

export function calculatePercentage(value: number, total: number): number {
  if (total === 0) return 0;
  return Math.round((value / total) * 100);
}

// Semantic grade colors: blue for excellent, yellow for warning, red for failing
export function getGradeColor(grade: number): string {
  if (grade >= 80) return 'text-blue-500'; // A/B - Excellent
  if (grade >= 60) return 'text-yellow-500'; // C/D - Warning
  return 'text-red-500'; // F - Failing
}

export function getGradeLetter(grade: number): string {
  if (grade >= 90) return 'A';
  if (grade >= 80) return 'B';
  if (grade >= 70) return 'C';
  if (grade >= 60) return 'D';
  return 'F';
}

// Semantic badge class for grades
export function getGradeBadgeClass(grade: number): string {
  if (grade >= 80) return 'bg-blue-500 text-white border-0'; // A/B - Excellent
  if (grade >= 60) return 'bg-yellow-500 text-white border-0'; // C/D - Warning
  return 'bg-red-500 text-white border-0'; // F - Failing
}

// Semantic colors for status indication
export function getStatusBadgeClass(status: 'success' | 'warning' | 'error' | 'info'): string {
  switch (status) {
    case 'success':
      return 'bg-blue-500 text-white border-0';
    case 'warning':
      return 'bg-yellow-500 text-white border-0';
    case 'error':
      return 'bg-red-500 text-white border-0';
    case 'info':
    default:
      return 'bg-slate-500 text-white border-0';
  }
}

// Get due date status based on days remaining
export function getDueDateStatus(dueDate: string): 'error' | 'warning' | 'success' {
  const days = Math.floor((new Date(dueDate).getTime() - Date.now()) / (24 * 60 * 60 * 1000));
  if (days < 0) return 'error'; // Overdue
  if (days <= 1) return 'error'; // Due today or tomorrow
  if (days <= 3) return 'warning'; // Due soon
  return 'success'; // Not urgent
}

// Get due date color class
export function getDueDateColor(dueDate: string): string {
  const status = getDueDateStatus(dueDate);
  switch (status) {
    case 'error':
      return 'text-red-500 font-semibold';
    case 'warning':
      return 'text-yellow-500 font-medium';
    default:
      return 'text-muted-foreground';
  }
}

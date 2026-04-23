export interface RouteEntry {
  path: string;
  label: string;
  group?: string;
}

export const STUDENT_ROUTE_CATALOG: RouteEntry[] = [
  { path: '/today', label: 'Today', group: 'Pages' },
  { path: '/courses', label: 'Courses', group: 'Pages' },
  { path: '/assignments', label: 'Assignments', group: 'Pages' },
  { path: '/grades', label: 'Grades', group: 'Pages' },
  { path: '/plan', label: '16-week plan', group: 'Pages' },
  { path: '/inbox', label: 'Inbox', group: 'Pages' },
  { path: '/people', label: 'People', group: 'Pages' },
  { path: '/office-hours', label: 'Office hours', group: 'Pages' },
  { path: '/profile', label: 'Profile', group: 'Pages' },
  { path: '/settings', label: 'Settings', group: 'Pages' },
];

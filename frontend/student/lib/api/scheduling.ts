import { get, post } from './client';

export interface InstructorSummary {
  id: string;
  name: string;
  initials: string;
  courseCode?: string;
}

export interface OfficeHoursSlot {
  id: string;
  instructorId: string;
  instructor: InstructorSummary;
  startsAt: string;
  endsAt: string;
  available: boolean;
  location?: string;
  modality: 'online' | 'in_person';
}

export interface BookingRequest {
  slotId: string;
  question?: string;
  assignmentContextId?: string;
}

export interface Booking {
  id: string;
  slotId: string;
  studentId: string;
  question: string | null;
  status: 'booked' | 'cancelled' | 'completed';
  createdAt: string;
}

// Aligned with scheduling.SchedulingService.ListAvailableSlots proto:
// required instructor_id (string), date_from + date_to ("YYYY-MM-DD").
// See services/scheduling-service/api/proto/scheduling.proto.
export function getSlots(params?: {
  instructorId?: string;
  courseId?: string;
  dateFrom?: string;
  dateTo?: string;
}): Promise<OfficeHoursSlot[]> {
  const p = new URLSearchParams();
  if (params?.instructorId) p.set('instructor_id', params.instructorId);
  if (params?.courseId) p.set('course_id', params.courseId);
  if (params?.dateFrom) p.set('date_from', params.dateFrom);
  if (params?.dateTo) p.set('date_to', params.dateTo);
  const q = p.toString();
  return get<OfficeHoursSlot[]>(`/api/scheduling/slots${q ? `?${q}` : ''}`);
}

export function createBooking(body: BookingRequest): Promise<Booking> {
  return post<Booking>('/api/scheduling/bookings', body);
}

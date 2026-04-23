import { apiClient } from '../../../shared/lib/api/client'

// W6 scheduling-service contract (services/scheduling-service/api/proto/scheduling.proto).

export type SlotMode = 'online' | 'in_person'
export type SlotStatus = 'available' | 'booked' | 'cancelled'

export interface ScheduleRecurrence {
  days_of_week: number[]
  start_time: string // "HH:MM"
  end_time: string // "HH:MM"
  slot_minutes: number
  location?: string
  mode: SlotMode
  start_date: string // ISO
  end_date?: string | null
}

export interface Schedule {
  id: string
  instructor_id: string
  course_id?: string
  title?: string
  recurrence: ScheduleRecurrence
  created_at: string
  updated_at: string
}

export interface CreateScheduleDto {
  instructor_id: string
  course_id?: string
  title?: string
  recurrence: ScheduleRecurrence
}

export interface Booking {
  id: string
  slot_id: string
  student_id: string
  student_name?: string
  student_avatar?: string | null
  context?: string
  pre_read_done?: boolean
  risk_level?: 'healthy' | 'slipping' | 'at_risk'
  status: SlotStatus
  created_at: string
}

export interface InstructorSlot {
  id: string
  schedule_id: string
  start_time: string
  end_time: string
  mode: SlotMode
  location?: string
  status: SlotStatus
  booking?: Booking | null
}

export interface InstructorDayResponse {
  instructor_id: string
  date: string // YYYY-MM-DD
  slots: InstructorSlot[]
  booked_count: number
  open_count: number
}

export async function getInstructorDay(date?: string): Promise<InstructorDayResponse> {
  const q = new URLSearchParams()
  q.set('date', date ?? 'today')
  return apiClient.get<InstructorDayResponse>(`/api/scheduling/instructor-day?${q.toString()}`)
}

export async function listSchedules(instructorId?: string): Promise<Schedule[]> {
  const q = new URLSearchParams()
  if (instructorId) q.set('instructor_id', instructorId)
  const suffix = q.toString() ? `?${q.toString()}` : ''
  return apiClient.get<Schedule[]>(`/api/scheduling/schedules${suffix}`)
}

export async function createSchedule(data: CreateScheduleDto): Promise<Schedule> {
  return apiClient.post<Schedule>('/api/scheduling/schedules', data)
}

export type GradientType =
  | 'blue-cyan'
  | 'purple-pink'
  | 'emerald-teal'
  | 'orange-red'
  | 'amber-yellow'
  | 'violet-indigo'
  | 'indigo-purple'

export type Priority = 'low' | 'medium' | 'high'
export type AssignmentType = 'essay' | 'quiz' | 'project' | 'assignment'
export type ClassType = 'lecture' | 'lab' | 'office-hours' | 'exam'
export type ActivityType = 'submission' | 'question' | 'completion' | 'discussion'
export type RiskLevel = 'low' | 'medium' | 'high'
export type QuestionStatus = 'pending' | 'answered'

export interface Course {
  id: string
  code: string
  name: string
  section: string
  description: string
  semester: string
  year: number
  credits: number
  studentCount: number
  progress: number
  averageGrade: number
  pendingGrading: number
  activeStudents: number
  gradient: GradientType
  schedule: Array<{
    day: string
    time: string
    location: string
  }>
}

export interface GradingQueueItem {
  id: string
  studentId: string
  studentName: string
  studentAvatar: string
  courseId: string
  courseName: string
  assignmentId: string
  assignmentName: string
  assignmentType: AssignmentType
  submittedAt: string
  dueDate: string
  points: number
  priority: Priority
}

export interface UpcomingClass {
  id: string
  courseId: string
  courseName: string
  courseCode: string
  topic: string
  date: string
  startTime: string
  endTime: string
  location: string
  type: ClassType
  prepNotes?: string
}

export interface RecentActivityItem {
  id: string
  type: ActivityType
  studentName: string
  studentAvatar: string
  courseName: string
  action: string
  item: string
  timestamp: string
}

export interface StudentPerformance {
  overall: {
    averageGrade: number
    totalStudents: number
    activeStudents: number
    atRiskStudents: number
  }
  gradeDistribution: Array<{
    grade: string
    count: number
    percentage: number
  }>
  engagementTrend: Array<{
    week: string
    score: number
  }>
  atRiskStudents: Array<{
    id: string
    name: string
    avatar: string
    courseId: string
    courseName: string
    currentGrade: number
    missedAssignments: number
    lastActive: string
    riskLevel: RiskLevel
  }>
}

export interface Announcement {
  id: string
  courseId: string
  courseName: string
  title: string
  message: string
  priority: Priority
  createdAt: string
  isPinned: boolean
}

export interface StudentQuestion {
  id: string
  studentId: string
  studentName: string
  studentAvatar: string
  courseId: string
  courseName: string
  question: string
  topic: string
  createdAt: string
  status: QuestionStatus
  isUrgent: boolean
}

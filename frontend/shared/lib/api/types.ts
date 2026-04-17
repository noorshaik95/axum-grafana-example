// =============================================================================
// Slate LMS — API Type Definitions
// =============================================================================
// These interfaces reflect the actual database schemas across all services.
// Date fields are strings (ISO 8601) as they arrive from JSON responses.
// ID fields are strings (UUIDs for PostgreSQL, ObjectId hex for MongoDB).
// =============================================================================

// ---------------------------------------------------------------------------
// API Wrappers
// ---------------------------------------------------------------------------

export interface ApiResponse<T> {
  readonly success: boolean
  readonly data: T
  readonly message?: string
  readonly timestamp: string
}

export interface ApiError {
  readonly success: false
  readonly error: {
    readonly code: string
    readonly message: string
    readonly details?: Record<string, unknown>
  }
  readonly timestamp: string
}

export interface PaginatedResponse<T> {
  readonly success: boolean
  readonly data: readonly T[]
  readonly pagination: {
    readonly page: number
    readonly pageSize: number
    readonly totalItems: number
    readonly totalPages: number
  }
  readonly timestamp: string
}

// ---------------------------------------------------------------------------
// Auth / Users (userauth DB)
// ---------------------------------------------------------------------------

export type AuthMethod = 'normal' | 'oauth' | 'saml'

export type RoleName = 'student' | 'instructor' | 'admin' | 'superadmin' | 'user' | 'manager'

export interface Role {
  readonly id: string
  readonly name: RoleName
  readonly description: string | null
  readonly permissions: readonly string[]
  readonly createdAt: string
  readonly updatedAt: string
}

export interface User {
  readonly id: string
  readonly email: string
  readonly firstName: string
  readonly lastName: string
  readonly phone?: string | null
  readonly isActive: boolean
  readonly authMethod: AuthMethod
  readonly timezone: string
  readonly avatarUrl?: string | null
  readonly bio?: string | null
  readonly organizationId?: string | null
  readonly roles: readonly Role[]
  readonly createdAt: string
  readonly updatedAt: string
}

export interface AuthTokens {
  readonly accessToken: string
  readonly refreshToken: string
  readonly expiresIn: number
  readonly tokenType: 'Bearer'
}

export interface LoginRequest {
  readonly email: string
  readonly password: string
}

export interface RegisterRequest {
  readonly email: string
  readonly password: string
  readonly firstName: string
  readonly lastName: string
  readonly phone?: string
  readonly timezone?: string
}

export interface UpdateProfileRequest {
  readonly firstName?: string
  readonly lastName?: string
  readonly phone?: string | null
  readonly timezone?: string
  readonly avatarUrl?: string | null
  readonly bio?: string | null
}

export interface OAuthProvider {
  readonly id: string
  readonly provider: string
  readonly providerType: 'google' | 'microsoft' | 'custom'
  readonly providerUserId: string
  readonly createdAt: string
  readonly updatedAt: string
}

export interface UserGroup {
  readonly id: string
  readonly name: string
  readonly description: string | null
  readonly organizationId: string | null
  readonly isActive: boolean
  readonly createdBy: string | null
  readonly createdAt: string
  readonly updatedAt: string
}

export interface GroupMember {
  readonly groupId: string
  readonly userId: string
  readonly role: string | null
  readonly joinedAt: string
}

// ---------------------------------------------------------------------------
// Courses (MongoDB — course-service)
// ---------------------------------------------------------------------------

export interface CourseMetadata {
  readonly maxStudents?: number
  readonly department?: string
  readonly courseCode?: string
  readonly credits?: number
  readonly tags?: readonly string[]
}

export interface Course {
  readonly id: string
  readonly title: string
  readonly description: string
  readonly term: string
  readonly syllabus?: string | null
  readonly instructorId: string
  readonly coInstructorIds: readonly string[]
  readonly isPublished: boolean
  readonly prerequisiteCourseIds: readonly string[]
  readonly templateId?: string | null
  readonly crossListingGroupId?: string | null
  readonly metadata?: CourseMetadata
  readonly createdAt: string
  readonly updatedAt: string
}

export interface Schedule {
  readonly daysOfWeek: readonly string[]
  readonly startTime: string
  readonly endTime: string
}

export interface Section {
  readonly id: string
  readonly courseId: string
  readonly sectionNumber: string
  readonly instructorId: string
  readonly schedule: Schedule
  readonly location?: string | null
  readonly maxStudents: number
  readonly enrolledCount: number
  readonly createdAt: string
  readonly updatedAt: string
}

export type EnrollmentType = 'SELF' | 'INSTRUCTOR' | 'ADMIN'
export type EnrollmentStatus = 'ACTIVE' | 'DROPPED' | 'COMPLETED' | 'WAITLISTED'

export interface Enrollment {
  readonly id: string
  readonly courseId: string
  readonly studentId: string
  readonly enrollmentType: EnrollmentType
  readonly status: EnrollmentStatus
  readonly enrolledBy: string
  readonly enrolledAt: string
  readonly sectionId?: string | null
  readonly createdAt: string
  readonly updatedAt: string
}

export interface CourseTemplate {
  readonly id: string
  readonly name: string
  readonly description: string
  readonly syllabusTemplate?: string | null
  readonly createdBy: string
  readonly defaultMetadata?: CourseMetadata
  readonly createdAt: string
  readonly updatedAt: string
}

export interface CrossListing {
  readonly id: string
  readonly groupId: string
  readonly courseIds: readonly string[]
  readonly createdBy: string
  readonly createdAt: string
  readonly updatedAt: string
}

export interface CourseFilters {
  readonly term?: string
  readonly instructorId?: string
  readonly isPublished?: boolean
  readonly department?: string
  readonly search?: string
  readonly page?: number
  readonly pageSize?: number
}

// ---------------------------------------------------------------------------
// Assignments & Grades (assignment_grading DB)
// ---------------------------------------------------------------------------

export interface Assignment {
  readonly id: string
  readonly courseId: string
  readonly title: string
  readonly description: string | null
  readonly maxPoints: number
  readonly dueDate: string
  readonly latePenaltyPercent: number
  readonly maxLateDays: number
  readonly createdAt: string
  readonly updatedAt: string
}

export type SubmissionStatus = 'submitted' | 'graded' | 'returned'

export interface Submission {
  readonly id: string
  readonly assignmentId: string
  readonly studentId: string
  readonly filePath: string
  readonly submittedAt: string
  readonly status: SubmissionStatus
  readonly isLate: boolean
  readonly daysLate: number
  readonly createdAt: string
  readonly updatedAt: string
}

export type GradeStatus = 'draft' | 'published'

export interface Grade {
  readonly id: string
  readonly submissionId: string
  readonly studentId: string
  readonly assignmentId: string
  readonly score: number
  readonly adjustedScore: number
  readonly feedback: string | null
  readonly status: GradeStatus
  readonly gradedAt: string | null
  readonly publishedAt: string | null
  readonly gradedBy: string
  readonly createdAt: string
  readonly updatedAt: string
}

export interface GradeEntry {
  readonly assignmentId: string
  readonly assignmentTitle: string
  readonly maxPoints: number
  readonly score: number | null
  readonly adjustedScore: number | null
  readonly status: GradeStatus | 'ungraded'
  readonly submittedAt: string | null
  readonly gradedAt: string | null
  readonly feedback: string | null
  readonly isLate: boolean
}

export interface Gradebook {
  readonly courseId: string
  readonly studentId: string
  readonly entries: readonly GradeEntry[]
  readonly totalPoints: number
  readonly earnedPoints: number
  readonly percentage: number
}

// ---------------------------------------------------------------------------
// Content Management (cms DB)
// ---------------------------------------------------------------------------

export interface Module {
  readonly id: string
  readonly courseId: string
  readonly name: string
  readonly description: string | null
  readonly displayOrder: number
  readonly createdBy: string
  readonly createdAt: string
  readonly updatedAt: string
}

export interface Lesson {
  readonly id: string
  readonly moduleId: string
  readonly name: string
  readonly description: string | null
  readonly displayOrder: number
  readonly createdAt: string
  readonly updatedAt: string
}

export interface Resource {
  readonly id: string
  readonly lessonId: string
  readonly name: string
  readonly description: string | null
  readonly contentType: string
  readonly fileSize: number
  readonly storageKey: string
  readonly manifestUrl: string | null
  readonly durationSeconds: number | null
  readonly published: boolean
  readonly downloadable: boolean
  readonly copyrightSetting: string
  readonly displayOrder: number
  readonly createdAt: string
  readonly updatedAt: string
}

export interface ProgressRecord {
  readonly id: string
  readonly studentId: string
  readonly resourceId: string
  readonly completed: boolean
  readonly completedAt: string | null
  readonly lastPositionSeconds: number | null
  readonly updatedAt: string
}

export interface UploadSession {
  readonly id: string
  readonly userId: string
  readonly lessonId: string | null
  readonly filename: string
  readonly contentType: string
  readonly totalSize: number
  readonly chunkSize: number
  readonly totalChunks: number
  readonly uploadedChunks: number
  readonly storageKey: string
  readonly status: string
  readonly createdAt: string
  readonly expiresAt: string
  readonly completedAt: string | null
}

export interface TranscodingJob {
  readonly id: string
  readonly resourceId: string
  readonly status: string
  readonly retryCount: number
  readonly errorMessage: string | null
  readonly createdAt: string
  readonly startedAt: string | null
  readonly completedAt: string | null
}

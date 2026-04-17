package models

import (
	"database/sql"
	"time"
)

// EventLog represents a single event in the event_log table.
type EventLog struct {
	ID           string         `json:"id"`
	Time         time.Time      `json:"time"`
	TenantID     string         `json:"tenantId"`
	EventType    string         `json:"eventType"`
	UserID       sql.NullString `json:"userId,omitempty"`
	CourseID     sql.NullString `json:"courseId,omitempty"`
	AssignmentID sql.NullString `json:"assignmentId,omitempty"`
	Metadata     []byte         `json:"metadata,omitempty"`
}

// StudentProgress tracks per-student per-course progress.
type StudentProgress struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenantId"`
	StudentID          string    `json:"studentId"`
	CourseID           string    `json:"courseId"`
	CompletionPct      float64   `json:"completionPct"`
	TimeOnTaskMinutes  int       `json:"timeOnTaskMinutes"`
	LessonsCompleted   int       `json:"lessonsCompleted"`
	LastActivityAt     time.Time `json:"lastActivityAt,omitempty"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// GradeStats holds pre-computed grade statistics.
type GradeStats struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenantId"`
	CourseID     string    `json:"courseId"`
	AssignmentID string    `json:"assignmentId,omitempty"`
	MeanScore    float64   `json:"meanScore"`
	MedianScore  float64   `json:"medianScore"`
	P25          float64   `json:"p25"`
	P75          float64   `json:"p75"`
	StdDev       float64   `json:"stdDev"`
	StudentCount int       `json:"studentCount"`
	ComputedAt   time.Time `json:"computedAt"`
}

// PlatformMetrics aggregates platform-wide statistics.
type PlatformMetrics struct {
	ActiveTenants int `json:"activeTenants"`
	DAU           int `json:"dau"`
	MAU           int `json:"mau"`
	TotalCourses  int `json:"totalCourses"`
	TotalStudents int `json:"totalStudents"`
}

// TenantMetrics holds per-tenant usage data.
type TenantMetrics struct {
	TenantID      string `json:"tenantId"`
	EventCount    int    `json:"eventCount"`
	ActiveUsers   int    `json:"activeUsers"`
	TotalStudents int    `json:"totalStudents"`
	TotalCourses  int    `json:"totalCourses"`
}

// CourseEngagement holds course-level engagement metrics.
type CourseEngagement struct {
	CourseID           string  `json:"courseId"`
	TotalStudents      int     `json:"totalStudents"`
	AvgCompletionPct   float64 `json:"avgCompletionPct"`
	TotalSubmissions   int     `json:"totalSubmissions"`
	AvgTimeOnTask      float64 `json:"avgTimeOnTask"`
	ActiveLast7Days    int     `json:"activeLast7Days"`
}

// DistributionBucket is a single bucket in a grade distribution histogram.
type DistributionBucket struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Count int     `json:"count"`
}

// GradeDistribution is the full grade distribution for a course/assignment.
type GradeDistribution struct {
	AssignmentID  string               `json:"assignmentId,omitempty"`
	CourseID      string               `json:"courseId"`
	Buckets       []DistributionBucket `json:"buckets"`
	Mean          float64              `json:"mean"`
	Median        float64              `json:"median"`
	P25           float64              `json:"p25"`
	P75           float64              `json:"p75"`
	TotalStudents int                  `json:"totalStudents"`
}

// StudentComparison extends GradeDistribution with the student's position.
type StudentComparison struct {
	GradeDistribution
	StudentScore     float64 `json:"studentScore"`
	StudentPercentile float64 `json:"studentPercentile"`
	LetterGrade      string  `json:"letterGrade"`
}

// TimeOnTask holds per-course time-on-task data for a student.
type TimeOnTask struct {
	CourseID          string `json:"courseId"`
	TotalMinutes      int    `json:"totalMinutes"`
	LessonsCompleted  int    `json:"lessonsCompleted"`
}

// Kafka event types

type LessonCompletedEvent struct {
	TenantID        string  `json:"tenantId"`
	StudentID       string  `json:"studentId"`
	CourseID        string  `json:"courseId"`
	LessonID        string  `json:"lessonId"`
	CompletionPct   float64 `json:"completionPct"`
	DurationMinutes int     `json:"durationMinutes"`
}

type SubmissionEvent struct {
	TenantID     string `json:"tenantId"`
	StudentID    string `json:"studentId"`
	CourseID     string `json:"courseId"`
	AssignmentID string `json:"assignmentId"`
}

type GradeEvent struct {
	TenantID     string  `json:"tenantId"`
	StudentID    string  `json:"studentId"`
	CourseID     string  `json:"courseId"`
	AssignmentID string  `json:"assignmentId"`
	Score        float64 `json:"score"`
	MaxScore     float64 `json:"maxScore"`
}

type RoomEndedEvent struct {
	TenantID       string   `json:"tenantId"`
	RoomID         string   `json:"roomId"`
	CourseID       string   `json:"courseId"`
	Participants   []string `json:"participants"`
	DurationMinutes int     `json:"durationMinutes"`
}

type MessageSentEvent struct {
	TenantID string `json:"tenantId"`
	UserID   string `json:"userId"`
	CourseID string `json:"courseId"`
}

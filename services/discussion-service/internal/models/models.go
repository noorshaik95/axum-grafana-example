package models

import "time"

type Thread struct {
	ID             string
	TenantID       string
	CourseID       string
	Title          string
	CreatedBy      string
	CreatedAt      time.Time
	LastActivityAt time.Time
	ReplyCount     int
}

type Post struct {
	ID           string
	ThreadID     string
	AuthorID     string
	Content      string
	ParentPostID *string
	CreatedAt    time.Time
	EditedAt     *time.Time
}

type Mention struct {
	ID              string
	PostID          string
	MentionedUserID string
	SeenAt          *time.Time
}

type InboxItem struct {
	ID          string
	UserID      string
	Type        string // mention, reply, feedback, announcement
	ReferenceID string
	SeenAt      *time.Time
	CreatedAt   time.Time
}

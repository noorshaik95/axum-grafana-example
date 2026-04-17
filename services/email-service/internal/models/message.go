package models

import (
	"time"
)

type Message struct {
	ID                string     `json:"id" db:"id"`
	TenantID          string     `json:"tenant_id" db:"tenant_id"`
	ThreadID          string     `json:"thread_id" db:"thread_id"`
	ParentID          *string    `json:"parent_id,omitempty" db:"parent_id"`
	FromUserID        string     `json:"from_user_id" db:"from_user_id"`
	Subject           *string    `json:"subject,omitempty" db:"subject"`
	Body              string     `json:"body" db:"body"`
	IsDeletedBySender bool       `json:"is_deleted_by_sender" db:"is_deleted_by_sender"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	Recipients        []Recipient `json:"recipients,omitempty" db:"-"`
}

type Recipient struct {
	ID              string     `json:"id" db:"id"`
	MessageID       string     `json:"message_id" db:"message_id"`
	RecipientUserID string     `json:"recipient_user_id" db:"recipient_user_id"`
	TenantID        string     `json:"tenant_id" db:"tenant_id"`
	IsRead          bool       `json:"is_read" db:"is_read"`
	IsArchived      bool       `json:"is_archived" db:"is_archived"`
	ReadAt          *time.Time `json:"read_at,omitempty" db:"read_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

type SendMessageRequest struct {
	TenantID     string   `json:"tenant_id"`
	RecipientIDs []string `json:"recipient_ids"`
	Subject      string   `json:"subject"`
	Body         string   `json:"body"`
}

type ReplyMessageRequest struct {
	Body string `json:"body"`
}

type InboxMessage struct {
	ID         string     `json:"id" db:"id"`
	TenantID   string     `json:"tenant_id" db:"tenant_id"`
	ThreadID   string     `json:"thread_id" db:"thread_id"`
	FromUserID string     `json:"from_user_id" db:"from_user_id"`
	Subject    *string    `json:"subject,omitempty" db:"subject"`
	Body       string     `json:"body" db:"body"`
	IsRead     bool       `json:"is_read" db:"is_read"`
	IsArchived bool       `json:"is_archived" db:"is_archived"`
	ReadAt     *time.Time `json:"read_at,omitempty" db:"read_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

type PaginatedResponse struct {
	Messages   []InboxMessage `json:"messages"`
	NextCursor string         `json:"next_cursor,omitempty"`
	HasMore    bool           `json:"has_more"`
}

type UnreadCountResponse struct {
	Count int `json:"count"`
}

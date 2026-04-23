package models

import (
	"errors"
	"time"
)

// Attachment kinds.
const (
	AttachmentKindAttachment   = "attachment"
	AttachmentKindStarterCode  = "starter_code"
	AttachmentKindInstructions = "instructions"
)

// AssignmentAttachment is a per-assignment file in MinIO (W9.6).
type AssignmentAttachment struct {
	ID           string    `json:"id"`
	AssignmentID string    `json:"assignment_id"`
	FilePath     string    `json:"file_path"`
	FileName     string    `json:"file_name"`
	ContentType  string    `json:"content_type,omitempty"`
	SizeBytes    int64     `json:"size_bytes"`
	Kind         string    `json:"kind"`
	SignedURL    string    `json:"signed_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Validate ensures required attachment fields are set.
func (a *AssignmentAttachment) Validate() error {
	if a.AssignmentID == "" {
		return errors.New("assignment_id is required")
	}
	if a.FilePath == "" {
		return errors.New("file_path is required")
	}
	if a.FileName == "" {
		return errors.New("file_name is required")
	}
	if a.Kind == "" {
		a.Kind = AttachmentKindAttachment
	}
	switch a.Kind {
	case AttachmentKindAttachment, AttachmentKindStarterCode, AttachmentKindInstructions:
	default:
		return errors.New("kind must be one of: attachment, starter_code, instructions")
	}
	return nil
}

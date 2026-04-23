package service

import (
	"context"
	"fmt"
	"time"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
)

// URLSigner produces signed URLs for private object storage. MinIO in prod,
// a simple HMAC signer in tests.
type URLSigner interface {
	SignURL(path string, ttl time.Duration) (string, error)
}

// AttachmentService exposes per-assignment file lookups (W9.6).
type AttachmentService interface {
	Create(ctx context.Context, a *models.AssignmentAttachment) (*models.AssignmentAttachment, error)
	ListWithSignedURLs(ctx context.Context, assignmentID string, ttl time.Duration) ([]*models.AssignmentAttachment, error)
	Delete(ctx context.Context, id string) error
}

type attachmentService struct {
	repo   repository.AttachmentRepository
	signer URLSigner
}

// NewAttachmentService constructs the default attachment service.
func NewAttachmentService(repo repository.AttachmentRepository, signer URLSigner) AttachmentService {
	return &attachmentService{repo: repo, signer: signer}
}

func (s *attachmentService) Create(ctx context.Context, a *models.AssignmentAttachment) (*models.AssignmentAttachment, error) {
	if err := a.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *attachmentService) ListWithSignedURLs(ctx context.Context, assignmentID string, ttl time.Duration) ([]*models.AssignmentAttachment, error) {
	if assignmentID == "" {
		return nil, fmt.Errorf("assignment_id is required")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	attachments, err := s.repo.ListByAssignment(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if s.signer == nil {
		return attachments, nil
	}
	for _, a := range attachments {
		signed, err := s.signer.SignURL(a.FilePath, ttl)
		if err != nil {
			return nil, fmt.Errorf("sign url for %s: %w", a.FilePath, err)
		}
		a.SignedURL = signed
	}
	return attachments, nil
}

func (s *attachmentService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}
	return s.repo.Delete(ctx, id)
}

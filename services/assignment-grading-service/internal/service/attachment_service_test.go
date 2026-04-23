package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"slate/services/assignment-grading-service/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockAttachmentRepo implements repository.AttachmentRepository.
type mockAttachmentRepo struct {
	mock.Mock
}

func (m *mockAttachmentRepo) Create(ctx context.Context, a *models.AssignmentAttachment) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}
func (m *mockAttachmentRepo) GetByID(ctx context.Context, id string) (*models.AssignmentAttachment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AssignmentAttachment), args.Error(1)
}
func (m *mockAttachmentRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *mockAttachmentRepo) ListByAssignment(ctx context.Context, assignmentID string) ([]*models.AssignmentAttachment, error) {
	args := m.Called(ctx, assignmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AssignmentAttachment), args.Error(1)
}

// testSigner is a deterministic HMAC signer used to verify SignedURL output.
type testSigner struct {
	secret []byte
}

func (s *testSigner) SignURL(path string, ttl time.Duration) (string, error) {
	exp := time.Now().Add(ttl).Unix()
	mac := hmac.New(sha256.New, s.secret)
	fmt.Fprintf(mac, "%s:%d", path, exp)
	return fmt.Sprintf("https://files.local/%s?exp=%d&sig=%s", path, exp, hex.EncodeToString(mac.Sum(nil))), nil
}

func TestAttachmentService_ListWithSignedURLs(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAttachmentRepo)
	signer := &testSigner{secret: []byte("test-secret")}

	repo.On("ListByAssignment", ctx, "a1").Return([]*models.AssignmentAttachment{
		{ID: "att1", AssignmentID: "a1", FilePath: "starter/a1/main.py", FileName: "main.py", Kind: models.AttachmentKindStarterCode},
		{ID: "att2", AssignmentID: "a1", FilePath: "instructions/a1/spec.pdf", FileName: "spec.pdf", Kind: models.AttachmentKindInstructions},
	}, nil)

	svc := NewAttachmentService(repo, signer)
	atts, err := svc.ListWithSignedURLs(ctx, "a1", 5*time.Minute)
	assert.NoError(t, err)
	assert.Len(t, atts, 2)
	for _, a := range atts {
		assert.NotEmpty(t, a.SignedURL)
		assert.Contains(t, a.SignedURL, "sig=")
		assert.Contains(t, a.SignedURL, a.FilePath)
	}
}

func TestAttachmentService_CreateValidates(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAttachmentRepo)
	svc := NewAttachmentService(repo, nil)

	_, err := svc.Create(ctx, &models.AssignmentAttachment{FilePath: "p", FileName: "n"})
	assert.Error(t, err, "should require assignment_id")

	good := &models.AssignmentAttachment{AssignmentID: "a1", FilePath: "p", FileName: "n", Kind: models.AttachmentKindStarterCode}
	repo.On("Create", ctx, good).Return(nil)
	_, err = svc.Create(ctx, good)
	assert.NoError(t, err)
}

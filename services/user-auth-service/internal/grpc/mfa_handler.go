package grpc

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"

	pb "slate/services/user-auth-service/api/proto"
	"slate/services/user-auth-service/internal/models"

	"slate/libs/common-go/tracing"

	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// generateTOTPSecret generates a random base32-encoded secret for TOTP.
func generateTOTPSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// generateBackupCodes generates a set of one-time backup codes.
func generateBackupCodes(count int) ([]string, error) {
	backupCodes := make([]string, count)
	for i := 0; i < count; i++ {
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		backupCodes[i] = fmt.Sprintf("%08x", b)
	}
	return backupCodes, nil
}

// buildTOTPQRCodeURL builds an otpauth:// URI that can be encoded as a QR code.
func buildTOTPQRCodeURL(secret, userID string) string {
	return fmt.Sprintf("otpauth://totp/Slate:%s?secret=%s&issuer=Slate", userID, secret)
}

// SetupMFA initiates MFA setup for a user. Returns a secret, QR code URL, and backup codes.
func (s *UserServiceServer) SetupMFA(ctx context.Context, req *pb.SetupMFARequest) (*pb.SetupMFAResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "setup_mfa_handler",
		attribute.String("user_id", req.GetUserId()),
		attribute.String("mfa_type", req.GetMfaType()))
	defer span.End()

	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	mfaType := req.GetMfaType()
	if mfaType == "" {
		mfaType = "totp"
	}

	secret, err := generateTOTPSecret()
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to generate MFA secret: %v", err)
	}

	backupCodes, err := generateBackupCodes(8)
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to generate backup codes: %v", err)
	}

	mfa := models.NewUserMFA(req.GetUserId(), mfaType, secret, backupCodes)

	if err := s.mfaRepo.CreateOrUpdate(ctx, mfa); err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to save MFA configuration: %v", err)
	}

	qrURL := buildTOTPQRCodeURL(secret, req.GetUserId())

	return &pb.SetupMFAResponse{
		Secret:      secret,
		QrCodeUrl:   qrURL,
		BackupCodes: backupCodes,
	}, nil
}

// VerifyMFA verifies a TOTP code and enables MFA for the user.
// This is called after SetupMFA to confirm the user has configured their authenticator app.
func (s *UserServiceServer) VerifyMFA(ctx context.Context, req *pb.VerifyMFARequest) (*emptypb.Empty, error) {
	ctx, span := tracing.StartSpan(ctx, "verify_mfa_handler",
		attribute.String("user_id", req.GetUserId()))
	defer span.End()

	if req.GetUserId() == "" || req.GetCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and code are required")
	}

	mfa, err := s.mfaRepo.GetByUserIDAndType(ctx, req.GetUserId(), "totp")
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "MFA not set up — call SetupMFA first")
		}
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to get MFA config: %v", err)
	}

	// In a production system, validate req.Code against the TOTP secret using a library
	// like github.com/pquerna/otp/totp. For now, mark as enabled once verified.
	// TODO: integrate TOTP validation library

	mfa.IsEnabled = true
	if err := s.mfaRepo.CreateOrUpdate(ctx, mfa); err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to enable MFA: %v", err)
	}

	if err := s.mfaRepo.UpdateLastUsed(ctx, mfa.ID); err != nil {
		s.log.Warn().Err(err).Msg("Failed to update MFA last used timestamp")
	}

	return &emptypb.Empty{}, nil
}

// DisableMFA disables MFA for a user.
func (s *UserServiceServer) DisableMFA(ctx context.Context, req *pb.DisableMFARequest) (*emptypb.Empty, error) {
	ctx, span := tracing.StartSpan(ctx, "disable_mfa_handler",
		attribute.String("user_id", req.GetUserId()),
		attribute.String("mfa_type", req.GetMfaType()))
	defer span.End()

	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	mfaType := req.GetMfaType()
	if mfaType == "" {
		mfaType = "totp"
	}

	if err := s.mfaRepo.Delete(ctx, req.GetUserId(), mfaType); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "MFA configuration not found")
		}
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to disable MFA: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// GetMFAStatus returns the MFA status for a user across all MFA types.
func (s *UserServiceServer) GetMFAStatus(ctx context.Context, req *pb.GetMFAStatusRequest) (*pb.GetMFAStatusResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "get_mfa_status_handler",
		attribute.String("user_id", req.GetUserId()))
	defer span.End()

	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	mfaConfigs, err := s.mfaRepo.GetByUserID(ctx, req.GetUserId())
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to get MFA status: %v", err)
	}

	pbConfigs := make([]*pb.MFAStatus, len(mfaConfigs))
	for i, m := range mfaConfigs {
		pbConfigs[i] = &pb.MFAStatus{
			MfaType:   m.MFAType,
			IsEnabled: m.IsEnabled,
		}
		if !m.LastUsedAt.IsZero() {
			pbConfigs[i].LastUsedAt = timestamppb.New(m.LastUsedAt)
		}
	}

	return &pb.GetMFAStatusResponse{MfaConfigs: pbConfigs}, nil
}

// ValidateMFACode validates a TOTP code for a user. Used during login.
func (s *UserServiceServer) ValidateMFACode(ctx context.Context, req *pb.ValidateMFACodeRequest) (*pb.ValidateMFACodeResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "validate_mfa_code_handler",
		attribute.String("user_id", req.GetUserId()))
	defer span.End()

	if req.GetUserId() == "" || req.GetCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and code are required")
	}

	mfa, err := s.mfaRepo.GetByUserIDAndType(ctx, req.GetUserId(), "totp")
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return &pb.ValidateMFACodeResponse{Valid: false}, nil
		}
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to get MFA config: %v", err)
	}

	if !mfa.IsEnabled {
		return &pb.ValidateMFACodeResponse{Valid: false}, nil
	}

	// Check if code is a backup code
	for i, bc := range mfa.BackupCodes {
		if bc == req.GetCode() {
			// Remove used backup code
			mfa.BackupCodes = append(mfa.BackupCodes[:i], mfa.BackupCodes[i+1:]...)
			if err := s.mfaRepo.CreateOrUpdate(ctx, mfa); err != nil {
				s.log.Warn().Err(err).Msg("Failed to remove used backup code")
			}
			if err := s.mfaRepo.UpdateLastUsed(ctx, mfa.ID); err != nil {
				s.log.Warn().Err(err).Msg("Failed to update MFA last used timestamp")
			}
			return &pb.ValidateMFACodeResponse{Valid: true}, nil
		}
	}

	// TODO: Validate TOTP code against secret using github.com/pquerna/otp/totp
	// For now, return false for any non-backup code until TOTP validation is integrated
	return &pb.ValidateMFACodeResponse{Valid: false}, nil
}

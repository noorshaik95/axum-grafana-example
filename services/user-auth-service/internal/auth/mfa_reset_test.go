package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"slate/services/user-auth-service/internal/models"
	"slate/services/user-auth-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
)

type stubMFAStore struct {
	getErr   error
	deleted  []string
	configs  []*models.UserMFA
}

func (s *stubMFAStore) GetByUserID(_ context.Context, _ string) ([]*models.UserMFA, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.configs, nil
}

func (s *stubMFAStore) Delete(_ context.Context, userID, mfaType string) error {
	s.deleted = append(s.deleted, userID+":"+mfaType)
	return nil
}

func mintPlatformToken(t *testing.T, key *rsa.PrivateKey, aud, sub string, exp time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{"aud": aud, "sub": sub, "exp": exp.Unix(), "iat": time.Now().Unix()}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func TestMFAReset_PlatformAdminAllowed(t *testing.T) {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	verifier := NewRSAPlatformVerifier(&k.PublicKey)
	store := &stubMFAStore{configs: []*models.UserMFA{{UserID: "u", MFAType: "totp"}}}
	audit := repository.NewInMemoryAuditRepository()
	svc := NewMFAResetService(store, verifier, audit)

	token := mintPlatformToken(t, k, "platform", "admin-1", time.Now().Add(time.Minute))
	if err := svc.Reset(context.Background(), token, "u", "10.0.0.1"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if len(store.deleted) == 0 {
		t.Fatal("delete not called")
	}
	if len(audit.ByAction(models.AuditActionMFAChange)) != 1 {
		t.Fatal("want mfa_change audit row")
	}
}

func TestMFAReset_NonPlatformAudRejected(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	svc := NewMFAResetService(&stubMFAStore{}, NewRSAPlatformVerifier(&k.PublicKey), repository.NewInMemoryAuditRepository())
	token := mintPlatformToken(t, k, "tenant", "admin-1", time.Now().Add(time.Minute))
	if err := svc.Reset(context.Background(), token, "u", ""); err != ErrMFAResetNotPlatform {
		t.Fatalf("want ErrMFAResetNotPlatform, got %v", err)
	}
}

func TestMFAReset_MissingUserOrToken(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	svc := NewMFAResetService(&stubMFAStore{}, NewRSAPlatformVerifier(&k.PublicKey), nil)
	if err := svc.Reset(context.Background(), "tok", "", ""); err != ErrMFAResetUserRequired {
		t.Fatalf("want user required, got %v", err)
	}
	if err := svc.Reset(context.Background(), "", "u", ""); err != ErrMFAResetTokenMissing {
		t.Fatalf("want token missing, got %v", err)
	}
}

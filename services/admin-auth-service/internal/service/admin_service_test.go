package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"slate/services/admin-auth-service/internal/audit"
	jwtpkg "slate/services/admin-auth-service/internal/jwt"
	"slate/services/admin-auth-service/internal/repository"
)

// fakeProducer captures emitted Kafka payloads in-memory.
type fakeProducer struct {
	admin         [][]byte
	impersonation [][]byte
}

func (p *fakeProducer) ProduceAdminAction(_ context.Context, _ string, payload []byte) error {
	p.admin = append(p.admin, append([]byte{}, payload...))
	return nil
}
func (p *fakeProducer) ProduceImpersonationStarted(_ context.Context, _ string, payload []byte) error {
	p.impersonation = append(p.impersonation, append([]byte{}, payload...))
	return nil
}
func (p *fakeProducer) Close() error { return nil }

type harness struct {
	db       *sql.DB
	mock     sqlmock.Sqlmock
	svc      *AdminService
	tokens   *jwtpkg.TokenService
	privKey  *rsa.PrivateKey
	producer *fakeProducer
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tokens := jwtpkg.NewTokenService(
		"test-secret-must-be-at-least-32-chars-long-abc", time.Hour, 24*time.Hour, 5*time.Minute,
	)
	tokens.WithRSAKeys(priv, &priv.PublicKey)

	producer := &fakeProducer{}
	recorder := audit.NewRecorder(
		repository.NewAuditRepository(db),
		producer,
	)
	svc := NewAdminService(
		repository.NewAdminRepository(db),
		recorder,
		tokens,
		NewMemoryRevoker(),
		"http://{slug}.slate.local/auth/impersonate?token={token}",
	)
	return &harness{db: db, mock: mock, svc: svc, tokens: tokens, privKey: priv, producer: producer}
}

func (h *harness) expectAdminLookupByEmail(email string, hash string, id string, roles []string, disabled bool) {
	rolesArr := pq.Array(roles)
	_ = rolesArr
	rows := sqlmock.NewRows([]string{
		"id", "email", "password_hash", "full_name", "roles", "disabled", "created_at", "updated_at", "last_login_at",
	}).AddRow(id, email, hash, "Ada Admin", pq.StringArray(roles), disabled, time.Now(), time.Now(), nil)
	h.mock.ExpectQuery(regexp.QuoteMeta("FROM platform_admins WHERE email = $1")).
		WithArgs(email).
		WillReturnRows(rows)
}

func (h *harness) expectAdminLookupByID(id string, roles []string, disabled bool) {
	rows := sqlmock.NewRows([]string{
		"id", "email", "password_hash", "full_name", "roles", "disabled", "created_at", "updated_at", "last_login_at",
	}).AddRow(id, "ops@slate.local", "ignored", "Ada Admin", pq.StringArray(roles), disabled, time.Now(), time.Now(), nil)
	h.mock.ExpectQuery(regexp.QuoteMeta("FROM platform_admins WHERE id = $1")).
		WithArgs(id).
		WillReturnRows(rows)
}

func (h *harness) expectAuditInsert() {
	h.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO platform_audit")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("aud-1", time.Now()))
}

func (h *harness) expectLastLoginUpdate(id string) {
	h.mock.ExpectExec(regexp.QuoteMeta("UPDATE platform_admins SET last_login_at")).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func TestAdminLoginValidateLogoutFlow(t *testing.T) {
	h := newHarness(t)

	pw := "Pa$$w0rd-ok1"
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	require.NoError(t, err)

	// ---- Login expectations ----
	h.expectAdminLookupByEmail("ops@slate.local", string(hash), "admin-1", []string{"superadmin"}, false)
	h.expectLastLoginUpdate("admin-1")
	h.expectAuditInsert() // admin.login

	res, err := h.svc.Login(context.Background(), "ops@slate.local", pw)
	require.NoError(t, err)
	require.NotEmpty(t, res.AccessToken)
	require.NotEmpty(t, res.RefreshToken)

	// ---- Validate expectations (no DB calls required) ----
	v := h.svc.ValidateAdminToken(context.Background(), res.AccessToken)
	require.True(t, v.Valid, "validate should pass immediately after login")
	require.Equal(t, "admin-1", v.UserID)
	require.Equal(t, "platform", v.Audience)
	require.Contains(t, v.Roles, "superadmin")

	// ---- Logout expectations ----
	h.expectAuditInsert() // admin.logout
	require.NoError(t, h.svc.Logout(context.Background(), res.AccessToken))

	// Token should now be revoked.
	v2 := h.svc.ValidateAdminToken(context.Background(), res.AccessToken)
	require.False(t, v2.Valid, "validate should fail after logout revokes token")

	require.NoError(t, h.mock.ExpectationsWereMet())

	// Kafka audit events were produced.
	require.GreaterOrEqual(t, len(h.producer.admin), 2, "expected at least login+logout admin events")
}

func TestAdminLoginRejectsBadPassword(t *testing.T) {
	h := newHarness(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("right-password-123"), bcrypt.MinCost)
	require.NoError(t, err)
	h.expectAdminLookupByEmail("ops@slate.local", string(hash), "admin-1", []string{"superadmin"}, false)
	h.expectAuditInsert() // admin.login_failed

	_, err = h.svc.Login(context.Background(), "ops@slate.local", "wrong-password")
	require.ErrorIs(t, err, ErrInvalidCredentials)
	require.NoError(t, h.mock.ExpectationsWereMet())
}

func TestImpersonateRoundTripAndVerify(t *testing.T) {
	h := newHarness(t)

	h.expectAdminLookupByID("admin-42", []string{"superadmin"}, false)
	h.expectAuditInsert() // impersonation.started

	res, err := h.svc.Impersonate(context.Background(), "admin-42", "tenant-eastfield", "eastfield", "student-9")
	require.NoError(t, err)
	require.Contains(t, res.RedirectURL, "http://eastfield.slate.local/auth/impersonate?token=")
	require.Contains(t, res.RedirectURL, res.Token, "redirect_url should embed the freshly-minted token")
	require.True(t, res.ExpiresAt.After(time.Now()))

	// Verify the token using the public key (same pattern tenant user-auth will use).
	claims, err := h.tokens.VerifyImpersonationToken(res.Token)
	require.NoError(t, err)
	require.Equal(t, "admin-42", claims.AdminUserID)
	require.Equal(t, "eastfield", claims.TenantSlug)
	require.Equal(t, "student-9", claims.TargetUserID)
	require.Equal(t, res.ImpersonationID, claims.ImpersonationID)

	require.NoError(t, h.mock.ExpectationsWereMet())

	// audit.impersonation_started Kafka event fired.
	require.Len(t, h.producer.impersonation, 1, "expected exactly one impersonation kafka event")
}

func TestImpersonateRejectsRolesWithoutPermission(t *testing.T) {
	h := newHarness(t)

	h.expectAdminLookupByID("admin-42", []string{"readonly"}, false)
	h.expectAuditInsert() // impersonation.rejected

	_, err := h.svc.Impersonate(context.Background(), "admin-42", "tenant-eastfield", "eastfield", "student-9")
	require.ErrorIs(t, err, ErrForbidden)
	require.NoError(t, h.mock.ExpectationsWereMet())
	// No impersonation event should have been emitted since we rejected.
	require.Empty(t, h.producer.impersonation)
}

func TestRegisterCreatesAdmin(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO platform_admins")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow("admin-new", time.Now(), time.Now()))
	h.expectAuditInsert() // admin.register

	admin, err := h.svc.Register(context.Background(), "new@slate.local", "Pa$$w0rd-xyz", "New Admin", []string{"support"}, "admin-42")
	require.NoError(t, err)
	require.Equal(t, "admin-new", admin.ID)
	require.Equal(t, "new@slate.local", admin.Email)
	require.Contains(t, admin.Roles, "support")
	require.NoError(t, h.mock.ExpectationsWereMet())
}

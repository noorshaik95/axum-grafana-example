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

	"slate/services/admin-auth-service/internal/audit"
	jwtpkg "slate/services/admin-auth-service/internal/jwt"
	"slate/services/admin-auth-service/internal/repository"
)

// w25Harness wires the full W2.5 dependency graph onto a sqlmock-backed DB.
type w25Harness struct {
	db       *sql.DB
	mock     sqlmock.Sqlmock
	svc      *AdminService
	tokens   *jwtpkg.TokenService
	producer *fakeProducer
}

func newW25Harness(t *testing.T) *w25Harness {
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
	recorder := audit.NewRecorder(repository.NewAuditRepository(db), producer)

	svc := NewAdminService(
		repository.NewAdminRepository(db),
		recorder,
		tokens,
		NewMemoryRevoker(),
		"http://{slug}.slate.local/auth/impersonate?token={token}",
	).WithReads(
		repository.NewRoleRepository(db),
		repository.NewAuditRepository(db),
	)
	return &w25Harness{db: db, mock: mock, svc: svc, tokens: tokens, producer: producer}
}

// ---- ListRoles ----------------------------------------------------------

func TestListRoles_Basic(t *testing.T) {
	h := newW25Harness(t)
	h.mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM platform_roles")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	h.mock.ExpectQuery(regexp.QuoteMeta("FROM platform_roles")).
		WillReturnRows(sqlmock.NewRows([]string{"name", "description", "permissions", "created_at"}).
			AddRow("billing", "Billing ops", pq.StringArray{"billing:read", "billing:write"}, time.Now()).
			AddRow("readonly", "Read only", pq.StringArray{"admin:read", "audit:read"}, time.Now()).
			AddRow("superadmin", "Full access", pq.StringArray{"*"}, time.Now()))

	res, err := h.svc.ListRoles(context.Background(), 50, "")
	require.NoError(t, err)
	require.Len(t, res.Roles, 3)
	require.Equal(t, 3, res.Total)
	require.Empty(t, res.NextPageToken, "no more pages when row count <= page size")
	// First role should be "billing" alphabetically.
	require.Equal(t, "billing", res.Roles[0].Key)
	require.Equal(t, "Billing", res.Roles[0].Name)
	require.Contains(t, res.Roles[2].Permissions, "*")
	require.Equal(t, "Super Admin", res.Roles[2].Name, "superadmin → Super Admin human label")
	require.NoError(t, h.mock.ExpectationsWereMet())
}

func TestListRoles_Pagination(t *testing.T) {
	h := newW25Harness(t)
	// Over-fetch by one: we'll ask for page_size=2 and return 3 rows (2+extra).
	h.mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM platform_roles")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
	h.mock.ExpectQuery(regexp.QuoteMeta("FROM platform_roles")).
		WillReturnRows(sqlmock.NewRows([]string{"name", "description", "permissions", "created_at"}).
			AddRow("billing", "", pq.StringArray{}, time.Now()).
			AddRow("readonly", "", pq.StringArray{}, time.Now()).
			AddRow("superadmin", "", pq.StringArray{}, time.Now())) // 3rd row → signals hasMore

	res, err := h.svc.ListRoles(context.Background(), 2, "")
	require.NoError(t, err)
	require.Len(t, res.Roles, 2)
	require.NotEmpty(t, res.NextPageToken, "next token should be set when hasMore")
	// Decode cursor and confirm it points at the last returned row.
	after, err := decodeRoleCursor(res.NextPageToken)
	require.NoError(t, err)
	require.Equal(t, "readonly", after)
	require.NoError(t, h.mock.ExpectationsWereMet())
}

func TestListRoles_InvalidCursor(t *testing.T) {
	h := newW25Harness(t)
	_, err := h.svc.ListRoles(context.Background(), 50, "!!!not-base64!!!")
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestListRoles_ErrReadsUnavailable(t *testing.T) {
	// Service without WithReads() should refuse cleanly.
	svc := NewAdminService(nil, nil, nil, nil, "")
	_, err := svc.ListRoles(context.Background(), 50, "")
	require.ErrorIs(t, err, ErrReadsUnavailable)
}

// ---- GetAuditLog --------------------------------------------------------

func TestGetAuditLog_FilterByActor(t *testing.T) {
	h := newW25Harness(t)

	now := time.Now()
	h.mock.ExpectQuery(regexp.QuoteMeta("FROM platform_audit a\nLEFT JOIN platform_admins u")).
		WithArgs("admin-42", 51).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "actor_id", "action", "target_id", "target_type",
			"metadata", "request_id", "created_at", "admin_email",
		}).
			AddRow("aud-1", "admin-42", "admin.login", "admin-42", "admin",
				[]byte(`{"request_id":"req-1"}`), "req-1", now, "ops@slate.local").
			AddRow("aud-2", "admin-42", "admin.login_failed", "admin-42", "admin",
				[]byte(`{"reason":"bad_password"}`), "", now.Add(-time.Minute), "ops@slate.local"))

	res, err := h.svc.GetAuditLog(context.Background(), AuditFilter{
		ActorID: "admin-42",
		Limit:   50,
	})
	require.NoError(t, err)
	require.Len(t, res.Events, 2)
	require.Equal(t, "success", res.Events[0].Outcome)
	require.Equal(t, "failure", res.Events[1].Outcome)
	require.Equal(t, "bad_password", res.Events[1].Error, "error should be extracted from metadata.reason")
	require.Equal(t, "ops@slate.local", res.Events[0].AdminEmail, "admin_email should come from JOIN")
	require.Empty(t, res.NextCursor, "should not paginate when row count <= limit")
	require.NoError(t, h.mock.ExpectationsWereMet())
}

func TestGetAuditLog_TimeRangeAndPagination(t *testing.T) {
	h := newW25Harness(t)

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	// Over-fetch by one (limit=2 requested → repo called with 3).
	rows := sqlmock.NewRows([]string{
		"id", "actor_id", "action", "target_id", "target_type",
		"metadata", "request_id", "created_at", "admin_email",
	})
	rows.AddRow("e-3", "admin-1", "admin.login", "", "", []byte(`null`), "", to.Add(-time.Minute), "a@x.com")
	rows.AddRow("e-2", "admin-1", "admin.logout", "", "", []byte(`null`), "", to.Add(-2*time.Minute), "a@x.com")
	rows.AddRow("e-1", "admin-1", "impersonation.started", "", "", []byte(`null`), "", to.Add(-3*time.Minute), "a@x.com")

	h.mock.ExpectQuery(regexp.QuoteMeta("FROM platform_audit a")).
		WithArgs(from, to, 3).
		WillReturnRows(rows)

	res, err := h.svc.GetAuditLog(context.Background(), AuditFilter{
		From:  from,
		To:    to,
		Limit: 2,
	})
	require.NoError(t, err)
	require.Len(t, res.Events, 2)
	require.NotEmpty(t, res.NextCursor, "next cursor should be set when hasMore")

	// Decode cursor → should point at the last returned row (e-2).
	ts, id, err := decodeAuditCursor(res.NextCursor)
	require.NoError(t, err)
	require.Equal(t, "e-2", id)
	require.WithinDuration(t, to.Add(-2*time.Minute), ts, time.Second)

	require.NoError(t, h.mock.ExpectationsWereMet())
}

func TestGetAuditLog_ClampsOversizedLimit(t *testing.T) {
	h := newW25Harness(t)

	// Caller asks for 10_000 → service clamps to 500 → repo sees 501.
	h.mock.ExpectQuery(regexp.QuoteMeta("FROM platform_audit")).
		WithArgs(501).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "actor_id", "action", "target_id", "target_type",
			"metadata", "request_id", "created_at", "admin_email",
		}))

	_, err := h.svc.GetAuditLog(context.Background(), AuditFilter{Limit: 10000})
	require.NoError(t, err)
	require.NoError(t, h.mock.ExpectationsWereMet())
}

func TestGetAuditLog_InvalidCursor(t *testing.T) {
	h := newW25Harness(t)
	_, err := h.svc.GetAuditLog(context.Background(), AuditFilter{Cursor: "!!not-valid!!"})
	require.ErrorIs(t, err, ErrInvalidInput)
}

// ---- RefreshToken -------------------------------------------------------

func TestRefreshToken_RotatesAndRevokesOld(t *testing.T) {
	h := newW25Harness(t)

	// Issue an initial refresh token for an admin.
	refresh, _, err := h.tokens.IssueRefreshToken("admin-1", "ops@slate.local", []string{"superadmin"})
	require.NoError(t, err)

	// Refresh path: GetByID → audit insert.
	h.mock.ExpectQuery(regexp.QuoteMeta("FROM platform_admins WHERE id = $1")).
		WithArgs("admin-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "roles", "disabled", "created_at", "updated_at", "last_login_at",
		}).AddRow("admin-1", "ops@slate.local", "", "", pq.StringArray{"superadmin"}, false, time.Now(), time.Now(), nil))
	h.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO platform_audit")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("aud-1", time.Now()))

	res, err := h.svc.RefreshToken(context.Background(), refresh)
	require.NoError(t, err)
	require.NotEmpty(t, res.AccessToken)
	require.NotEmpty(t, res.RefreshToken)
	require.NotEqual(t, refresh, res.RefreshToken, "refresh token must rotate")

	// Old refresh token should now be revoked.
	_, err = h.svc.RefreshToken(context.Background(), refresh)
	require.ErrorIs(t, err, ErrInvalidCredentials, "old refresh token should no longer work after rotation")

	require.NoError(t, h.mock.ExpectationsWereMet())
	// Audit emission: 1 Kafka event for the refresh action.
	require.GreaterOrEqual(t, len(h.producer.admin), 1)
}

func TestRefreshToken_RejectsAccessToken(t *testing.T) {
	h := newW25Harness(t)
	access, _, err := h.tokens.IssueAccessToken("admin-1", "ops@slate.local", nil)
	require.NoError(t, err)

	_, err = h.svc.RefreshToken(context.Background(), access)
	require.ErrorIs(t, err, ErrInvalidCredentials, "access tokens must be rejected on refresh path")
}

func TestRefreshToken_RejectsDisabledAdmin(t *testing.T) {
	h := newW25Harness(t)
	refresh, _, err := h.tokens.IssueRefreshToken("admin-dis", "x@slate.local", nil)
	require.NoError(t, err)

	h.mock.ExpectQuery(regexp.QuoteMeta("FROM platform_admins WHERE id = $1")).
		WithArgs("admin-dis").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "full_name", "roles", "disabled", "created_at", "updated_at", "last_login_at",
		}).AddRow("admin-dis", "x@slate.local", "", "", pq.StringArray{}, true /* disabled */, time.Now(), time.Now(), nil))

	_, err = h.svc.RefreshToken(context.Background(), refresh)
	require.ErrorIs(t, err, ErrAccountDisabled)
	require.NoError(t, h.mock.ExpectationsWereMet())
}

func TestRefreshToken_RejectsEmpty(t *testing.T) {
	h := newW25Harness(t)
	_, err := h.svc.RefreshToken(context.Background(), "")
	require.ErrorIs(t, err, ErrInvalidInput)
}

// ---- Helpers ------------------------------------------------------------

func TestDeriveOutcome(t *testing.T) {
	cases := []struct {
		name    string
		action  string
		meta    map[string]any
		outcome string
		errMsg  string
	}{
		{"login success", "admin.login", nil, "success", ""},
		{"login failed w/ reason", "admin.login_failed", map[string]any{"reason": "bad_password"}, "failure", "bad_password"},
		{"impersonation rejected", "impersonation.rejected", map[string]any{"reason": "insufficient_role"}, "failure", "insufficient_role"},
		{"failed without reason", "admin.login_failed", map[string]any{}, "failure", ""},
		{"logout success", "admin.logout", nil, "success", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o, e := deriveOutcome(tc.action, tc.meta)
			require.Equal(t, tc.outcome, o)
			require.Equal(t, tc.errMsg, e)
		})
	}
}

func TestFlattenMetadata(t *testing.T) {
	m := map[string]any{
		"reason":   "bad_password",
		"attempts": 3,
		"tenant":   map[string]any{"slug": "eastfield"},
		"nil":      nil,
	}
	out := flattenMetadata(m)
	require.Equal(t, "bad_password", out["reason"])
	require.Equal(t, "3", out["attempts"])
	require.Contains(t, out["tenant"], "eastfield")
	require.Equal(t, "", out["nil"])
}

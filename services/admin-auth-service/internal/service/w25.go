package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	commontracing "slate/libs/common-go/tracing"
	"slate/services/admin-auth-service/internal/audit"
	"slate/services/admin-auth-service/internal/models"
	"slate/services/admin-auth-service/internal/repository"
)

// W2.5: ListAdminRoles + GetAuditLog + RefreshAdminToken.

// ErrReadsUnavailable is returned when the W2.5 read-path repositories were
// not attached via WithReads().
var ErrReadsUnavailable = errors.New("admin-auth read path not configured")

// RoleResult is the service-layer view of a role row (proto-agnostic).
type RoleResult struct {
	Key         string
	Name        string
	Description string
	Permissions []string
	CreatedAt   time.Time
}

// ListRolesResult is what ListRoles returns to the handler.
type ListRolesResult struct {
	Roles         []RoleResult
	NextPageToken string
	Total         int
}

// ListRoles returns paginated platform roles with human-readable name labels.
func (s *AdminService) ListRoles(ctx context.Context, pageSize int, pageToken string) (*ListRolesResult, error) {
	if s.roles == nil {
		return nil, ErrReadsUnavailable
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	afterName, err := decodeRoleCursor(pageToken)
	if err != nil {
		return nil, ErrInvalidInput
	}

	// Over-fetch by one to tell whether a next page exists.
	raw, total, err := s.roles.List(ctx, afterName, pageSize+1)
	if err != nil {
		return nil, err
	}
	hasMore := len(raw) > pageSize
	if hasMore {
		raw = raw[:pageSize]
	}

	out := make([]RoleResult, 0, len(raw))
	for _, r := range raw {
		out = append(out, RoleResult{
			Key:         r.Name,
			Name:        humanizeRoleName(r.Name),
			Description: r.Description,
			Permissions: r.Permissions,
			CreatedAt:   r.CreatedAt,
		})
	}

	next := ""
	if hasMore && len(raw) > 0 {
		next = encodeRoleCursor(raw[len(raw)-1].Name)
	}
	return &ListRolesResult{Roles: out, NextPageToken: next, Total: total}, nil
}

// AuditFilter is the transport-agnostic shape of the audit-log query.
type AuditFilter struct {
	ActorID    string
	Action     string
	TargetID   string
	TargetType string
	From       time.Time
	To         time.Time
	Limit      int
	Cursor     string
}

// AuditResultRow is the shaped audit event returned to the handler.
type AuditResultRow struct {
	ID            string
	AdminUserID   string
	AdminEmail    string
	ActionType    string
	TargetType    string
	TargetID      string
	Outcome       string // "success" | "failure"
	Error         string
	OccurredAt    time.Time
	Metadata      map[string]string
	RequestID     string
}

// GetAuditLogResult is what GetAuditLog returns to the handler.
type GetAuditLogResult struct {
	Events     []AuditResultRow
	NextCursor string
}

// GetAuditLog returns a page of platform_audit rows, newest-first,
// with outcome + error derived from the action name and metadata payload.
func (s *AdminService) GetAuditLog(ctx context.Context, f AuditFilter) (*GetAuditLogResult, error) {
	if s.auditReads == nil {
		return nil, ErrReadsUnavailable
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	beforeTime, beforeID, err := decodeAuditCursor(f.Cursor)
	if err != nil {
		return nil, ErrInvalidInput
	}

	// Over-fetch by one to detect more pages.
	rows, err := s.auditReads.Query(ctx, repository.AuditFilter{
		ActorID:    f.ActorID,
		Action:     f.Action,
		TargetID:   f.TargetID,
		TargetType: f.TargetType,
		From:       f.From,
		To:         f.To,
		Limit:      limit + 1,
		BeforeID:   beforeID,
		BeforeTime: beforeTime,
	})
	if err != nil {
		return nil, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	events := make([]AuditResultRow, 0, len(rows))
	for _, r := range rows {
		outcome, errMsg := deriveOutcome(r.Action, r.Metadata)
		events = append(events, AuditResultRow{
			ID:          r.ID,
			AdminUserID: r.ActorID,
			AdminEmail:  r.AdminEmail,
			ActionType:  r.Action,
			TargetType:  r.TargetType,
			TargetID:    r.TargetID,
			Outcome:     outcome,
			Error:       errMsg,
			OccurredAt:  r.CreatedAt,
			Metadata:    flattenMetadata(r.Metadata),
			RequestID:   r.RequestID,
		})
	}

	next := ""
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		next = encodeAuditCursor(last.CreatedAt, last.ID)
	}
	return &GetAuditLogResult{Events: events, NextCursor: next}, nil
}

// RefreshToken validates a refresh token, rotates it, and returns a new pair.
// The old refresh jti is added to the revoker (replay protection).
func (s *AdminService) RefreshToken(ctx context.Context, raw string) (*LoginResult, error) {
	if raw == "" {
		return nil, ErrInvalidInput
	}
	claims, err := s.tokens.ParseAdminToken(raw)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if claims.Type != "refresh" {
		return nil, ErrInvalidCredentials
	}
	if s.revoker != nil && claims.ID != "" {
		revoked, err := s.revoker.IsRevoked(ctx, claims.ID)
		if err == nil && revoked {
			return nil, ErrInvalidCredentials
		}
	}

	admin, err := s.admins.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if admin.Disabled {
		return nil, ErrAccountDisabled
	}

	access, exp, err := s.tokens.IssueAccessToken(admin.ID, admin.Email, admin.Roles)
	if err != nil {
		return nil, err
	}
	refresh, _, err := s.tokens.IssueRefreshToken(admin.ID, admin.Email, admin.Roles)
	if err != nil {
		return nil, err
	}

	// Rotate: revoke the old refresh jti for its remaining TTL.
	if s.revoker != nil && claims.ID != "" {
		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl > 0 {
			_ = s.revoker.Revoke(ctx, claims.ID, ttl)
		}
	}

	_ = s.audits.Record(ctx, &models.AuditEntry{
		ActorID:    admin.ID,
		Action:     audit.ActionAdminTokenRefresh,
		TargetID:   admin.ID,
		TargetType: "admin",
		Metadata:   map[string]any{"request_id": commontracing.RequestIDFromContext(ctx)},
	})

	return &LoginResult{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    exp,
		User:         admin,
	}, nil
}

// ---- helpers ------------------------------------------------------------

// deriveOutcome maps an action name + metadata to (outcome, error) per CONTRACTS.md W2.5.
// Failure actions carry one of the three suffixes emitted by the audit writer today:
// `_failed`, `_rejected`, or the dotted variants `.failed`, `.rejected`.
// `error` is extracted from metadata["reason"] when present.
func deriveOutcome(action string, metadata map[string]any) (string, string) {
	failed := strings.HasSuffix(action, "_failed") || strings.HasSuffix(action, ".failed") ||
		strings.HasSuffix(action, "_rejected") || strings.HasSuffix(action, ".rejected")
	if !failed {
		return "success", ""
	}
	reason := ""
	if v, ok := metadata["reason"]; ok {
		if s, ok := v.(string); ok {
			reason = s
		}
	}
	return "failure", reason
}

func flattenMetadata(m map[string]any) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		switch t := v.(type) {
		case string:
			out[k] = t
		case nil:
			out[k] = ""
		default:
			b, err := json.Marshal(v)
			if err != nil {
				out[k] = fmt.Sprintf("%v", v)
			} else {
				out[k] = string(b)
			}
		}
	}
	return out
}

// humanizeRoleName turns "superadmin" → "Super Admin", "support" → "Support".
// Kept small and deterministic; grows with the role catalog.
func humanizeRoleName(key string) string {
	switch key {
	case "superadmin":
		return "Super Admin"
	case "support":
		return "Support"
	case "billing":
		return "Billing"
	case "readonly":
		return "Read Only"
	default:
		if key == "" {
			return ""
		}
		return strings.ToUpper(key[:1]) + key[1:]
	}
}

// ---- cursor encode/decode ----------------------------------------------

type roleCursor struct {
	AfterName string `json:"a"`
}

func encodeRoleCursor(afterName string) string {
	b, _ := json.Marshal(roleCursor{AfterName: afterName})
	return base64.URLEncoding.EncodeToString(b)
}

func decodeRoleCursor(tok string) (string, error) {
	if tok == "" {
		return "", nil
	}
	raw, err := base64.URLEncoding.DecodeString(tok)
	if err != nil {
		return "", fmt.Errorf("invalid page token: %w", err)
	}
	var c roleCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return "", fmt.Errorf("invalid page token: %w", err)
	}
	return c.AfterName, nil
}

type auditCursor struct {
	BeforeTimeUnixNano int64  `json:"t"`
	BeforeID           string `json:"i"`
}

func encodeAuditCursor(t time.Time, id string) string {
	b, _ := json.Marshal(auditCursor{BeforeTimeUnixNano: t.UnixNano(), BeforeID: id})
	return base64.URLEncoding.EncodeToString(b)
}

func decodeAuditCursor(tok string) (time.Time, string, error) {
	if tok == "" {
		return time.Time{}, "", nil
	}
	raw, err := base64.URLEncoding.DecodeString(tok)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor: %w", err)
	}
	var c auditCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor: %w", err)
	}
	return time.Unix(0, c.BeforeTimeUnixNano), c.BeforeID, nil
}

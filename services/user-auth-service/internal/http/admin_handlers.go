// Package http wires the W14 HTTP endpoints onto an http.ServeMux:
//
//	GET  /api/auth/sso/initiate
//	GET  /api/auth/sso/callback
//	POST /api/admin/auth/test-sso
//	POST /api/auth/impersonate
//	DELETE /api/auth/impersonate/:impersonation_id
//	POST /api/admin/auth/mfa/reset
//
// The handlers are thin adapters — the heavy lifting lives in the sso + auth
// packages.
package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"strings"
	"time"

	commontracing "slate/libs/common-go/tracing"
	"slate/services/user-auth-service/internal/auth"
	"slate/services/user-auth-service/internal/auth/sso"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Handler holds the wiring needed by the HTTP routes. Any field may be nil if
// the corresponding feature is disabled at this tenant (e.g. SSO not
// provisioned).
//
// LandingURLTemplate is the base URL for the option-A impersonation redirect.
// When a GET /auth/impersonate request succeeds the handler builds:
//
//	{LandingURLTemplate}#token={access_token}&expires_at={expires_in}&tenant_slug={slug}
//
// and returns a 302 to that URL. The default value is
// "http://teach.slate.local/auth/impersonate-landing" (overridden via the
// IMPERSONATION_LANDING_URL_TEMPLATE env var).
type Handler struct {
	SSOManager             *sso.Manager
	ImpersonationValidator *auth.ImpersonationValidator
	MFAReset               *auth.MFAResetService
	// LandingURLTemplate is the base URL (no fragment) for the browser redirect.
	LandingURLTemplate string
}

// Register mounts W14 endpoints on the given ServeMux. Every handler is
// wrapped in tracingMiddleware to satisfy plan/CONTRACTS.md §trace.propagation:
// extract traceparent + X-Request-ID inbound (or generate a request id), tag
// the span with request_id + tenant.slug, echo X-Request-ID on the response.
//
// Routes:
//   - GET  /api/auth/sso/initiate
//   - GET  /api/auth/sso/callback       (also POST for SAML-HTTP-POST binding)
//   - POST /api/admin/auth/test-sso
//   - POST /api/auth/impersonate        (token in body or ?token=)
//   - GET  /auth/impersonate            (browser-facing redirect target per
//                                        plan/CONTRACTS.md admin_auth.redirect_url;
//                                        token via ?token=)
//   - DELETE /api/auth/impersonate/:id
//   - POST /api/admin/auth/mfa/reset
func (h *Handler) Register(mux *stdhttp.ServeMux) {
	mux.Handle("/api/auth/sso/initiate", tracingMiddleware("sso.initiate", h.handleSSOInitiate))
	mux.Handle("/api/auth/sso/callback", tracingMiddleware("sso.callback", h.handleSSOCallback))
	mux.Handle("/api/admin/auth/test-sso", tracingMiddleware("sso.test", h.handleTestSSO))
	mux.Handle("/api/auth/impersonate", tracingMiddleware("impersonate.validate", h.handleImpersonate))
	mux.Handle("/auth/impersonate", tracingMiddleware("impersonate.validate", h.handleImpersonate))
	mux.Handle("/api/auth/impersonate/", tracingMiddleware("impersonate.revoke", h.handleImpersonateByID))
	mux.Handle("/api/admin/auth/mfa/reset", tracingMiddleware("mfa.reset", h.handleMFAReset))
}

// tracingMiddleware extracts W3C traceparent + X-Request-ID from the incoming
// request, starts a server span, tags it with request_id + tenant.slug, and
// echoes X-Request-ID on the response. Satisfies
// plan/CONTRACTS.md§trace.propagation for the new W14 HTTP endpoints.
func tracingMiddleware(spanName string, next stdhttp.HandlerFunc) stdhttp.Handler {
	tracer := otel.Tracer("user-auth-service")
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		reqID := r.Header.Get(commontracing.RequestIDHeader)
		if reqID == "" {
			reqID = newRequestID()
		}
		ctx = commontracing.WithRequestID(ctx, reqID)
		if slug := r.Header.Get(commontracing.TenantSlugHeader); slug != "" {
			ctx = commontracing.WithTenantSlug(ctx, slug)
		}
		ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		commontracing.TagSpanWithCorrelation(ctx, span)

		w.Header().Set(commontracing.RequestIDHeader, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// newRequestID returns a 16-byte hex id for responses when the caller did not
// supply X-Request-ID.
func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func (h *Handler) handleSSOInitiate(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodGet {
		writeErr(w, stdhttp.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.SSOManager == nil {
		writeErr(w, stdhttp.StatusNotFound, "sso not configured")
		return
	}
	redirectBack := r.URL.Query().Get("redirect_uri")
	authURL, state, err := h.SSOManager.Initiate(r.Context(), redirectBack)
	if err != nil {
		writeErr(w, stdhttp.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]string{
		"authorize_url": authURL,
		"state":         state,
	})
}

func (h *Handler) handleSSOCallback(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.SSOManager == nil {
		writeErr(w, stdhttp.StatusNotFound, "sso not configured")
		return
	}
	q := r.URL.Query()
	params := sso.CallbackParams{
		Code:         q.Get("code"),
		State:        q.Get("state"),
		SAMLResponse: q.Get("SAMLResponse"),
		RelayState:   q.Get("RelayState"),
	}
	if r.Method == stdhttp.MethodPost {
		_ = r.ParseForm()
		if v := r.PostFormValue("SAMLResponse"); v != "" {
			params.SAMLResponse = v
		}
		if v := r.PostFormValue("RelayState"); v != "" {
			params.RelayState = v
		}
	}
	result, err := h.SSOManager.Callback(r.Context(), params, clientIP(r))
	if err != nil {
		writeErr(w, stdhttp.StatusUnauthorized, err.Error())
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]interface{}{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
		"user": map[string]interface{}{
			"id":    result.User.ID,
			"email": result.User.Email,
			"roles": result.User.Roles,
		},
	})
}

func (h *Handler) handleTestSSO(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodPost {
		writeErr(w, stdhttp.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.SSOManager == nil {
		writeErr(w, stdhttp.StatusNotFound, "sso not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.SSOManager.TestConnection(ctx); err != nil {
		writeJSON(w, stdhttp.StatusBadGateway, map[string]interface{}{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]interface{}{
		"ok":   true,
		"kind": string(h.SSOManager.Kind()),
	})
}

type impersonateRequest struct {
	Token string `json:"token"`
}

func (h *Handler) handleImpersonate(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	// Accept GET (browser redirect per plan/CONTRACTS.md admin_auth.redirect_url)
	// and POST (API/backchannel). Both flows carry the token via ?token=
	// query param; POST additionally accepts {"token":...} in the body.
	if r.Method != stdhttp.MethodPost && r.Method != stdhttp.MethodGet {
		writeErr(w, stdhttp.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.ImpersonationValidator == nil {
		writeErr(w, stdhttp.StatusNotFound, "impersonation not configured")
		return
	}
	token := r.URL.Query().Get("token")
	if token == "" && r.Method == stdhttp.MethodPost {
		var body impersonateRequest
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
			token = body.Token
		}
	}
	res, err := h.ImpersonationValidator.Validate(r.Context(), token, clientIP(r))
	if err != nil {
		writeErr(w, impersonationStatus(err), err.Error())
		return
	}

	// Option-A: browser redirect (GET) — return 302 to the provider FE landing
	// page with the tenant JWT in the URL fragment so it never hits server logs.
	if r.Method == stdhttp.MethodGet {
		base := h.LandingURLTemplate
		if base == "" {
			base = "http://teach.slate.local/auth/impersonate-landing"
		}
		landingURL := fmt.Sprintf("%s#token=%s&expires_at=%d&tenant_slug=%s",
			base, res.AccessToken, res.ExpiresIn, res.TenantSlug)
		stdhttp.Redirect(w, r, landingURL, stdhttp.StatusFound)
		return
	}

	// POST / backchannel — return JSON as before.
	writeJSON(w, stdhttp.StatusOK, map[string]interface{}{
		"access_token":     res.AccessToken,
		"expires_in":       res.ExpiresIn,
		"impersonation_id": res.ImpersonationID,
		"target_user_id":   res.TargetUserID,
		"actor_id":         res.ActorID,
	})
}

func (h *Handler) handleImpersonateByID(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodDelete {
		writeErr(w, stdhttp.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.ImpersonationValidator == nil {
		writeErr(w, stdhttp.StatusNotFound, "impersonation not configured")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/auth/impersonate/")
	if id == "" {
		writeErr(w, stdhttp.StatusBadRequest, "impersonation_id is required")
		return
	}
	actor := r.Header.Get("X-Actor-Id")
	if err := h.ImpersonationValidator.Revoke(r.Context(), id, actor, clientIP(r)); err != nil {
		writeErr(w, stdhttp.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(stdhttp.StatusNoContent)
}

type mfaResetRequest struct {
	UserID        string `json:"user_id"`
	PlatformToken string `json:"platform_token"`
}

func (h *Handler) handleMFAReset(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodPost {
		writeErr(w, stdhttp.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.MFAReset == nil {
		writeErr(w, stdhttp.StatusNotFound, "mfa reset not configured")
		return
	}
	var req mfaResetRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	token := req.PlatformToken
	if token == "" {
		token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	if err := h.MFAReset.Reset(r.Context(), token, req.UserID, clientIP(r)); err != nil {
		writeErr(w, mfaResetStatus(err), err.Error())
		return
	}
	w.WriteHeader(stdhttp.StatusNoContent)
}

func impersonationStatus(err error) int {
	switch err {
	case auth.ErrImpersonationTokenMissing:
		return stdhttp.StatusBadRequest
	case auth.ErrImpersonationWrongTenant, auth.ErrImpersonationWrongType, auth.ErrImpersonationRevoked:
		return stdhttp.StatusForbidden
	case auth.ErrImpersonationTokenExpired:
		return stdhttp.StatusUnauthorized
	case auth.ErrImpersonationTokenInvalid:
		return stdhttp.StatusUnauthorized
	default:
		return stdhttp.StatusInternalServerError
	}
}

func mfaResetStatus(err error) int {
	switch err {
	case auth.ErrMFAResetTokenMissing, auth.ErrMFAResetUserRequired:
		return stdhttp.StatusBadRequest
	case auth.ErrMFAResetNotPlatform:
		return stdhttp.StatusForbidden
	case auth.ErrMFAResetTokenInvalid:
		return stdhttp.StatusUnauthorized
	default:
		return stdhttp.StatusInternalServerError
	}
}

func writeJSON(w stdhttp.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w stdhttp.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func clientIP(r *stdhttp.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if r.RemoteAddr != "" {
		if i := strings.LastIndex(r.RemoteAddr, ":"); i > 0 {
			return r.RemoteAddr[:i]
		}
		return r.RemoteAddr
	}
	return ""
}

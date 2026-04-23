package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"slate/services/email-service/internal/email"
	"slate/services/email-service/internal/tenants"
)

func newBroadcastReq(t *testing.T, body string, platformAdmin bool) *http.Request {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/broadcast", bytes.NewBufferString(body))
	req.Header.Set("X-User-ID", "admin-1")
	if platformAdmin {
		req.Header.Set("X-Platform-Admin", "true")
	}
	return req
}

func TestBroadcast_RequiresAuthUser(t *testing.T) {
	h := NewBroadcastHandler(nil, &email.RecordingSender{}, tenants.NewStaticResolver(nil))
	req := httptest.NewRequest("POST", "/api/broadcast", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	h.CreateBroadcast(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestBroadcast_NonAdminReturns403(t *testing.T) {
	h := NewBroadcastHandler(nil, &email.RecordingSender{}, tenants.NewStaticResolver(nil))
	body := `{"targets":{"all_tenants":true},"message":"hi","channels":["in_app"]}`
	req := newBroadcastReq(t, body, false) // no platform admin header
	w := httptest.NewRecorder()
	h.CreateBroadcast(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestBroadcast_RoleHeaderGrantsAccess(t *testing.T) {
	sender := &email.RecordingSender{}
	resolver := tenants.NewStaticResolver([]tenants.Admin{
		{TenantID: "t-1", Email: "admin-t1@example.com", UserID: "u1"},
	})
	h := NewBroadcastHandler(nil, sender, resolver)
	body := `{"targets":{"tenant_ids":["t-1"]},"message":"hi","channels":["email_admins"]}`
	req := httptest.NewRequest("POST", "/api/broadcast", bytes.NewBufferString(body))
	req.Header.Set("X-User-ID", "admin-1")
	req.Header.Set("X-User-Roles", "platform_admin, viewer")
	w := httptest.NewRecorder()
	h.CreateBroadcast(w, req)
	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202; body=%s", w.Code, w.Body.String())
	}
	if len(sender.Sent) != 1 {
		t.Fatalf("sent = %d, want 1", len(sender.Sent))
	}
}

func TestBroadcast_MissingMessage(t *testing.T) {
	h := NewBroadcastHandler(nil, &email.RecordingSender{}, tenants.NewStaticResolver(nil))
	body := `{"targets":{"all_tenants":true},"message":"","channels":["in_app"]}`
	req := newBroadcastReq(t, body, true)
	w := httptest.NewRecorder()
	h.CreateBroadcast(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestBroadcast_MutuallyExclusiveTargets(t *testing.T) {
	h := NewBroadcastHandler(nil, &email.RecordingSender{}, tenants.NewStaticResolver(nil))
	body := `{"targets":{"all_tenants":true,"tenant_ids":["t1"]},"message":"x","channels":["in_app"]}`
	req := newBroadcastReq(t, body, true)
	w := httptest.NewRecorder()
	h.CreateBroadcast(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestBroadcast_NoTargets(t *testing.T) {
	h := NewBroadcastHandler(nil, &email.RecordingSender{}, tenants.NewStaticResolver(nil))
	body := `{"targets":{},"message":"x","channels":["in_app"]}`
	req := newBroadcastReq(t, body, true)
	w := httptest.NewRecorder()
	h.CreateBroadcast(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestBroadcast_NoChannels(t *testing.T) {
	h := NewBroadcastHandler(nil, &email.RecordingSender{}, tenants.NewStaticResolver(nil))
	body := `{"targets":{"tenant_ids":["t1"]},"message":"x","channels":[]}`
	req := newBroadcastReq(t, body, true)
	w := httptest.NewRecorder()
	h.CreateBroadcast(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestBroadcast_UnknownChannel(t *testing.T) {
	h := NewBroadcastHandler(nil, &email.RecordingSender{}, tenants.NewStaticResolver(nil))
	body := `{"targets":{"tenant_ids":["t1"]},"message":"x","channels":["sms"]}`
	req := newBroadcastReq(t, body, true)
	w := httptest.NewRecorder()
	h.CreateBroadcast(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestBroadcast_EmailFanOutToMultipleTenants(t *testing.T) {
	sender := &email.RecordingSender{}
	resolver := tenants.NewStaticResolver([]tenants.Admin{
		{TenantID: "tenant-1", Email: "admin1@example.com", UserID: "u1"},
		{TenantID: "tenant-2", Email: "admin2@example.com", UserID: "u2"},
		{TenantID: "tenant-3", Email: "admin3@example.com", UserID: "u3"},
	})
	h := NewBroadcastHandler(nil, sender, resolver)

	body := `{"targets":{"tenant_ids":["tenant-1","tenant-2","tenant-3"]},"message":"Rolling restart","channels":["email_admins"]}`
	req := newBroadcastReq(t, body, true)
	w := httptest.NewRecorder()
	h.CreateBroadcast(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", w.Code, w.Body.String())
	}
	if len(sender.Sent) != 3 {
		t.Fatalf("sent count = %d, want 3", len(sender.Sent))
	}
	seen := map[string]bool{}
	for _, p := range sender.Sent {
		seen[p.To] = true
		if p.Template != "platform_broadcast" {
			t.Errorf("template = %q, want platform_broadcast", p.Template)
		}
		if p.Body != "Rolling restart" {
			t.Errorf("body = %q, want Rolling restart", p.Body)
		}
	}
	for _, want := range []string{"admin1@example.com", "admin2@example.com", "admin3@example.com"} {
		if !seen[want] {
			t.Errorf("missing recipient %s", want)
		}
	}

	var resp BroadcastResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	if resp.EmailsSent != 3 {
		t.Errorf("emails_sent = %d, want 3", resp.EmailsSent)
	}
	if len(resp.TenantIDs) != 3 {
		t.Errorf("tenant_ids len = %d, want 3", len(resp.TenantIDs))
	}
}

func TestBroadcast_AllTenantsFansOut(t *testing.T) {
	sender := &email.RecordingSender{}
	resolver := tenants.NewStaticResolver([]tenants.Admin{
		{TenantID: "a", Email: "a@e.com"},
		{TenantID: "b", Email: "b@e.com"},
	})
	h := NewBroadcastHandler(nil, sender, resolver)

	body := `{"targets":{"all_tenants":true},"message":"ping","channels":["email_admins"]}`
	req := newBroadcastReq(t, body, true)
	w := httptest.NewRecorder()
	h.CreateBroadcast(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d; body=%s", w.Code, w.Body.String())
	}
	if len(sender.Sent) != 2 {
		t.Errorf("sent = %d, want 2", len(sender.Sent))
	}
}

func TestBroadcast_SkipsTenantWithoutAdmin(t *testing.T) {
	sender := &email.RecordingSender{}
	resolver := tenants.NewStaticResolver([]tenants.Admin{
		{TenantID: "known", Email: "k@e.com"},
		// "unknown" has no admin registered
	})
	h := NewBroadcastHandler(nil, sender, resolver)
	body := `{"targets":{"tenant_ids":["known","unknown"]},"message":"x","channels":["email_admins"]}`
	req := newBroadcastReq(t, body, true)
	w := httptest.NewRecorder()
	h.CreateBroadcast(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", w.Code, w.Body.String())
	}
	if len(sender.Sent) != 1 {
		t.Errorf("sent = %d, want 1 (unknown tenant should be skipped)", len(sender.Sent))
	}
}

func TestIsPlatformAdmin(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    bool
	}{
		{"platform header true", map[string]string{"X-Platform-Admin": "true"}, true},
		{"roles has platform_admin", map[string]string{"X-User-Roles": "user, platform_admin"}, true},
		{"roles has super_admin", map[string]string{"X-User-Roles": "super_admin"}, true},
		{"roles unrelated", map[string]string{"X-User-Roles": "viewer,student"}, false},
		{"no headers", map[string]string{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/broadcast", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			if got := isPlatformAdmin(req); got != tc.want {
				t.Errorf("isPlatformAdmin = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV("a, b ,c ")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

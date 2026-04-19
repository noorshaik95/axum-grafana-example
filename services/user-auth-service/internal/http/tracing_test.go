package http

import (
	stdhttp "net/http"
	"testing"
)

// Verifies plan/CONTRACTS.md §trace.propagation: X-Request-ID is generated when
// the caller omits it and echoed on the response.
func TestTracingMiddleware_EchoesRequestID_Generated(t *testing.T) {
	srv, _, _, _ := newTestServer(t)
	defer srv.Close()
	resp, err := stdhttp.Get(srv.URL + "/api/auth/sso/initiate")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	got := resp.Header.Get("X-Request-Id")
	if got == "" {
		t.Fatal("expected generated X-Request-ID on response")
	}
}

// Verifies the middleware honours an inbound X-Request-ID.
func TestTracingMiddleware_EchoesRequestID_Incoming(t *testing.T) {
	srv, _, _, _ := newTestServer(t)
	defer srv.Close()
	req, _ := stdhttp.NewRequest("GET", srv.URL+"/api/auth/sso/initiate", nil)
	req.Header.Set("X-Request-Id", "req-test-42")
	resp, err := stdhttp.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("X-Request-Id"); got != "req-test-42" {
		t.Fatalf("want req-test-42 echoed, got %q", got)
	}
}

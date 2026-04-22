package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"slate/libs/common-go/tracing"

	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware implements the plan/CONTRACTS.md `trace.propagation`
// acceptance gate for this service:
//
//  1. Pull `traceparent` + `X-Request-ID` from inbound headers (generate a
//     request_id if absent, echo on response).
//  2. Stash request_id and tenant slug on the context so downstream handlers
//     and the Kafka producer can include them in outbound spans/headers.
//  3. Tag the active span with `request_id` and `tenant.slug` attributes.
func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := tracing.ExtractTraceparent(r.Context())

		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = r.Header.Get("x-request-id")
		}
		if reqID == "" {
			reqID = randomRequestID()
		}
		ctx = tracing.WithRequestID(ctx, reqID)

		if slug := r.Header.Get("X-Tenant-Slug"); slug != "" {
			ctx = tracing.WithTenantSlug(ctx, slug)
		}

		tracing.TagSpanWithCorrelation(ctx, trace.SpanFromContext(ctx))

		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func randomRequestID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "req-" + hex.EncodeToString(b[:])
}

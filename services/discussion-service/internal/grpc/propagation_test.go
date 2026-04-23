package grpc

import (
	"context"
	"testing"

	commontracing "slate/libs/common-go/tracing"
	pb "slate/services/discussion-service/api/proto"
	"slate/services/discussion-service/internal/kafka"
	"slate/services/discussion-service/internal/userauth"

	"google.golang.org/grpc/metadata"
)

// TestPrepareCtxExtractsCorrelation: prepareCtx pulls x-request-id +
// x-tenant-slug from inbound gRPC metadata and stashes them on ctx.
func TestPrepareCtxExtractsCorrelation(t *testing.T) {
	commontracing.EnsureDefaultPropagator()
	srv := New(Options{Repo: newFakeStore(), TenantSlug: "fallback-slug"})

	md := metadata.New(map[string]string{
		"x-request-id":  "req-abc-123",
		"x-tenant-slug": "acme",
	})
	ctx := srv.prepareCtx(metadata.NewIncomingContext(context.Background(), md))

	if got := commontracing.RequestIDFromContext(ctx); got != "req-abc-123" {
		t.Errorf("request_id = %q, want req-abc-123", got)
	}
	if got := commontracing.TenantSlugFromContext(ctx); got != "acme" {
		t.Errorf("tenant.slug = %q, want acme", got)
	}
}

// TestPrepareCtxFallsBackToEnvTenantSlug: when caller did not send a slug,
// the server uses Options.TenantSlug (i.e. TENANT_SLUG env).
func TestPrepareCtxFallsBackToEnvTenantSlug(t *testing.T) {
	commontracing.EnsureDefaultPropagator()
	srv := New(Options{Repo: newFakeStore(), TenantSlug: "acme"})

	ctx := srv.prepareCtx(context.Background())
	if got := commontracing.TenantSlugFromContext(ctx); got != "acme" {
		t.Errorf("tenant.slug = %q, want acme", got)
	}
	if got := commontracing.RequestIDFromContext(ctx); got == "" {
		t.Errorf("expected synthesized request_id, got empty")
	}
}

// TestKafkaPublishContextCarriesCorrelation: end-to-end through CreatePost,
// the producer receives a context that carries the inbound request_id + slug
// — this is what KafkaHeadersFromContext relies on to attach headers to the
// outbound discussion.mention event.
func TestKafkaPublishContextCarriesCorrelation(t *testing.T) {
	commontracing.EnsureDefaultPropagator()
	pub := &contextCapturingPublisher{}
	srv := New(Options{
		Repo:         newFakeStore(),
		Producer:     pub,
		UserResolver: userauth.StaticResolver{Users: map[string]string{"alice": "user-alice"}},
		TenantSlug:   "acme",
	})

	md := metadata.New(map[string]string{"x-request-id": "req-xyz"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	th, err := srv.CreateThread(ctx, &pb.CreateThreadRequest{
		TenantId: "t1", CourseId: "c1", Title: "t", CreatedBy: "u1",
	})
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	if _, err := srv.CreatePost(ctx, &pb.CreatePostRequest{
		TenantId: "t1", ThreadId: th.Id, AuthorId: "u1", Content: "@alice hi",
	}); err != nil {
		t.Fatalf("CreatePost: %v", err)
	}

	if len(pub.captured) != 1 {
		t.Fatalf("want 1 captured context, got %d", len(pub.captured))
	}
	if got := commontracing.RequestIDFromContext(pub.captured[0]); got != "req-xyz" {
		t.Errorf("producer ctx request_id = %q, want req-xyz", got)
	}
	if got := commontracing.TenantSlugFromContext(pub.captured[0]); got != "acme" {
		t.Errorf("producer ctx tenant.slug = %q, want acme", got)
	}
}

type contextCapturingPublisher struct {
	captured []context.Context
}

func (c *contextCapturingPublisher) PublishMention(ctx context.Context, _ kafka.MentionEvent) error {
	c.captured = append(c.captured, ctx)
	return nil
}

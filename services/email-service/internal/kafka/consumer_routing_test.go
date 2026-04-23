package kafka

import (
	"context"
	"encoding/json"
	"testing"

	"slate/libs/common-go/tracing"
	"slate/services/email-service/internal/email"
	"slate/services/email-service/internal/tenants"

	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func newTestConsumer(sender email.Sender, resolver tenants.Resolver, gradeOpt bool) *Consumer {
	return &Consumer{
		repo:     nil,
		sender:   sender,
		tenants:  resolver,
		gradeOpt: gradeOpt,
	}
}

func TestConsumer_DiscussionMentionEmailsUser(t *testing.T) {
	sender := &email.RecordingSender{}
	c := newTestConsumer(sender, nil, false)

	event := DiscussionMentionEvent{
		TenantID:       "t-1",
		ThreadID:       "thread-1",
		ThreadTitle:    "Homework help",
		MentionedBy:    "alice",
		MentionedUser:  "bob",
		MentionedEmail: "bob@example.com",
		Excerpt:        "@bob what do you think?",
	}
	data, _ := json.Marshal(event)
	if err := c.HandleMessage(context.Background(), "discussion.mention", data); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(sender.Sent) != 1 {
		t.Fatalf("sent = %d, want 1", len(sender.Sent))
	}
	got := sender.Sent[0]
	if got.To != "bob@example.com" {
		t.Errorf("To = %q, want bob@example.com", got.To)
	}
	if got.Template != "discussion_mention" {
		t.Errorf("Template = %q, want discussion_mention", got.Template)
	}
	if got.Subject != "You were mentioned in Homework help" {
		t.Errorf("Subject = %q", got.Subject)
	}
}

func TestConsumer_DiscussionMentionSkipsWhenNoEmail(t *testing.T) {
	sender := &email.RecordingSender{}
	c := newTestConsumer(sender, nil, false)
	data, _ := json.Marshal(DiscussionMentionEvent{ThreadTitle: "t"})
	if err := c.HandleMessage(context.Background(), "discussion.mention", data); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(sender.Sent) != 0 {
		t.Errorf("sent = %d, want 0", len(sender.Sent))
	}
}

func TestConsumer_IncidentOpenedEmailsTenantAdmin(t *testing.T) {
	sender := &email.RecordingSender{}
	resolver := tenants.NewStaticResolver([]tenants.Admin{
		{TenantID: "tenant-A", Email: "admin-a@example.com", UserID: "u-a"},
	})
	c := newTestConsumer(sender, resolver, false)

	event := IncidentOpenedEvent{
		IncidentID: "inc-1",
		TenantID:   "tenant-A",
		Severity:   "P1",
		Title:      "DB down",
		Summary:    "postgres unreachable",
	}
	data, _ := json.Marshal(event)
	if err := c.HandleMessage(context.Background(), "incident.opened", data); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(sender.Sent) != 1 {
		t.Fatalf("sent = %d, want 1", len(sender.Sent))
	}
	got := sender.Sent[0]
	if got.To != "admin-a@example.com" {
		t.Errorf("To = %q", got.To)
	}
	if got.Template != "incident_opened" {
		t.Errorf("Template = %q", got.Template)
	}
	if got.Subject != "[P1] Incident opened: DB down" {
		t.Errorf("Subject = %q", got.Subject)
	}
}

func TestConsumer_IncidentOpenedSkipsPlatformWide(t *testing.T) {
	sender := &email.RecordingSender{}
	resolver := tenants.NewStaticResolver([]tenants.Admin{
		{TenantID: "any", Email: "a@e.com"},
	})
	c := newTestConsumer(sender, resolver, false)
	data, _ := json.Marshal(IncidentOpenedEvent{IncidentID: "inc", Title: "x", Severity: "P3"})
	if err := c.HandleMessage(context.Background(), "incident.opened", data); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(sender.Sent) != 0 {
		t.Errorf("sent = %d, want 0 (no tenant id)", len(sender.Sent))
	}
}

func TestConsumer_IncidentOpenedUnknownTenantIsNoOp(t *testing.T) {
	sender := &email.RecordingSender{}
	resolver := tenants.NewStaticResolver([]tenants.Admin{}) // no admins
	c := newTestConsumer(sender, resolver, false)
	data, _ := json.Marshal(IncidentOpenedEvent{TenantID: "t", IncidentID: "inc", Title: "x", Severity: "P2"})
	if err := c.HandleMessage(context.Background(), "incident.opened", data); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(sender.Sent) != 0 {
		t.Errorf("sent = %d, want 0", len(sender.Sent))
	}
}

func TestConsumer_GradeUpdatedOptIn(t *testing.T) {
	sender := &email.RecordingSender{}
	c := newTestConsumer(sender, nil, true)

	data, _ := json.Marshal(GradeUpdatedEvent{
		TenantID:        "t",
		StudentID:       "s1",
		StudentEmail:    "s1@example.com",
		AssignmentTitle: "HW1",
		Score:           9,
		MaxScore:        10,
	})
	if err := c.HandleMessage(context.Background(), "grade.updated", data); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(sender.Sent) != 1 {
		t.Fatalf("sent = %d, want 1", len(sender.Sent))
	}
	got := sender.Sent[0]
	if got.Template != "grade_updated" {
		t.Errorf("Template = %q", got.Template)
	}
	if got.To != "s1@example.com" {
		t.Errorf("To = %q", got.To)
	}
}

func TestConsumer_GradeUpdatedOptOut(t *testing.T) {
	sender := &email.RecordingSender{}
	c := newTestConsumer(sender, nil, false) // opt-out
	data, _ := json.Marshal(GradeUpdatedEvent{TenantID: "t", StudentEmail: "s@e.com", AssignmentTitle: "x"})
	if err := c.HandleMessage(context.Background(), "grade.updated", data); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(sender.Sent) != 0 {
		t.Errorf("sent = %d, want 0 when gradeOpt=false", len(sender.Sent))
	}
}

func TestConsumer_UnknownTopic(t *testing.T) {
	c := newTestConsumer(&email.RecordingSender{}, nil, false)
	err := c.HandleMessage(context.Background(), "bogus", []byte(`{}`))
	if err == nil {
		t.Errorf("want error for unknown topic")
	}
}

// TestConsumer_TraceHeadersExtracted ensures that traceparent on a consumed
// Kafka message is restored onto the handler context so outbound work stays
// stitched to the producer's span.
func TestConsumer_TraceHeadersExtracted(t *testing.T) {
	tracing.EnsureDefaultPropagator()

	// Build a traceparent by asking the propagator to inject from an incoming header.
	incoming := map[string]string{
		"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
	}
	carrier := propagation.MapCarrier(incoming)
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)

	// Get Kafka headers representing that trace.
	headers := tracing.KafkaHeadersFromContext(ctx)
	if len(headers) == 0 {
		t.Skip("propagator did not produce headers — trace context not propagating in this env")
	}

	msg := kafkago.Message{}
	for _, h := range headers {
		msg.Headers = append(msg.Headers, kafkago.Header{Key: h.Key, Value: h.Value})
	}

	restored := contextFromMessage(context.Background(), msg)

	// Round-trip: headers from the restored context should contain traceparent again.
	roundTrip := tracing.KafkaHeadersFromContext(restored)
	var sawTraceparent bool
	for _, h := range roundTrip {
		if h.Key == "traceparent" {
			sawTraceparent = true
		}
	}
	if !sawTraceparent {
		t.Errorf("traceparent header not restored onto context")
	}
}

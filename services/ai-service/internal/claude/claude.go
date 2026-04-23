// Package claude is the thin wrapper the ai-service uses to reach the
// Anthropic API. Every code-path that calls Claude goes through Invoke —
// this is the single place where:
//
//   1. W7.5 token-budget Check/Record is enforced.
//   2. cache_control is attached to the frozen system prompt, so every
//      request participates in prompt caching.
//   3. the OTel span for the call is tagged with model name and token usage
//      per CONTRACTS.md.
//
// Callers build a Prompt (a stable system prefix + a per-request user
// message) and receive the concatenated text content plus usage stats.
package claude

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	commontracing "slate/libs/common-go/tracing"
	"slate/services/ai-service/internal/budget"
)

const tracerName = "ai-service"

// Invoker is the surface the service handlers depend on. It exists so tests
// can swap in a deterministic stub (see fake_test.go pattern).
type Invoker interface {
	Invoke(ctx context.Context, tenantSlug string, p Prompt) (Result, error)
}

// Prompt is the input shape for a single Claude call.
type Prompt struct {
	Model        string // empty string → DefaultModel on the client
	MaxTokens    int64
	SystemPrompt string // stable prefix — cached
	UserText     string // per-request suffix
	Purpose      string // span attribute: "welcome", "study_plan", ...
}

// Result is what the handlers get back.
type Result struct {
	Text         string
	InputTokens  int64
	OutputTokens int64
	CacheRead    int64
	CacheWrite   int64
	Model        string
}

// Client is the real implementation backed by the Anthropic SDK.
type Client struct {
	sdk          anthropic.Client
	budget       *budget.Tracker
	defaultModel string
}

// NewClient builds a Client. apiKey may be empty in dev mode — callers that
// need to enforce presence should gate at config load time. The SDK will
// pick up ANTHROPIC_API_KEY from env when apiKey is empty.
func NewClient(apiKey, defaultModel string, bt *budget.Tracker) *Client {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &Client{
		sdk:          anthropic.NewClient(opts...),
		budget:       bt,
		defaultModel: defaultModel,
	}
}

// Invoke runs a single Claude request with prompt caching enabled on the
// system prompt. It enforces the monthly token budget, records usage, and
// tags the active span with the standard CONTRACTS.md attributes.
func (c *Client) Invoke(ctx context.Context, tenantSlug string, p Prompt) (Result, error) {
	if err := c.budget.Check(ctx, tenantSlug); err != nil {
		return Result{}, err
	}

	model := p.Model
	if model == "" {
		model = c.defaultModel
	}

	// Propagate tenant slug onto the context so downstream spans / Kafka
	// producers see it even if the gateway didn't inject x-tenant-slug.
	ctx = commontracing.WithTenantSlug(ctx, tenantSlug)

	ctx, span := otel.Tracer(tracerName).Start(ctx, "claude.invoke", trace.WithAttributes(
		attribute.String("ai.provider", "anthropic"),
		attribute.String("ai.purpose", p.Purpose),
		attribute.String("ai.tenant_slug", tenantSlug),
	))
	defer span.End()

	// CONTRACTS.md trace.propagation — stamp request_id + tenant.slug so the
	// Claude span joins back to the originating gateway request in Tempo.
	commontracing.TagSpanWithCorrelation(ctx, span)

	// System prompt is cached — caller guarantees it is byte-stable across
	// requests with the same Purpose. prompt_caching is REQUIRED on every
	// call per the W7 cost-containment gate.
	system := []anthropic.TextBlockParam{
		{
			Type:         "text",
			Text:         p.SystemPrompt,
			CacheControl: anthropic.CacheControlEphemeralParam{Type: "ephemeral"},
		},
	}

	resp, err := c.sdk.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: p.MaxTokens,
		System:    system,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(p.UserText)),
		},
	})
	if err != nil {
		span.RecordError(err)
		return Result{}, fmt.Errorf("claude invoke: %w", err)
	}

	var text string
	for _, block := range resp.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}

	res := Result{
		Text:         text,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		CacheRead:    resp.Usage.CacheReadInputTokens,
		CacheWrite:   resp.Usage.CacheCreationInputTokens,
		Model:        string(resp.Model),
	}

	span.SetAttributes(
		attribute.String("ai.model_name", res.Model),
		attribute.Int64("ai.input_tokens", res.InputTokens),
		attribute.Int64("ai.output_tokens", res.OutputTokens),
		attribute.Int64("ai.cache_read_tokens", res.CacheRead),
		attribute.Int64("ai.cache_write_tokens", res.CacheWrite),
	)

	total := res.InputTokens + res.OutputTokens
	if err := c.budget.Record(ctx, tenantSlug, total); err != nil {
		// Record failure is non-fatal — we've already paid the API cost,
		// and failing closed here would make the response unavailable.
		span.RecordError(err)
	}

	return res, nil
}

// IsBudgetError exposes the budget sentinel so gRPC handlers can map it to
// a specific status code without importing the budget package directly.
func IsBudgetError(err error) bool { return errors.Is(err, budget.ErrBudgetExceeded) }

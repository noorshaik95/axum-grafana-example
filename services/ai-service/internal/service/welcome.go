package service

import (
	"context"
	"errors"
	"strings"

	aipb "slate/services/ai-service/api/proto"
	"slate/services/ai-service/internal/claude"
)

// welcomeSystemPrompt is stable across requests — that's what lets prompt
// caching kick in. Anything tenant- or user-specific goes in UserText.
const welcomeSystemPrompt = `You are a warm, concise LMS welcome bot. ` +
	`Given a student's "today_context" (a short list of upcoming events and ` +
	`due items), return a SINGLE short paragraph (<= 60 words, no lists, ` +
	`no emoji) that greets the student and calls out the one most important ` +
	`thing for today. Do not invent items that are not in the context.`

type Welcomer struct {
	claude       claude.Invoker
	maxTokens    int64
	defaultModel string
}

func NewWelcomer(c claude.Invoker, defaultModel string, maxTokens int) *Welcomer {
	return &Welcomer{claude: c, maxTokens: int64(maxTokens), defaultModel: defaultModel}
}

func (w *Welcomer) Welcome(ctx context.Context, req *aipb.WelcomeMessageRequest) (*aipb.WelcomeMessageResponse, error) {
	if req == nil || req.UserId == "" {
		return nil, errors.New("welcome: user_id required")
	}
	user := "today_context: " + strings.TrimSpace(req.TodayContext)
	result, err := w.claude.Invoke(ctx, req.TenantSlug, claude.Prompt{
		Model:        w.defaultModel,
		MaxTokens:    w.maxTokens,
		SystemPrompt: welcomeSystemPrompt,
		UserText:     user,
		Purpose:      "welcome",
	})
	if err != nil {
		return nil, err
	}
	return &aipb.WelcomeMessageResponse{Text: strings.TrimSpace(result.Text)}, nil
}

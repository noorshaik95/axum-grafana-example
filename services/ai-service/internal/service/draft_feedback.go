package service

import (
	"context"
	"errors"

	aipb "slate/services/ai-service/api/proto"
)

// ErrFlagDisabled is returned when ai_draft_feedback is off for the tenant.
// First-pass implementation per W7.6: the real Claude invocation is stubbed
// behind the flag until we have the rubric evaluation contract nailed down
// — the stub preserves the API surface and lets the frontend wire the flow.
var ErrFlagDisabled = errors.New("ai_draft_feedback flag disabled")

type DraftFeedback struct{}

func NewDraftFeedback() *DraftFeedback { return &DraftFeedback{} }

func (d *DraftFeedback) Evaluate(_ context.Context, req *aipb.DraftFeedbackRequest) (*aipb.DraftFeedbackResponse, error) {
	if req == nil {
		return nil, errors.New("draft_feedback: nil request")
	}
	if !req.FlagEnabled {
		return nil, ErrFlagDisabled
	}
	// Flag is on — placeholder until the rubric-evaluation pipeline lands.
	rows := make([]*aipb.RubricFeedback, 0, len(req.Rubric))
	for _, r := range req.Rubric {
		rows = append(rows, &aipb.RubricFeedback{
			RubricId:        r.Id,
			Feedback:        "AI draft feedback is enabled; evaluation pipeline coming online.",
			SuggestedPoints: r.MaxPoints,
		})
	}
	return &aipb.DraftFeedbackResponse{
		Rows:    rows,
		Overall: "Draft feedback stub — flag is on, awaiting W7.6 pipeline rollout.",
	}, nil
}

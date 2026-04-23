package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	aipb "slate/services/ai-service/api/proto"
	"slate/services/ai-service/internal/cache"
	"slate/services/ai-service/internal/claude"
)

const studyPlanTTL = 24 * time.Hour

const studyPlanSystemPrompt = `You are a study-plan generator for university ` +
	`students. Given a list of assignments with due dates and estimated ` +
	`hours, produce a 16-week plan that groups related work, accounts for ` +
	`due dates, and spreads load evenly. Output STRICT JSON only, matching ` +
	`this schema: {"weeks":[{"week":1,"tasks":[{"day":"mon","duration_minutes":60,"title":"..."}]}]}. ` +
	`Days are lowercase three-letter abbreviations (mon..sun). Do not ` +
	`include any text outside the JSON object.`

type StudyPlanner struct {
	claude     claude.Invoker
	cache      *cache.Cache
	maxTokens  int64
	model      string // Opus for study plans per W7.3
}

func NewStudyPlanner(c claude.Invoker, cc *cache.Cache, model string, maxTokens int) *StudyPlanner {
	return &StudyPlanner{claude: c, cache: cc, maxTokens: int64(maxTokens), model: model}
}

func (s *StudyPlanner) Generate(ctx context.Context, req *aipb.StudyPlanRequest) (*aipb.StudyPlanResponse, error) {
	if req == nil || req.UserId == "" {
		return nil, errors.New("study plan: user_id required")
	}

	key := cache.StudyPlanKey(req.TenantSlug, req.UserId)
	var cached aipb.StudyPlanResponse
	if hit, err := s.cache.GetJSON(ctx, key, &cached); err == nil && hit {
		cached.FromCache = true
		return &cached, nil
	}

	userPayload := struct {
		University string                `json:"university"`
		Today      int64                 `json:"today_unix"`
		Assignments []*aipb.AssignmentSummary `json:"assignments"`
	}{
		University:  req.UniversityName,
		Today:       req.TodayUnix,
		Assignments: req.Assignments,
	}
	userJSON, err := json.Marshal(userPayload)
	if err != nil {
		return nil, fmt.Errorf("study plan marshal: %w", err)
	}

	result, err := s.claude.Invoke(ctx, req.TenantSlug, claude.Prompt{
		Model:        s.model,
		MaxTokens:    s.maxTokens,
		SystemPrompt: studyPlanSystemPrompt,
		UserText:     string(userJSON),
		Purpose:      "study_plan",
	})
	if err != nil {
		return nil, err
	}

	weeks, err := parseStudyPlan(result.Text)
	if err != nil {
		return nil, fmt.Errorf("study plan parse: %w", err)
	}
	resp := &aipb.StudyPlanResponse{Weeks: weeks, ModelUsed: result.Model, FromCache: false}
	_ = s.cache.SetJSON(ctx, key, resp, studyPlanTTL)
	return resp, nil
}

// parseStudyPlan is tolerant of Claude wrapping JSON in code fences.
func parseStudyPlan(raw string) ([]*aipb.StudyPlanWeek, error) {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "{"); i > 0 {
		raw = raw[i:]
	}
	if j := strings.LastIndex(raw, "}"); j >= 0 && j < len(raw)-1 {
		raw = raw[:j+1]
	}
	var wire struct {
		Weeks []struct {
			Week  int32 `json:"week"`
			Tasks []struct {
				Day             string `json:"day"`
				DurationMinutes int32  `json:"duration_minutes"`
				Title           string `json:"title"`
			} `json:"tasks"`
		} `json:"weeks"`
	}
	if err := json.Unmarshal([]byte(raw), &wire); err != nil {
		return nil, err
	}
	weeks := make([]*aipb.StudyPlanWeek, 0, len(wire.Weeks))
	for _, w := range wire.Weeks {
		tasks := make([]*aipb.StudyPlanTask, 0, len(w.Tasks))
		for _, t := range w.Tasks {
			tasks = append(tasks, &aipb.StudyPlanTask{
				Day: t.Day, DurationMinutes: t.DurationMinutes, Title: t.Title,
			})
		}
		weeks = append(weeks, &aipb.StudyPlanWeek{Week: w.Week, Tasks: tasks})
	}
	return weeks, nil
}

// InvalidateStudyPlan clears the plan cache on assignment.created so the
// next request regenerates — per W7.3.
func (s *StudyPlanner) InvalidateStudyPlan(ctx context.Context, slug, userID string) error {
	return s.cache.Del(ctx, cache.StudyPlanKey(slug, userID))
}

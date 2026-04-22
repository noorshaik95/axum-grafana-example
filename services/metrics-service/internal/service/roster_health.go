package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"slate/services/metrics-service/internal/cache"
)

// Risk levels returned by the roster health classifier.
const (
	RiskHealthy  = "healthy"
	RiskSlipping = "slipping"
	RiskAtRisk   = "at_risk"
)

// RosterCacheTTL is the cache TTL for roster health results.
const RosterCacheTTL = 2 * time.Minute

// RosterEntry is the per-student record returned by GetRosterHealth.
type RosterEntry struct {
	UserID            string    `json:"user_id"`
	DisplayName       string    `json:"display_name"`
	RiskLevel         string    `json:"risk_level"`
	MissedAssignments int       `json:"missed_assignments"`
	DaysSinceActive   int       `json:"days_since_active"`
	GradeTrend        []float64 `json:"grade_trend"`
	SuggestedAction   string    `json:"suggested_action"`
}

// RosterSignals is the raw per-student input the service needs to classify
// risk. It is built by the repository/handler layer and passed into the
// service for stateless processing and caching.
type RosterSignals struct {
	UserID            string
	DisplayName       string
	MissedAssignments int
	DaysSinceActive   int
	GradeTrend        []float64
}

// RosterService classifies student risk for a course and caches the result
// under `tenant:{slug}:roster:{course_id}`.
type RosterService struct {
	cache cache.Cache
	ttl   time.Duration
}

// NewRosterService returns a RosterService backed by the given cache.
func NewRosterService(c cache.Cache) *RosterService {
	return &RosterService{cache: c, ttl: RosterCacheTTL}
}

// RosterCacheKey returns the cache key for a tenant/course pair.
func RosterCacheKey(tenantSlug, courseID string) string {
	return fmt.Sprintf("tenant:%s:roster:%s", tenantSlug, courseID)
}

// Fetcher returns raw signals for a course. Separated from the service so the
// service can be tested without a database.
type Fetcher func(ctx context.Context, tenantSlug, courseID, instructorID string) ([]RosterSignals, error)

// GetRosterHealth returns the roster entries sorted by risk (highest first).
// On cache hit the cached JSON is returned untouched. On miss the fetcher is
// called, each signal is classified, results are sorted and cached.
func (s *RosterService) GetRosterHealth(ctx context.Context, tenantSlug, courseID, instructorID string, fetch Fetcher) ([]RosterEntry, error) {
	key := RosterCacheKey(tenantSlug, courseID)
	if s.cache != nil {
		if raw, ok := s.cache.Get(key); ok {
			var cached []RosterEntry
			if err := json.Unmarshal(raw, &cached); err == nil {
				return cached, nil
			}
		}
	}

	signals, err := fetch(ctx, tenantSlug, courseID, instructorID)
	if err != nil {
		return nil, err
	}

	entries := make([]RosterEntry, 0, len(signals))
	for _, sig := range signals {
		entries = append(entries, classify(sig))
	}
	sortByRisk(entries)

	if s.cache != nil {
		if buf, err := json.Marshal(entries); err == nil {
			s.cache.Set(key, buf, s.ttl)
		}
	}
	return entries, nil
}

// InvalidateRoster drops every cached roster for the given tenant so stale
// data does not survive a grade update.
func (s *RosterService) InvalidateRoster(tenantSlug, courseID string) {
	if s.cache == nil {
		return
	}
	if courseID == "" {
		s.cache.DeletePrefix(fmt.Sprintf("tenant:%s:roster:", tenantSlug))
		return
	}
	s.cache.Delete(RosterCacheKey(tenantSlug, courseID))
}

// classify applies the W12.1 risk algorithm to a single student's signals.
// at_risk: 2+ missed OR grade trend down 3+ weeks OR 5+ days inactive.
// slipping: 1 missed OR grade trend down 2 weeks.
// healthy: otherwise.
func classify(sig RosterSignals) RosterEntry {
	downStreak := trailingDownStreak(sig.GradeTrend)

	level := RiskHealthy
	action := "No action needed"

	switch {
	case sig.MissedAssignments >= 2 || downStreak >= 3 || sig.DaysSinceActive >= 5:
		level = RiskAtRisk
		action = suggestAtRisk(sig, downStreak)
	case sig.MissedAssignments == 1 || downStreak >= 2:
		level = RiskSlipping
		action = suggestSlipping(sig, downStreak)
	}

	return RosterEntry{
		UserID:            sig.UserID,
		DisplayName:       sig.DisplayName,
		RiskLevel:         level,
		MissedAssignments: sig.MissedAssignments,
		DaysSinceActive:   sig.DaysSinceActive,
		GradeTrend:        sig.GradeTrend,
		SuggestedAction:   action,
	}
}

// trailingDownStreak counts the number of consecutive strictly-decreasing
// steps ending at the last grade. A trend of [90, 85, 80] has a streak of 2.
func trailingDownStreak(trend []float64) int {
	if len(trend) < 2 {
		return 0
	}
	streak := 0
	for i := len(trend) - 1; i > 0; i-- {
		if trend[i] < trend[i-1] {
			streak++
			continue
		}
		break
	}
	return streak
}

func suggestAtRisk(sig RosterSignals, down int) string {
	switch {
	case sig.MissedAssignments >= 2:
		return "Reach out: multiple missed assignments"
	case down >= 3:
		return "Schedule 1:1: sustained grade decline"
	case sig.DaysSinceActive >= 5:
		return "Check in: student has been inactive"
	default:
		return "Escalate: student at risk"
	}
}

func suggestSlipping(sig RosterSignals, down int) string {
	if sig.MissedAssignments == 1 {
		return "Send reminder about missed assignment"
	}
	if down >= 2 {
		return "Monitor: grade trend slipping"
	}
	return "Monitor closely"
}

// sortByRisk puts at_risk first, slipping second, healthy last. Ties broken
// by missed-assignments desc, then days-inactive desc, then name asc.
func sortByRisk(entries []RosterEntry) {
	order := map[string]int{RiskAtRisk: 0, RiskSlipping: 1, RiskHealthy: 2}
	sort.SliceStable(entries, func(i, j int) bool {
		oi, oj := order[entries[i].RiskLevel], order[entries[j].RiskLevel]
		if oi != oj {
			return oi < oj
		}
		if entries[i].MissedAssignments != entries[j].MissedAssignments {
			return entries[i].MissedAssignments > entries[j].MissedAssignments
		}
		if entries[i].DaysSinceActive != entries[j].DaysSinceActive {
			return entries[i].DaysSinceActive > entries[j].DaysSinceActive
		}
		return entries[i].DisplayName < entries[j].DisplayName
	})
}

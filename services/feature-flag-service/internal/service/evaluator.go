// Package service — flag evaluator.
//
// Evaluates a set of flags against an EvalContext (tenant + user + roles)
// using the rule types defined in plan W3.2:
//
//   - all        — enabled for everyone
//   - tenant     — enabled when ctx.TenantID is in rule_value.tenant_ids
//   - role       — enabled when any ctx.Role is in rule_value.roles
//   - percentage — enabled when deterministic_hash(user_id) < rule_value.percent
//   - user       — enabled when ctx.UserID is in rule_value.user_ids
//
// A flag's base `state` is the fallback if no rules match. If any rule matches
// the flag is enabled; rules are pure OR semantics.
package service

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"

	"slate/services/feature-flag-service/internal/repository"
)

// EvalContext is the caller identity used to evaluate flag rules.
type EvalContext struct {
	TenantID   string
	TenantSlug string
	UserID     string
	Roles      []string
}

// Evaluator holds no state — it's a pure function bundle. A struct lets tests
// override the hash strategy later if needed.
type Evaluator struct{}

func NewEvaluator() *Evaluator { return &Evaluator{} }

// EvaluateAll returns a map of flag-key → enabled given ctx.
func (e *Evaluator) EvaluateAll(flags []repository.Flag, ctx EvalContext) map[string]bool {
	out := make(map[string]bool, len(flags))
	for _, f := range flags {
		out[f.Key] = e.IsEnabled(f, ctx)
	}
	return out
}

// IsEnabled evaluates a single flag against ctx. Any matching rule wins;
// otherwise the flag's base state is returned.
func (e *Evaluator) IsEnabled(flag repository.Flag, ctx EvalContext) bool {
	for _, rule := range flag.Rules {
		if e.ruleMatches(rule, ctx) {
			return true
		}
	}
	return flag.Enabled
}

func (e *Evaluator) ruleMatches(rule repository.Rule, ctx EvalContext) bool {
	switch rule.RuleType {
	case repository.RuleTypeAll:
		var v struct {
			Enabled *bool `json:"enabled"`
		}
		if err := json.Unmarshal([]byte(rule.RuleValue), &v); err == nil && v.Enabled != nil {
			return *v.Enabled
		}
		// An "all" rule with no explicit enabled field means enabled-for-everyone.
		return true

	case repository.RuleTypeTenant:
		var v struct {
			TenantIDs   []string `json:"tenant_ids"`
			TenantSlugs []string `json:"tenant_slugs"`
		}
		if err := json.Unmarshal([]byte(rule.RuleValue), &v); err != nil {
			return false
		}
		if containsString(v.TenantIDs, ctx.TenantID) {
			return true
		}
		return containsString(v.TenantSlugs, ctx.TenantSlug)

	case repository.RuleTypeRole:
		var v struct {
			Roles []string `json:"roles"`
		}
		if err := json.Unmarshal([]byte(rule.RuleValue), &v); err != nil {
			return false
		}
		for _, r := range ctx.Roles {
			if containsString(v.Roles, r) {
				return true
			}
		}
		return false

	case repository.RuleTypePercentage:
		var v struct {
			Percent int `json:"percent"`
		}
		if err := json.Unmarshal([]byte(rule.RuleValue), &v); err != nil {
			return false
		}
		if v.Percent <= 0 {
			return false
		}
		if v.Percent >= 100 {
			return true
		}
		if ctx.UserID == "" {
			return false
		}
		return stableBucket(ctx.UserID) < uint32(v.Percent)

	case repository.RuleTypeUser:
		var v struct {
			UserIDs []string `json:"user_ids"`
		}
		if err := json.Unmarshal([]byte(rule.RuleValue), &v); err != nil {
			return false
		}
		return containsString(v.UserIDs, ctx.UserID)
	}
	return false
}

// stableBucket returns a deterministic value in [0, 100) derived from user_id.
// Matches the W3.2 spec ("deterministic hash of user_id").
func stableBucket(userID string) uint32 {
	sum := sha1.Sum([]byte(userID))
	return binary.BigEndian.Uint32(sum[:4]) % 100
}

func containsString(haystack []string, needle string) bool {
	if needle == "" {
		return false
	}
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

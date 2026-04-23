package service

import (
	"testing"

	"slate/services/feature-flag-service/internal/repository"
)

func flagWith(key string, base bool, rules ...repository.Rule) repository.Flag {
	return repository.Flag{Key: key, Enabled: base, Rules: rules}
}

func TestEvaluator_AllRule(t *testing.T) {
	e := NewEvaluator()
	cases := []struct {
		name     string
		rule     repository.Rule
		base     bool
		expected bool
	}{
		{"all_no_body", repository.Rule{RuleType: repository.RuleTypeAll, RuleValue: "{}"}, false, true},
		{"all_explicit_true", repository.Rule{RuleType: repository.RuleTypeAll, RuleValue: `{"enabled":true}`}, false, true},
		{"all_explicit_false_falls_back_to_base", repository.Rule{RuleType: repository.RuleTypeAll, RuleValue: `{"enabled":false}`}, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := e.IsEnabled(flagWith("f", tc.base, tc.rule), EvalContext{UserID: "u1"})
			if got != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestEvaluator_TenantRule(t *testing.T) {
	e := NewEvaluator()
	rule := repository.Rule{RuleType: repository.RuleTypeTenant, RuleValue: `{"tenant_ids":["t-1","t-2"],"tenant_slugs":["eastfield"]}`}
	f := flagWith("f", false, rule)

	if !e.IsEnabled(f, EvalContext{TenantID: "t-1"}) {
		t.Fatal("expected enabled for t-1")
	}
	if !e.IsEnabled(f, EvalContext{TenantSlug: "eastfield"}) {
		t.Fatal("expected enabled for slug eastfield")
	}
	if e.IsEnabled(f, EvalContext{TenantID: "t-3"}) {
		t.Fatal("expected disabled for t-3")
	}
}

func TestEvaluator_RoleRule(t *testing.T) {
	e := NewEvaluator()
	rule := repository.Rule{RuleType: repository.RuleTypeRole, RuleValue: `{"roles":["instructor","admin"]}`}
	f := flagWith("f", false, rule)

	if !e.IsEnabled(f, EvalContext{Roles: []string{"student", "instructor"}}) {
		t.Fatal("expected enabled when any role matches")
	}
	if e.IsEnabled(f, EvalContext{Roles: []string{"student"}}) {
		t.Fatal("expected disabled when no role matches")
	}
}

func TestEvaluator_PercentageRule(t *testing.T) {
	e := NewEvaluator()

	zero := repository.Rule{RuleType: repository.RuleTypePercentage, RuleValue: `{"percent":0}`}
	if e.IsEnabled(flagWith("f", false, zero), EvalContext{UserID: "u"}) {
		t.Fatal("percent=0 should never enable")
	}

	hundred := repository.Rule{RuleType: repository.RuleTypePercentage, RuleValue: `{"percent":100}`}
	if !e.IsEnabled(flagWith("f", false, hundred), EvalContext{UserID: "u"}) {
		t.Fatal("percent=100 should always enable")
	}

	fifty := repository.Rule{RuleType: repository.RuleTypePercentage, RuleValue: `{"percent":50}`}
	f := flagWith("f", false, fifty)

	// Determinism: same user → same verdict across calls.
	first := e.IsEnabled(f, EvalContext{UserID: "stable-user-id"})
	for i := 0; i < 5; i++ {
		if e.IsEnabled(f, EvalContext{UserID: "stable-user-id"}) != first {
			t.Fatal("percentage bucketing must be deterministic for the same user_id")
		}
	}

	// Distribution sanity: ~half of 200 users should match a 50% rule.
	enabled := 0
	for i := 0; i < 200; i++ {
		if e.IsEnabled(f, EvalContext{UserID: string(rune('a' + i%26)) + uidSuffix(i)}) {
			enabled++
		}
	}
	if enabled < 60 || enabled > 140 {
		t.Fatalf("percentage=50 bucket count %d outside sane range [60,140]", enabled)
	}
}

func TestEvaluator_UserRule(t *testing.T) {
	e := NewEvaluator()
	rule := repository.Rule{RuleType: repository.RuleTypeUser, RuleValue: `{"user_ids":["u-1","u-2"]}`}
	f := flagWith("f", false, rule)

	if !e.IsEnabled(f, EvalContext{UserID: "u-1"}) {
		t.Fatal("expected enabled for u-1")
	}
	if e.IsEnabled(f, EvalContext{UserID: "u-3"}) {
		t.Fatal("expected disabled for u-3")
	}
}

func TestEvaluator_BaseStateFallback(t *testing.T) {
	e := NewEvaluator()
	if !e.IsEnabled(flagWith("on", true), EvalContext{}) {
		t.Fatal("expected base=true to pass through")
	}
	if e.IsEnabled(flagWith("off", false), EvalContext{}) {
		t.Fatal("expected base=false to pass through")
	}
}

func TestEvaluator_EvaluateAll(t *testing.T) {
	e := NewEvaluator()
	flags := []repository.Flag{
		flagWith("a", true),
		flagWith("b", false),
		flagWith("c", false, repository.Rule{RuleType: repository.RuleTypeUser, RuleValue: `{"user_ids":["u-1"]}`}),
	}
	result := e.EvaluateAll(flags, EvalContext{UserID: "u-1"})
	if !result["a"] || result["b"] || !result["c"] {
		t.Fatalf("unexpected result map: %+v", result)
	}
}

func uidSuffix(n int) string {
	const alphabet = "abcdefghij"
	return string(alphabet[n%len(alphabet)]) + string(alphabet[(n/10)%len(alphabet)])
}

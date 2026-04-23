package cache

import (
	"strings"
	"testing"
)

// TestKeyFormats pins the Redis key shapes required by the W7 spec.
// Downstream tools (billing reconciliation, dashboards) parse these keys,
// so any drift needs to be a conscious break.
func TestKeyFormats(t *testing.T) {
	if got := GradeKey("acme", "u1", "c1"); got != "tenant:acme:grades:u1:c1" {
		t.Fatalf("grade key: %s", got)
	}
	if got := StudyPlanKey("acme", "u1"); got != "tenant:acme:study_plan:u1" {
		t.Fatalf("study plan key: %s", got)
	}
	got := CmdPaletteKey("acme", "u1", "show my assignments")
	if !strings.HasPrefix(got, "tenant:acme:cmd_palette:u1:") {
		t.Fatalf("cmd palette key prefix: %s", got)
	}
	if len(got) != len("tenant:acme:cmd_palette:u1:")+40 {
		t.Fatalf("expected sha1 suffix of 40 hex chars, got %s", got)
	}
}

// TestSHA1HexStable proves hashes are deterministic — important because
// the cache key for the SAME query must collapse to the SAME slot.
func TestSHA1HexStable(t *testing.T) {
	a := SHA1Hex("hello world")
	b := SHA1Hex("hello world")
	if a != b {
		t.Fatalf("hash differs for same input: %s vs %s", a, b)
	}
	c := SHA1Hex("hello world!")
	if a == c {
		t.Fatalf("hash identical for different inputs")
	}
}

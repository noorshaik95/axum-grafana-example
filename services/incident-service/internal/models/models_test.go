package models

import "testing"

func TestIsValidPriority(t *testing.T) {
	valid := []string{"P0", "P1", "P2", "P3", "P4"}
	for _, p := range valid {
		if !IsValidPriority(p) {
			t.Errorf("expected %q to be valid", p)
		}
	}
	invalid := []string{"", "P5", "p1", "SEV1", "low"}
	for _, p := range invalid {
		if IsValidPriority(p) {
			t.Errorf("expected %q to be invalid", p)
		}
	}
}

func TestIsValidStatus(t *testing.T) {
	for _, s := range []string{"open", "watching", "resolved"} {
		if !IsValidStatus(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}
	for _, s := range []string{"", "OPEN", "closed", "pending"} {
		if IsValidStatus(s) {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

func TestIsValidTransition(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{"open", "open", true},
		{"open", "watching", true},
		{"open", "resolved", true},
		{"watching", "open", true},
		{"watching", "resolved", true},
		{"watching", "watching", true},
		{"resolved", "resolved", true},

		// resolved is terminal
		{"resolved", "open", false},
		{"resolved", "watching", false},

		// unknown source
		{"unknown", "open", false},
	}
	for _, c := range cases {
		got := IsValidTransition(c.from, c.to)
		if got != c.want {
			t.Errorf("IsValidTransition(%q,%q) = %v want %v", c.from, c.to, got, c.want)
		}
	}
}

package kafka

import (
	"testing"
)

func TestHandleGradeUpdated_InvokesCallbackWithSlug(t *testing.T) {
	c := &Consumer{}

	var gotSlug, gotCourse string
	c.OnGradeUpdated = func(slug, course string) {
		gotSlug, gotCourse = slug, course
	}

	payload := []byte(`{"tenant_slug":"acme","course_id":"c1"}`)
	if err := c.HandleGradeUpdatedEvent(payload); err != nil {
		t.Fatal(err)
	}
	if gotSlug != "acme" || gotCourse != "c1" {
		t.Fatalf("expected (acme,c1), got (%s,%s)", gotSlug, gotCourse)
	}
}

func TestHandleGradeUpdated_FallsBackToTenantID(t *testing.T) {
	c := &Consumer{}
	var gotSlug string
	c.OnGradeUpdated = func(slug, course string) { gotSlug = slug }

	payload := []byte(`{"tenant_id":"t-uuid","course_id":"c1"}`)
	if err := c.HandleGradeUpdatedEvent(payload); err != nil {
		t.Fatal(err)
	}
	if gotSlug != "t-uuid" {
		t.Fatalf("expected fallback to tenant_id, got %s", gotSlug)
	}
}

func TestHandleGradeUpdated_InvalidJSON(t *testing.T) {
	c := &Consumer{}
	if err := c.HandleGradeUpdatedEvent([]byte("not-json")); err == nil {
		t.Fatal("expected error")
	}
}

func TestHandleGradeUpdated_NilCallbackNoPanic(t *testing.T) {
	c := &Consumer{}
	payload := []byte(`{"tenant_slug":"acme","course_id":"c1"}`)
	if err := c.HandleGradeUpdatedEvent(payload); err != nil {
		t.Fatal(err)
	}
}

package kafka

import (
	"encoding/json"
	"testing"
)

func TestAssignmentGradedEventUnmarshal(t *testing.T) {
	data := `{
		"tenant_id": "tenant-1",
		"student_id": "student-1",
		"assignment_title": "Homework 1",
		"score": 85.5,
		"max_score": 100.0,
		"feedback": "Good work!"
	}`

	var event AssignmentGradedEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if event.TenantID != "tenant-1" {
		t.Errorf("TenantID = %q, want %q", event.TenantID, "tenant-1")
	}
	if event.StudentID != "student-1" {
		t.Errorf("StudentID = %q, want %q", event.StudentID, "student-1")
	}
	if event.Score != 85.5 {
		t.Errorf("Score = %f, want %f", event.Score, 85.5)
	}
	if event.MaxScore != 100.0 {
		t.Errorf("MaxScore = %f, want %f", event.MaxScore, 100.0)
	}
}

func TestAnnouncementPostedEventUnmarshal(t *testing.T) {
	data := `{
		"tenant_id": "tenant-1",
		"instructor_id": "instructor-1",
		"title": "New Announcement",
		"body": "Important update",
		"enrolled_student_ids": ["s1", "s2", "s3"]
	}`

	var event AnnouncementPostedEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if event.InstructorID != "instructor-1" {
		t.Errorf("InstructorID = %q, want %q", event.InstructorID, "instructor-1")
	}
	if len(event.EnrolledStudentIDs) != 3 {
		t.Errorf("EnrolledStudentIDs length = %d, want 3", len(event.EnrolledStudentIDs))
	}
}

func TestRoomCreatedEventUnmarshal(t *testing.T) {
	data := `{
		"tenant_id": "tenant-1",
		"room_name": "CS101 Lecture",
		"instructor_id": "instructor-1",
		"invited_student_ids": ["s1", "s2"],
		"start_time": "2026-04-17T10:00:00Z"
	}`

	var event RoomCreatedEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if event.RoomName != "CS101 Lecture" {
		t.Errorf("RoomName = %q, want %q", event.RoomName, "CS101 Lecture")
	}
	if len(event.InvitedStudentIDs) != 2 {
		t.Errorf("InvitedStudentIDs length = %d, want 2", len(event.InvitedStudentIDs))
	}
}

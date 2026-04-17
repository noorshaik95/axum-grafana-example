package models

import (
	"encoding/json"
	"testing"
)

func TestUserDataPayload_ToJSON(t *testing.T) {
	tests := []struct {
		name    string
		payload UserDataPayload
		wantErr bool
	}{
		{
			name: "valid user data",
			payload: UserDataPayload{
				Email:       "john.doe@university.edu",
				FirstName:   "John",
				LastName:    "Doe",
				Role:        RoleStudent,
				CourseCodes: []string{"CS101", "CS102"},
				CustomFields: map[string]string{
					"student_id": "S12345",
				},
			},
			wantErr: false,
		},
		{
			name: "minimal user data",
			payload: UserDataPayload{
				Email:     "jane@test.com",
				FirstName: "Jane",
				LastName:  "Smith",
				Role:      RoleInstructor,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonStr, err := tt.payload.ToJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify it's valid JSON
				var decoded UserDataPayload
				if err := json.Unmarshal([]byte(jsonStr), &decoded); err != nil {
					t.Errorf("ToJSON() produced invalid JSON: %v", err)
				}

				// Verify key fields
				if decoded.Email != tt.payload.Email {
					t.Errorf("Email mismatch: got %v, want %v", decoded.Email, tt.payload.Email)
				}
				if decoded.Role != tt.payload.Role {
					t.Errorf("Role mismatch: got %v, want %v", decoded.Role, tt.payload.Role)
				}
			}
		})
	}
}

func TestUserDataPayload_FromJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		check   func(*testing.T, *UserDataPayload)
	}{
		{
			name: "valid JSON",
			json: `{
				"email": "test@university.edu",
				"first_name": "Test",
				"last_name": "User",
				"role": "student",
				"course_codes": ["CS101"]
			}`,
			wantErr: false,
			check: func(t *testing.T, p *UserDataPayload) {
				if p.Email != "test@university.edu" {
					t.Errorf("Email = %v, want test@university.edu", p.Email)
				}
				if p.Role != RoleStudent {
					t.Errorf("Role = %v, want %v", p.Role, RoleStudent)
				}
				if len(p.CourseCodes) != 1 || p.CourseCodes[0] != "CS101" {
					t.Errorf("CourseCodes = %v, want [CS101]", p.CourseCodes)
				}
			},
		},
		{
			name:    "invalid JSON",
			json:    `{invalid}`,
			wantErr: true,
		},
		{
			name:    "empty JSON object",
			json:    `{}`,
			wantErr: false,
			check: func(t *testing.T, p *UserDataPayload) {
				if p.Email != "" {
					t.Errorf("Email should be empty, got %v", p.Email)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var payload UserDataPayload
			err := payload.FromJSON(tt.json)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, &payload)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Test job statuses
	jobStatuses := []string{
		JobStatusPending,
		JobStatusProcessing,
		JobStatusCompleted,
		JobStatusFailed,
		JobStatusCancelled,
	}

	for _, status := range jobStatuses {
		if status == "" {
			t.Error("Job status constant is empty")
		}
	}

	// Test task statuses
	taskStatuses := []string{
		TaskStatusPending,
		TaskStatusProcessing,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusSkipped,
	}

	for _, status := range taskStatuses {
		if status == "" {
			t.Error("Task status constant is empty")
		}
	}

	// Test source types
	sourceTypes := []string{
		SourceTypeCSV,
		SourceTypeLDAP,
		SourceTypeSAML,
		SourceTypeGoogle,
		SourceTypeMicrosoft,
		SourceTypeAPI,
	}

	for _, sourceType := range sourceTypes {
		if sourceType == "" {
			t.Error("Source type constant is empty")
		}
	}

	// Test roles
	roles := []string{
		RoleStudent,
		RoleInstructor,
		RoleStaff,
		RoleAdmin,
	}

	for _, role := range roles {
		if role == "" {
			t.Error("Role constant is empty")
		}
	}
}

func TestOnboardingTaskMessage(t *testing.T) {
	msg := OnboardingTaskMessage{
		TaskID:   "task-123",
		JobID:    "job-456",
		TenantID: "tenant-789",
		UserData: UserDataPayload{
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
			Role:      RoleStudent,
		},
		Attempt: 1,
	}

	// Verify serialization
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal OnboardingTaskMessage: %v", err)
	}

	// Verify deserialization
	var decoded OnboardingTaskMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal OnboardingTaskMessage: %v", err)
	}

	if decoded.TaskID != msg.TaskID {
		t.Errorf("TaskID mismatch: got %v, want %v", decoded.TaskID, msg.TaskID)
	}
	if decoded.UserData.Email != msg.UserData.Email {
		t.Errorf("Email mismatch: got %v, want %v", decoded.UserData.Email, msg.UserData.Email)
	}
	if decoded.Attempt != msg.Attempt {
		t.Errorf("Attempt mismatch: got %v, want %v", decoded.Attempt, msg.Attempt)
	}
}

func TestProgressUpdateMessage(t *testing.T) {
	msg := ProgressUpdateMessage{
		JobID:              "job-123",
		TenantID:           "tenant-456",
		CurrentStage:       "processing_users",
		ProgressPercentage: 75.5,
		ProcessedCount:     7550,
		TotalCount:         10000,
		SuccessCount:       7500,
		FailedCount:        50,
	}

	// Verify serialization
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal ProgressUpdateMessage: %v", err)
	}

	// Verify deserialization
	var decoded ProgressUpdateMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ProgressUpdateMessage: %v", err)
	}

	if decoded.ProgressPercentage != msg.ProgressPercentage {
		t.Errorf("ProgressPercentage mismatch: got %v, want %v", decoded.ProgressPercentage, msg.ProgressPercentage)
	}
	if decoded.SuccessCount != msg.SuccessCount {
		t.Errorf("SuccessCount mismatch: got %v, want %v", decoded.SuccessCount, msg.SuccessCount)
	}
}

func TestAuditEventMessage(t *testing.T) {
	msg := AuditEventMessage{
		TenantID:  "tenant-123",
		JobID:     "job-456",
		TaskID:    "task-789",
		EventType: EventUserCreated,
		EventData: map[string]interface{}{
			"user_id": "user-001",
			"email":   "test@example.com",
			"role":    RoleStudent,
		},
		PerformedBy: "system",
		IPAddress:   "192.168.1.1",
		UserAgent:   "OnboardingWorker/1.0",
	}

	// Verify serialization
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal AuditEventMessage: %v", err)
	}

	// Verify deserialization
	var decoded AuditEventMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal AuditEventMessage: %v", err)
	}

	if decoded.EventType != msg.EventType {
		t.Errorf("EventType mismatch: got %v, want %v", decoded.EventType, msg.EventType)
	}
	if decoded.EventData["user_id"] != "user-001" {
		t.Errorf("EventData user_id mismatch: got %v, want user-001", decoded.EventData["user_id"])
	}
}

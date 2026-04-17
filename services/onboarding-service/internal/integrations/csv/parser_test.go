package csv

import (
	"strings"
	"testing"

	"github.com/noorshaik95/slate/services/onboarding-service/internal/models"
)

func TestParser_Parse_ValidCSV(t *testing.T) {
	csvData := `email,first_name,last_name,role,department,course_codes,graduation_year
john.doe@university.edu,John,Doe,student,Computer Science,"CS101,CS102",2026
jane.smith@university.edu,Jane,Smith,instructor,Mathematics,"MATH201,MATH202",
bob.wilson@university.edu,Bob,Wilson,staff,Administration,,
`

	parser := NewParser()
	reader := strings.NewReader(csvData)

	result, err := parser.Parse(reader, "job-123", "tenant-456")
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Verify counts
	if result.TotalRows != 3 {
		t.Errorf("TotalRows = %d, want 3", result.TotalRows)
	}
	if result.ValidRows != 3 {
		t.Errorf("ValidRows = %d, want 3", result.ValidRows)
	}
	if result.InvalidRows != 0 {
		t.Errorf("InvalidRows = %d, want 0", result.InvalidRows)
	}
	if len(result.Tasks) != 3 {
		t.Fatalf("len(Tasks) = %d, want 3", len(result.Tasks))
	}

	// Verify first task (student)
	task1 := result.Tasks[0]
	if task1.Email != "john.doe@university.edu" {
		t.Errorf("Task1 Email = %v, want john.doe@university.edu", task1.Email)
	}
	if task1.FirstName != "John" {
		t.Errorf("Task1 FirstName = %v, want John", task1.FirstName)
	}
	if task1.Role != models.RoleStudent {
		t.Errorf("Task1 Role = %v, want %v", task1.Role, models.RoleStudent)
	}
	if task1.Department != "Computer Science" {
		t.Errorf("Task1 Department = %v, want Computer Science", task1.Department)
	}
	if len(task1.CourseCodes) != 2 {
		t.Errorf("Task1 CourseCodes length = %d, want 2", len(task1.CourseCodes))
	}
	if task1.GraduationYear.Valid && task1.GraduationYear.Int32 != 2026 {
		t.Errorf("Task1 GraduationYear = %v, want 2026", task1.GraduationYear.Int32)
	}
	if task1.JobID != "job-123" {
		t.Errorf("Task1 JobID = %v, want job-123", task1.JobID)
	}
	if task1.TenantID != "tenant-456" {
		t.Errorf("Task1 TenantID = %v, want tenant-456", task1.TenantID)
	}
	if task1.Status != models.TaskStatusPending {
		t.Errorf("Task1 Status = %v, want %v", task1.Status, models.TaskStatusPending)
	}

	// Verify second task (instructor)
	task2 := result.Tasks[1]
	if task2.Role != models.RoleInstructor {
		t.Errorf("Task2 Role = %v, want %v", task2.Role, models.RoleInstructor)
	}

	// Verify third task (staff)
	task3 := result.Tasks[2]
	if task3.Role != models.RoleStaff {
		t.Errorf("Task3 Role = %v, want %v", task3.Role, models.RoleStaff)
	}
}

func TestParser_Parse_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name          string
		csvData       string
		expectedError string
	}{
		{
			name:          "missing email header",
			csvData:       "first_name,last_name,role\nJohn,Doe,student\n",
			expectedError: "missing required field: email",
		},
		{
			name:          "missing first_name header",
			csvData:       "email,last_name,role\njohn@test.com,Doe,student\n",
			expectedError: "missing required field: first_name",
		},
		{
			name:          "missing last_name header",
			csvData:       "email,first_name,role\njohn@test.com,John,student\n",
			expectedError: "missing required field: last_name",
		},
		{
			name:          "missing role header",
			csvData:       "email,first_name,last_name\njohn@test.com,John,Doe\n",
			expectedError: "missing required field: role",
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.csvData)
			_, err := parser.Parse(reader, "job-123", "tenant-456")
			if err == nil {
				t.Error("Parse() should have failed but didn't")
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Parse() error = %v, want error containing %v", err, tt.expectedError)
			}
		})
	}
}

func TestParser_Parse_ValidationErrors(t *testing.T) {
	csvData := `email,first_name,last_name,role,graduation_year
,John,Doe,student,2026
invalid-email,Jane,Smith,student,2026
john@test.com,,Doe,student,2026
jane@test.com,Jane,,student,2026
bob@test.com,Bob,Wilson,,2026
alice@test.com,Alice,Johnson,invalid_role,2026
charlie@test.com,Charlie,Brown,student,invalid_year
`

	parser := NewParser()
	reader := strings.NewReader(csvData)

	result, err := parser.Parse(reader, "job-123", "tenant-456")
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// All rows should be invalid
	if result.ValidRows != 0 {
		t.Errorf("ValidRows = %d, want 0", result.ValidRows)
	}
	if result.InvalidRows != 7 {
		t.Errorf("InvalidRows = %d, want 7", result.InvalidRows)
	}

	// Verify validation errors
	if len(result.ValidationErrors) == 0 {
		t.Fatal("Expected validation errors but got none")
	}

	// Check for specific error types
	errorTypes := make(map[string]bool)
	for _, verr := range result.ValidationErrors {
		errorTypes[verr.Field] = true
	}

	expectedErrors := []string{"email", "first_name", "last_name", "role"}
	for _, field := range expectedErrors {
		if !errorTypes[field] {
			t.Errorf("Expected validation error for field %v but didn't find one", field)
		}
	}
}

func TestParser_Parse_ValidRoles(t *testing.T) {
	csvData := `email,first_name,last_name,role
student@test.com,Student,Test,student
instructor@test.com,Instructor,Test,instructor
staff@test.com,Staff,Test,staff
admin@test.com,Admin,Test,admin
mixed@test.com,Mixed,Test,Student
`

	parser := NewParser()
	reader := strings.NewReader(csvData)

	result, err := parser.Parse(reader, "job-123", "tenant-456")
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// All rows should be valid (roles are case-insensitive)
	if result.ValidRows != 5 {
		t.Errorf("ValidRows = %d, want 5", result.ValidRows)
	}

	// Verify roles are normalized
	expectedRoles := []string{
		models.RoleStudent,
		models.RoleInstructor,
		models.RoleStaff,
		models.RoleAdmin,
		models.RoleStudent, // "Student" should be normalized to "student"
	}

	for i, task := range result.Tasks {
		if task.Role != expectedRoles[i] {
			t.Errorf("Task %d Role = %v, want %v", i, task.Role, expectedRoles[i])
		}
	}
}

func TestParser_Parse_CourseCodes(t *testing.T) {
	csvData := `email,first_name,last_name,role,course_codes
john@test.com,John,Doe,student,"CS101,CS102,CS103"
jane@test.com,Jane,Smith,student,MATH201
bob@test.com,Bob,Wilson,student,
`

	parser := NewParser()
	reader := strings.NewReader(csvData)

	result, err := parser.Parse(reader, "job-123", "tenant-456")
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if result.ValidRows != 3 {
		t.Errorf("ValidRows = %d, want 3", result.ValidRows)
	}

	// Task 1: Multiple courses
	if len(result.Tasks[0].CourseCodes) != 3 {
		t.Errorf("Task 1 CourseCodes length = %d, want 3", len(result.Tasks[0].CourseCodes))
	}
	expectedCourses := []string{"CS101", "CS102", "CS103"}
	for i, code := range result.Tasks[0].CourseCodes {
		if code != expectedCourses[i] {
			t.Errorf("Task 1 CourseCodes[%d] = %v, want %v", i, code, expectedCourses[i])
		}
	}

	// Task 2: Single course
	if len(result.Tasks[1].CourseCodes) != 1 {
		t.Errorf("Task 2 CourseCodes length = %d, want 1", len(result.Tasks[1].CourseCodes))
	}

	// Task 3: No courses
	if len(result.Tasks[2].CourseCodes) != 0 {
		t.Errorf("Task 3 CourseCodes length = %d, want 0", len(result.Tasks[2].CourseCodes))
	}
}

func TestParser_Parse_PreferredLanguage(t *testing.T) {
	csvData := `email,first_name,last_name,role,preferred_language
john@test.com,John,Doe,student,en
jane@test.com,Jane,Smith,student,es
bob@test.com,Bob,Wilson,student,
`

	parser := NewParser()
	reader := strings.NewReader(csvData)

	result, err := parser.Parse(reader, "job-123", "tenant-456")
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Task 1: Explicit language
	if result.Tasks[0].PreferredLang != "en" {
		t.Errorf("Task 1 PreferredLang = %v, want en", result.Tasks[0].PreferredLang)
	}

	// Task 2: Different language
	if result.Tasks[1].PreferredLang != "es" {
		t.Errorf("Task 2 PreferredLang = %v, want es", result.Tasks[1].PreferredLang)
	}

	// Task 3: Default to English
	if result.Tasks[2].PreferredLang != "en" {
		t.Errorf("Task 3 PreferredLang = %v, want en (default)", result.Tasks[2].PreferredLang)
	}
}

func TestParser_Parse_EmptyCSV(t *testing.T) {
	csvData := `email,first_name,last_name,role
`

	parser := NewParser()
	reader := strings.NewReader(csvData)

	result, err := parser.Parse(reader, "job-123", "tenant-456")
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if result.TotalRows != 0 {
		t.Errorf("TotalRows = %d, want 0", result.TotalRows)
	}
	if len(result.Tasks) != 0 {
		t.Errorf("len(Tasks) = %d, want 0", len(result.Tasks))
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{"john.doe@university.edu", true},
		{"test@example.com", true},
		{"user+tag@domain.co.uk", true},
		{"invalid", false},
		{"@domain.com", false},
		{"user@", false},
		{"", false},
		{"no-at-sign.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			if got := isValidEmail(tt.email); got != tt.want {
				t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{"student", true},
		{"instructor", true},
		{"staff", true},
		{"admin", true},
		{"Student", true},    // case-insensitive
		{"INSTRUCTOR", true}, // case-insensitive
		{"invalid", false},
		{"teacher", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			if got := isValidRole(tt.role); got != tt.want {
				t.Errorf("isValidRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestParser_Parse_GraduationYear(t *testing.T) {
	csvData := `email,first_name,last_name,role,graduation_year
john@test.com,John,Doe,student,2026
jane@test.com,Jane,Smith,student,
bob@test.com,Bob,Wilson,student,invalid
`

	parser := NewParser()
	reader := strings.NewReader(csvData)

	result, err := parser.Parse(reader, "job-123", "tenant-456")
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Task 1: Valid graduation year
	if !result.Tasks[0].GraduationYear.Valid {
		t.Error("Task 1 GraduationYear should be valid")
	}
	if result.Tasks[0].GraduationYear.Int32 != 2026 {
		t.Errorf("Task 1 GraduationYear = %v, want 2026", result.Tasks[0].GraduationYear.Int32)
	}

	// Task 2: No graduation year
	if result.Tasks[1].GraduationYear.Valid {
		t.Error("Task 2 GraduationYear should not be valid")
	}

	// Task 3: Invalid graduation year should produce validation error
	hasGradYearError := false
	for _, verr := range result.ValidationErrors {
		if verr.Field == "graduation_year" && verr.Row == 3 {
			hasGradYearError = true
			break
		}
	}
	if !hasGradYearError {
		t.Error("Expected graduation_year validation error for row 3")
	}
}

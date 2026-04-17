package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/noorshaik95/slate/services/onboarding-service/internal/models"
)

// ValidationError represents a CSV validation error
type ValidationError struct {
	Row   int
	Field string
	Value string
	Error string
}

// ParseResult contains parsed tasks and any validation errors
type ParseResult struct {
	Tasks            []*models.Task
	ValidationErrors []ValidationError
	TotalRows        int
	ValidRows        int
	InvalidRows      int
}

// Parser handles CSV parsing for bulk user onboarding
type Parser struct {
	requiredFields []string
	optionalFields []string
}

// NewParser creates a new CSV parser
func NewParser() *Parser {
	return &Parser{
		requiredFields: []string{"email", "first_name", "last_name", "role"},
		optionalFields: []string{"student_id", "department", "course_codes", "graduation_year", "phone", "preferred_language"},
	}
}

// Parse parses a CSV file and returns tasks
func (p *Parser) Parse(reader io.Reader, jobID, tenantID string) (*ParseResult, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	// Read header row
	headers, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// Validate headers
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(header))] = i
	}

	// Check required fields
	for _, field := range p.requiredFields {
		if _, ok := headerMap[field]; !ok {
			return nil, fmt.Errorf("missing required field: %s", field)
		}
	}

	result := &ParseResult{
		Tasks:            make([]*models.Task, 0),
		ValidationErrors: make([]ValidationError, 0),
	}

	rowNum := 1 // Start from 1 (after header)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV row %d: %w", rowNum, err)
		}

		result.TotalRows++

		// Parse and validate row
		task, validationErrs := p.parseRow(record, headerMap, jobID, tenantID, rowNum)
		if len(validationErrs) > 0 {
			result.ValidationErrors = append(result.ValidationErrors, validationErrs...)
			result.InvalidRows++
		} else {
			result.Tasks = append(result.Tasks, task)
			result.ValidRows++
		}

		rowNum++
	}

	return result, nil
}

func (p *Parser) parseRow(record []string, headerMap map[string]int, jobID, tenantID string, rowNum int) (*models.Task, []ValidationError) {
	var errors []ValidationError

	// Helper function to get field value
	getField := func(field string) string {
		if idx, ok := headerMap[field]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		return ""
	}

	// Parse required fields
	email := getField("email")
	firstName := getField("first_name")
	lastName := getField("last_name")
	role := getField("role")

	// Validate required fields
	if email == "" {
		errors = append(errors, ValidationError{
			Row:   rowNum,
			Field: "email",
			Value: email,
			Error: "email is required",
		})
	} else if !isValidEmail(email) {
		errors = append(errors, ValidationError{
			Row:   rowNum,
			Field: "email",
			Value: email,
			Error: "invalid email format",
		})
	}

	if firstName == "" {
		errors = append(errors, ValidationError{
			Row:   rowNum,
			Field: "first_name",
			Value: firstName,
			Error: "first_name is required",
		})
	}

	if lastName == "" {
		errors = append(errors, ValidationError{
			Row:   rowNum,
			Field: "last_name",
			Value: lastName,
			Error: "last_name is required",
		})
	}

	if role == "" {
		errors = append(errors, ValidationError{
			Row:   rowNum,
			Field: "role",
			Value: role,
			Error: "role is required",
		})
	} else if !isValidRole(role) {
		errors = append(errors, ValidationError{
			Row:   rowNum,
			Field: "role",
			Value: role,
			Error: "role must be one of: student, instructor, staff, admin",
		})
	}

	// If there are validation errors, return them
	if len(errors) > 0 {
		return nil, errors
	}

	// Parse optional fields
	studentID := getField("student_id")
	department := getField("department")
	courseCodesStr := getField("course_codes")
	graduationYearStr := getField("graduation_year")
	phone := getField("phone")
	preferredLang := getField("preferred_language")

	// Parse course codes
	var courseCodes []string
	if courseCodesStr != "" {
		courseCodes = strings.Split(courseCodesStr, ",")
		for i := range courseCodes {
			courseCodes[i] = strings.TrimSpace(courseCodes[i])
		}
	}

	// Parse graduation year
	var graduationYear int
	if graduationYearStr != "" {
		year, parseErr := strconv.Atoi(graduationYearStr)
		switch {
		case parseErr != nil:
			errors = append(errors, ValidationError{
				Row:   rowNum,
				Field: "graduation_year",
				Value: graduationYearStr,
				Error: "invalid graduation year format",
			})
		case year < 1900 || year > 2200:
			errors = append(errors, ValidationError{
				Row:   rowNum,
				Field: "graduation_year",
				Value: graduationYearStr,
				Error: "graduation year must be between 1900 and 2200",
			})
		default:
			graduationYear = year
		}
	}

	// Default preferred language to English
	if preferredLang == "" {
		preferredLang = "en"
	}

	task := &models.Task{
		JobID:         jobID,
		TenantID:      tenantID,
		Email:         email,
		FirstName:     firstName,
		LastName:      lastName,
		Role:          strings.ToLower(role), // Normalize role to lowercase
		StudentID:     studentID,
		Department:    department,
		CourseCodes:   courseCodes,
		Phone:         phone,
		PreferredLang: preferredLang,
		Status:        models.TaskStatusPending,
	}

	if graduationYear > 0 {
		task.GraduationYear.Valid = true
		// #nosec G115 -- graduationYear is validated to be between 1900-2200, safe for int32
		task.GraduationYear.Int32 = int32(graduationYear)
	}

	return task, errors
}

// Helper functions

func isValidEmail(email string) bool {
	// Simple email validation
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return false
	}
	// Must have something before and after @
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	// Domain part must have at least one character before the dot
	if strings.HasPrefix(parts[1], ".") {
		return false
	}
	return true
}

func isValidRole(role string) bool {
	validRoles := []string{
		models.RoleStudent,
		models.RoleInstructor,
		models.RoleStaff,
		models.RoleAdmin,
	}

	role = strings.ToLower(role)
	for _, validRole := range validRoles {
		if role == validRole {
			return true
		}
	}
	return false
}

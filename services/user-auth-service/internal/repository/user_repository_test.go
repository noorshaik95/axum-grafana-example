package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"slate/services/user-auth-service/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)
	now := time.Now()

	user := &models.User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: "hashed-password",
		FirstName:    "John",
		LastName:     "Doe",
		Phone:        "+1234567890",
		Timezone:     "UTC",
		AvatarURL:    "https://example.com/avatar.jpg",
		Bio:          "Test bio",
		OrganizationID: "org-123",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName,
			user.Phone, user.Timezone, user.AvatarURL, user.Bio, user.OrganizationID,
			user.Username, user.IsActive, user.CreatedAt, user.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(context.Background(), user)
	require.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)
	now := time.Now()

	user := &models.User{
		ID:           "user-123",
		Email:        "existing@example.com",
		PasswordHash: "hashed-password",
		FirstName:    "John",
		LastName:     "Doe",
		Phone:        "+1234567890",
		Timezone:     "UTC",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Simulate unique constraint violation (23505 is PostgreSQL's unique violation code)
	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName,
			user.Phone, user.Timezone, user.AvatarURL, user.Bio, user.OrganizationID,
			user.Username, user.IsActive, user.CreatedAt, user.UpdatedAt).
		WillReturnError(&pq.Error{Code: "23505", Constraint: "users_email_key"})

	err = repo.Create(context.Background(), user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUserRepository_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)
	now := time.Now()

	roles := []string{"user", "admin"}
	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "first_name", "last_name",
		"phone", "timezone", "avatar_url", "bio", "organization_id",
		"is_active", "created_at", "updated_at", "roles"}).
		AddRow("user-123", "test@example.com", "hashed-password", "John", "Doe",
			"+1234567890", "UTC", "https://example.com/avatar.jpg", "Test bio", "org-123",
			true, now, now, pq.Array(roles))

	mock.ExpectQuery("SELECT (.+) FROM users u").
		WithArgs("user-123").
		WillReturnRows(rows)

	user, err := repo.GetByID(context.Background(), "user-123")
	require.NoError(t, err)
	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "John", user.FirstName)
	assert.Equal(t, roles, user.Roles)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectQuery("SELECT (.+) FROM users u").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	user, err := repo.GetByID(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "user not found")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUserRepository_GetByEmail_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)
	now := time.Now()

	roles := []string{"user"}
	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "first_name", "last_name",
		"phone", "timezone", "avatar_url", "bio", "organization_id",
		"is_active", "created_at", "updated_at", "roles"}).
		AddRow("user-123", "test@example.com", "hashed-password", "John", "Doe",
			"+1234567890", "UTC", "", "", "",
			true, now, now, pq.Array(roles))

	mock.ExpectQuery("SELECT (.+) FROM users u").
		WithArgs("test@example.com").
		WillReturnRows(rows)

	user, err := repo.GetByEmail(context.Background(), "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "test@example.com", user.Email)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUserRepository_Update_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)
	now := time.Now()

	user := &models.User{
		ID:        "user-123",
		Email:     "updated@example.com",
		FirstName: "Jane",
		LastName:  "Smith",
		Phone:     "+9876543210",
		Timezone:  "America/New_York",
		IsActive:  false,
		UpdatedAt: now,
	}

	mock.ExpectExec("UPDATE users SET").
		WithArgs(user.Email, user.FirstName, user.LastName, user.Phone, user.Timezone,
			user.AvatarURL, user.Bio, user.OrganizationID, user.IsActive, user.UpdatedAt, user.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Update(context.Background(), user)
	require.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUserRepository_Update_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)
	now := time.Now()

	user := &models.User{
		ID:        "nonexistent",
		Email:     "test@example.com",
		UpdatedAt: now,
	}

	mock.ExpectExec("UPDATE users SET").
		WithArgs(user.Email, user.FirstName, user.LastName, user.Phone, user.Timezone,
			user.AvatarURL, user.Bio, user.OrganizationID, user.IsActive, user.UpdatedAt, user.ID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Update(context.Background(), user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUserRepository_Delete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)

	// Soft delete - sets is_active to false
	mock.ExpectExec("UPDATE users SET is_active = false WHERE id = \\$1").
		WithArgs("user-123").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(context.Background(), "user-123")
	require.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUserRepository_Delete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET is_active = false WHERE id = \\$1").
		WithArgs("nonexistent").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Delete(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUserRepository_UpdatePassword_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET password_hash = \\$1 WHERE id = \\$2").
		WithArgs("new-hashed-password", "user-123").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdatePassword(context.Background(), "user-123", "new-hashed-password")
	require.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}


package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"slate/libs/common-go/tracing"
	"slate/services/user-auth-service/internal/models"

	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) (err error) {
	ctx, span := tracing.StartSpan(ctx, "db.create_user",
		attribute.String("email", user.Email),
		attribute.String("table", "users"))
	defer tracing.EndSpanWithError(span, &err)

	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, phone, timezone, avatar_url, bio, organization_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err = r.db.ExecContext(ctx, query, user.ID, user.Email, user.PasswordHash, user.FirstName,
		user.LastName, user.Phone, user.Timezone, user.AvatarURL, user.Bio, user.OrganizationID, user.IsActive, user.CreatedAt, user.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("user with email %s already exists", user.Email)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT u.id, u.email, u.password_hash, u.first_name, u.last_name, COALESCE(u.phone, '') as phone,
		       COALESCE(u.timezone, 'UTC') as timezone, COALESCE(u.avatar_url, '') as avatar_url,
		       COALESCE(u.bio, '') as bio, COALESCE(u.organization_id, '') as organization_id,
		       u.is_active, u.created_at, u.updated_at,
		       COALESCE(array_agg(ro.name) FILTER (WHERE ro.name IS NOT NULL), '{}') as roles
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles ro ON ur.role_id = ro.id
		WHERE u.id = $1
		GROUP BY u.id
	`

	var roles []string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.Phone, &user.Timezone, &user.AvatarURL, &user.Bio, &user.OrganizationID,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		pq.Array(&roles),
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.Roles = roles
	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (user *models.User, err error) {
	ctx, span := tracing.StartSpan(ctx, "db.get_user_by_email",
		attribute.String("email", email),
		attribute.String("table", "users"))
	defer tracing.EndSpanWithError(span, &err)

	user = &models.User{}
	query := `
		SELECT u.id, u.email, u.password_hash, u.first_name, u.last_name, COALESCE(u.phone, '') as phone,
		       COALESCE(u.timezone, 'UTC') as timezone, COALESCE(u.avatar_url, '') as avatar_url,
		       COALESCE(u.bio, '') as bio, COALESCE(u.organization_id, '') as organization_id,
		       u.is_active, u.created_at, u.updated_at,
		       COALESCE(array_agg(ro.name) FILTER (WHERE ro.name IS NOT NULL), '{}') as roles
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles ro ON ur.role_id = ro.id
		WHERE u.email = $1
		GROUP BY u.id
	`

	var roles []string
	err = r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.Phone, &user.Timezone, &user.AvatarURL, &user.Bio, &user.OrganizationID,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		pq.Array(&roles),
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.Roles = roles
	return user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET email = $1, first_name = $2, last_name = $3, phone = $4,
		    timezone = $5, avatar_url = $6, bio = $7, organization_id = $8,
		    is_active = $9, updated_at = $10
		WHERE id = $11
	`
	result, err := r.db.ExecContext(ctx, query, user.Email, user.FirstName, user.LastName,
		user.Phone, user.Timezone, user.AvatarURL, user.Bio, user.OrganizationID,
		user.IsActive, user.UpdatedAt, user.ID)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// Delete deletes a user (soft delete by setting is_active to false)
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE users SET is_active = false WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)

	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// List retrieves a paginated list of users
func (r *UserRepository) List(ctx context.Context, page, pageSize int, search, role string, isActive *bool) ([]*models.User, int, error) {
	// Build query conditions
	conditions := []string{}
	args := []interface{}{}
	argCount := 1

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(u.email ILIKE $%d OR u.first_name ILIKE $%d OR u.last_name ILIKE $%d)", argCount, argCount, argCount))
		args = append(args, "%"+search+"%")
		argCount++
	}

	if isActive != nil {
		conditions = append(conditions, fmt.Sprintf("u.is_active = $%d", argCount))
		args = append(args, *isActive)
		argCount++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles ro ON ur.role_id = ro.id
		%s
	`, whereClause)

	if role != "" {
		if whereClause != "" {
			countQuery = fmt.Sprintf("%s AND ro.name = $%d", countQuery, argCount)
		} else {
			countQuery = fmt.Sprintf("%s WHERE ro.name = $%d", countQuery, argCount)
		}
		args = append(args, role)
		argCount++
	}

	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get users
	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)

	dataQuery := fmt.Sprintf(`
		SELECT u.id, u.email, u.password_hash, u.first_name, u.last_name, COALESCE(u.phone, '') as phone,
		       COALESCE(u.timezone, 'UTC') as timezone, COALESCE(u.avatar_url, '') as avatar_url,
		       COALESCE(u.bio, '') as bio, COALESCE(u.organization_id, '') as organization_id,
		       u.is_active, u.created_at, u.updated_at,
		       COALESCE(array_agg(ro.name) FILTER (WHERE ro.name IS NOT NULL), '{}') as roles
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles ro ON ur.role_id = ro.id
		%s
		GROUP BY u.id
		ORDER BY u.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount, argCount+1)

	rows, err := r.db.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := []*models.User{}
	for rows.Next() {
		user := &models.User{}
		var roles []string
		err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
			&user.Phone, &user.Timezone, &user.AvatarURL, &user.Bio, &user.OrganizationID,
			&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
			pq.Array(&roles),
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		user.Roles = roles
		users = append(users, user)
	}

	return users, total, nil
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, passwordHash, userID)

	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

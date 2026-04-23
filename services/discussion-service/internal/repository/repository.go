package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"slate/services/discussion-service/internal/models"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// DB returns the underlying *sql.DB so callers can run transactions.
func (r *Repository) DB() *sql.DB { return r.db }

// ---------- threads ----------

func (r *Repository) CreateThread(ctx context.Context, tenantID, courseID, title, createdBy string) (*models.Thread, error) {
	id := uuid.NewString()
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO threads (id, tenant_id, course_id, title, created_by, created_at, last_activity_at, reply_count)
        VALUES ($1,$2,$3,$4,$5,$6,$6,0)`,
		id, tenantID, courseID, title, createdBy, now)
	if err != nil {
		return nil, fmt.Errorf("insert thread: %w", err)
	}
	return &models.Thread{
		ID: id, TenantID: tenantID, CourseID: courseID, Title: title,
		CreatedBy: createdBy, CreatedAt: now, LastActivityAt: now,
	}, nil
}

func (r *Repository) GetThread(ctx context.Context, tenantID, threadID string) (*models.Thread, error) {
	var t models.Thread
	err := r.db.QueryRowContext(ctx, `
        SELECT id, tenant_id, course_id, title, created_by, created_at, last_activity_at, reply_count
        FROM threads WHERE tenant_id=$1 AND id=$2`, tenantID, threadID).
		Scan(&t.ID, &t.TenantID, &t.CourseID, &t.Title, &t.CreatedBy, &t.CreatedAt, &t.LastActivityAt, &t.ReplyCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) ListThreads(ctx context.Context, tenantID, courseID string, limit int, cursor string) ([]models.Thread, string, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	q := `SELECT id, tenant_id, course_id, title, created_by, created_at, last_activity_at, reply_count
          FROM threads WHERE tenant_id=$1 AND course_id=$2`
	args := []any{tenantID, courseID}
	if cursor != "" {
		q += ` AND last_activity_at < (SELECT last_activity_at FROM threads WHERE id=$3)`
		args = append(args, cursor)
	}
	q += fmt.Sprintf(` ORDER BY last_activity_at DESC LIMIT %d`, limit+1)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	var out []models.Thread
	for rows.Next() {
		var t models.Thread
		if err := rows.Scan(&t.ID, &t.TenantID, &t.CourseID, &t.Title, &t.CreatedBy, &t.CreatedAt, &t.LastActivityAt, &t.ReplyCount); err != nil {
			return nil, "", false, err
		}
		out = append(out, t)
	}
	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}
	var next string
	if hasMore && len(out) > 0 {
		next = out[len(out)-1].ID
	}
	return out, next, hasMore, nil
}

// ---------- posts ----------

// CreatePostResult includes the newly inserted post and any mention rows.
type CreatePostResult struct {
	Post     models.Post
	Mentions []models.Mention
	CourseID string
}

// CreatePost inserts a post, materializes mentions + inbox items, and bumps
// the thread's last_activity_at / reply_count inside a single transaction.
func (r *Repository) CreatePost(
	ctx context.Context,
	tenantID, threadID, authorID, authorRole, content string,
	parentPostID *string,
	mentionedUserIDs []string,
) (*CreatePostResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Thread must exist and belong to the tenant; also fetch course_id for the event.
	var courseID string
	err = tx.QueryRowContext(ctx, `SELECT course_id FROM threads WHERE tenant_id=$1 AND id=$2`,
		tenantID, threadID).Scan(&courseID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load thread: %w", err)
	}

	postID := uuid.NewString()
	now := time.Now().UTC()

	_, err = tx.ExecContext(ctx, `
        INSERT INTO posts (id, tenant_id, thread_id, author_id, author_role, content, parent_post_id, created_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		postID, tenantID, threadID, authorID, authorRole, content, parentPostID, now)
	if err != nil {
		return nil, fmt.Errorf("insert post: %w", err)
	}

	var mentions []models.Mention
	for _, uid := range mentionedUserIDs {
		if uid == "" || uid == authorID {
			continue
		}
		mid := uuid.NewString()
		_, err = tx.ExecContext(ctx, `
            INSERT INTO mentions (id, tenant_id, post_id, mentioned_user_id, created_at)
            VALUES ($1,$2,$3,$4,$5)
            ON CONFLICT (post_id, mentioned_user_id) DO NOTHING`,
			mid, tenantID, postID, uid, now)
		if err != nil {
			return nil, fmt.Errorf("insert mention: %w", err)
		}
		_, err = tx.ExecContext(ctx, `
            INSERT INTO inbox_items (id, tenant_id, user_id, type, reference_id, created_at)
            VALUES ($1,$2,$3,'mention',$4,$5)`,
			uuid.NewString(), tenantID, uid, postID, now)
		if err != nil {
			return nil, fmt.Errorf("insert inbox_item: %w", err)
		}
		mentions = append(mentions, models.Mention{
			ID: mid, PostID: postID, MentionedUserID: uid,
		})
	}

	_, err = tx.ExecContext(ctx, `
        UPDATE threads SET last_activity_at=$1, reply_count = reply_count + 1
        WHERE tenant_id=$2 AND id=$3`, now, tenantID, threadID)
	if err != nil {
		return nil, fmt.Errorf("bump thread: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &CreatePostResult{
		Post: models.Post{
			ID: postID, ThreadID: threadID, AuthorID: authorID,
			Content: content, ParentPostID: parentPostID, CreatedAt: now,
		},
		Mentions: mentions,
		CourseID: courseID,
	}, nil
}

func (r *Repository) ListPosts(ctx context.Context, tenantID, threadID string) ([]models.Post, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT id, thread_id, author_id, content, parent_post_id, created_at, edited_at
        FROM posts WHERE tenant_id=$1 AND thread_id=$2 ORDER BY created_at ASC`,
		tenantID, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Post
	for rows.Next() {
		var p models.Post
		if err := rows.Scan(&p.ID, &p.ThreadID, &p.AuthorID, &p.Content, &p.ParentPostID, &p.CreatedAt, &p.EditedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// ---------- inbox ----------

func (r *Repository) GetInbox(ctx context.Context, tenantID, userID string, limit int, cursor string, unseenOnly bool) ([]models.InboxItem, string, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	q := `SELECT id, user_id, type, reference_id, seen_at, created_at
          FROM inbox_items WHERE tenant_id=$1 AND user_id=$2`
	args := []any{tenantID, userID}
	i := 3
	if unseenOnly {
		q += ` AND seen_at IS NULL`
	}
	if cursor != "" {
		q += fmt.Sprintf(` AND created_at < (SELECT created_at FROM inbox_items WHERE id=$%d)`, i)
		args = append(args, cursor)
		i++
	}
	q += fmt.Sprintf(` ORDER BY created_at DESC LIMIT %d`, limit+1)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	var out []models.InboxItem
	for rows.Next() {
		var it models.InboxItem
		if err := rows.Scan(&it.ID, &it.UserID, &it.Type, &it.ReferenceID, &it.SeenAt, &it.CreatedAt); err != nil {
			return nil, "", false, err
		}
		out = append(out, it)
	}
	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}
	var next string
	if hasMore && len(out) > 0 {
		next = out[len(out)-1].ID
	}
	return out, next, hasMore, nil
}

func (r *Repository) MarkInboxRead(ctx context.Context, tenantID, userID, itemID string) error {
	res, err := r.db.ExecContext(ctx, `
        UPDATE inbox_items SET seen_at = NOW()
        WHERE tenant_id=$1 AND user_id=$2 AND id=$3 AND seen_at IS NULL`,
		tenantID, userID, itemID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// Either already seen or not owned by caller; verify existence to
		// distinguish. Idempotent for the "already seen" case.
		var exists bool
		if err := r.db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM inbox_items WHERE tenant_id=$1 AND user_id=$2 AND id=$3)`,
			tenantID, userID, itemID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
	}
	return nil
}

// ---------- instructor queue ----------

// InstructorCoursesFor returns course IDs taught by the instructor within the
// tenant. Callers may pass a pre-filtered list; we still intersect with what
// this service knows locally (instructor_courses is eventually-consistent from
// the course-service via Kafka).
func (r *Repository) InstructorCoursesFor(ctx context.Context, tenantID, instructorID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT course_id FROM instructor_courses WHERE tenant_id=$1 AND instructor_id=$2`,
		tenantID, instructorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// UpsertInstructorCourse is exposed for tests and for the (future) Kafka
// consumer that mirrors course roster events from course-service.
func (r *Repository) UpsertInstructorCourse(ctx context.Context, tenantID, instructorID, courseID string) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO instructor_courses (tenant_id, instructor_id, course_id)
        VALUES ($1,$2,$3)
        ON CONFLICT (tenant_id, instructor_id, course_id) DO NOTHING`,
		tenantID, instructorID, courseID)
	return err
}

// ListThreadsNeedingReply returns threads in the given courses where:
//   - no instructor post in the last 24h, AND
//   - the thread has at least one student post newer than the latest instructor
//     post (or has never had an instructor reply).
//
// Sorted by last_activity_at ASC (oldest-first — instructor's "most overdue").
// now is parameterized for deterministic tests.
func (r *Repository) ListThreadsNeedingReply(ctx context.Context, tenantID string, courseIDs []string, limit int, now time.Time) ([]models.Thread, error) {
	if len(courseIDs) == 0 {
		return nil, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := `
    WITH last_instr AS (
        SELECT thread_id, MAX(created_at) AS last_at
        FROM posts
        WHERE tenant_id=$1 AND author_role='instructor'
        GROUP BY thread_id
    ),
    last_student AS (
        SELECT thread_id, MAX(created_at) AS last_at
        FROM posts
        WHERE tenant_id=$1 AND author_role='student'
        GROUP BY thread_id
    )
    SELECT t.id, t.tenant_id, t.course_id, t.title, t.created_by,
           t.created_at, t.last_activity_at, t.reply_count
    FROM threads t
    LEFT JOIN last_instr li ON li.thread_id = t.id
    LEFT JOIN last_student ls ON ls.thread_id = t.id
    WHERE t.tenant_id = $1
      AND t.course_id = ANY($2)
      AND (li.last_at IS NULL OR li.last_at < $3)
      AND ls.last_at IS NOT NULL
      AND (li.last_at IS NULL OR ls.last_at > li.last_at)
    ORDER BY t.last_activity_at ASC
    LIMIT $4`

	cutoff := now.Add(-24 * time.Hour)
	rows, err := r.db.QueryContext(ctx, q, tenantID, pqStringArray(courseIDs), cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Thread
	for rows.Next() {
		var t models.Thread
		if err := rows.Scan(&t.ID, &t.TenantID, &t.CourseID, &t.Title, &t.CreatedBy,
			&t.CreatedAt, &t.LastActivityAt, &t.ReplyCount); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

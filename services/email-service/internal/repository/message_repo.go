package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"slate/services/email-service/internal/models"

	"github.com/google/uuid"
)

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// SendMessage creates a new message and its recipient records.
// For the first message in a thread, thread_id = message.id.
func (r *MessageRepository) SendMessage(ctx context.Context, tenantID, fromUserID, subject, body string, recipientIDs []string) (*models.Message, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	msgID := uuid.New().String()
	threadID := msgID // first message: thread_id = message id

	var subjectPtr *string
	if subject != "" {
		subjectPtr = &subject
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO messages (id, tenant_id, thread_id, from_user_id, subject, body)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		msgID, tenantID, threadID, fromUserID, subjectPtr, body,
	)
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	for _, recipientID := range recipientIDs {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO message_recipients (id, message_id, recipient_user_id, tenant_id)
			 VALUES ($1, $2, $3, $4)`,
			uuid.New().String(), msgID, recipientID, tenantID,
		)
		if err != nil {
			return nil, fmt.Errorf("insert recipient: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &models.Message{
		ID:         msgID,
		TenantID:   tenantID,
		ThreadID:   threadID,
		FromUserID: fromUserID,
		Subject:    subjectPtr,
		Body:       body,
		CreatedAt:  time.Now(),
	}, nil
}

// ReplyToMessage creates a reply to an existing message within the same thread.
func (r *MessageRepository) ReplyToMessage(ctx context.Context, parentMsgID, fromUserID, body string) (*models.Message, error) {
	// Look up parent message to get thread_id and tenant_id
	var threadID, tenantID string
	err := r.db.QueryRowContext(ctx,
		`SELECT thread_id, tenant_id FROM messages WHERE id = $1`, parentMsgID,
	).Scan(&threadID, &tenantID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("parent message not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query parent: %w", err)
	}

	// Get recipients of original message + the sender (to notify everyone in thread)
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT recipient_user_id FROM message_recipients WHERE message_id = $1
		 UNION
		 SELECT from_user_id FROM messages WHERE id = $1`,
		parentMsgID,
	)
	if err != nil {
		return nil, fmt.Errorf("query recipients: %w", err)
	}
	defer rows.Close()

	var recipientIDs []string
	for rows.Next() {
		var rid string
		if err := rows.Scan(&rid); err != nil {
			return nil, fmt.Errorf("scan recipient: %w", err)
		}
		// Don't add the sender as a recipient of their own reply
		if rid != fromUserID {
			recipientIDs = append(recipientIDs, rid)
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	msgID := uuid.New().String()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO messages (id, tenant_id, thread_id, parent_id, from_user_id, body)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		msgID, tenantID, threadID, parentMsgID, fromUserID, body,
	)
	if err != nil {
		return nil, fmt.Errorf("insert reply: %w", err)
	}

	for _, recipientID := range recipientIDs {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO message_recipients (id, message_id, recipient_user_id, tenant_id)
			 VALUES ($1, $2, $3, $4)`,
			uuid.New().String(), msgID, recipientID, tenantID,
		)
		if err != nil {
			return nil, fmt.Errorf("insert recipient: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &models.Message{
		ID:         msgID,
		TenantID:   tenantID,
		ThreadID:   threadID,
		ParentID:   &parentMsgID,
		FromUserID: fromUserID,
		Body:       body,
		CreatedAt:  time.Now(),
	}, nil
}

// GetInbox returns paginated inbox messages for a user.
func (r *MessageRepository) GetInbox(ctx context.Context, tenantID, userID, cursor string, limit int) (*models.PaginatedResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	query := `
		SELECT m.id, m.tenant_id, m.thread_id, m.from_user_id, m.subject, m.body,
		       mr.is_read, mr.is_archived, mr.read_at, m.created_at
		FROM messages m
		JOIN message_recipients mr ON mr.message_id = m.id
		WHERE mr.recipient_user_id = $1
		  AND mr.tenant_id = $2
		  AND mr.is_archived = false`

	args := []interface{}{userID, tenantID}
	argIdx := 3

	if cursor != "" {
		query += fmt.Sprintf(` AND m.created_at < (SELECT created_at FROM messages WHERE id = $%d)`, argIdx)
		args = append(args, cursor)
		argIdx++
	}

	query += ` ORDER BY m.created_at DESC LIMIT $` + fmt.Sprintf("%d", argIdx)
	args = append(args, limit+1) // fetch one extra to detect hasMore

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query inbox: %w", err)
	}
	defer rows.Close()

	var messages []models.InboxMessage
	for rows.Next() {
		var msg models.InboxMessage
		if err := rows.Scan(
			&msg.ID, &msg.TenantID, &msg.ThreadID, &msg.FromUserID,
			&msg.Subject, &msg.Body, &msg.IsRead, &msg.IsArchived,
			&msg.ReadAt, &msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan inbox: %w", err)
		}
		messages = append(messages, msg)
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	var nextCursor string
	if hasMore && len(messages) > 0 {
		nextCursor = messages[len(messages)-1].ID
	}

	return &models.PaginatedResponse{
		Messages:   messages,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// GetSent returns paginated sent messages for a user.
func (r *MessageRepository) GetSent(ctx context.Context, tenantID, userID, cursor string, limit int) (*models.PaginatedResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	query := `
		SELECT m.id, m.tenant_id, m.thread_id, m.from_user_id, m.subject, m.body,
		       true as is_read, false as is_archived, NULL as read_at, m.created_at
		FROM messages m
		WHERE m.from_user_id = $1
		  AND m.tenant_id = $2
		  AND m.is_deleted_by_sender = false`

	args := []interface{}{userID, tenantID}
	argIdx := 3

	if cursor != "" {
		query += fmt.Sprintf(` AND m.created_at < (SELECT created_at FROM messages WHERE id = $%d)`, argIdx)
		args = append(args, cursor)
		argIdx++
	}

	query += ` ORDER BY m.created_at DESC LIMIT $` + fmt.Sprintf("%d", argIdx)
	args = append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query sent: %w", err)
	}
	defer rows.Close()

	var messages []models.InboxMessage
	for rows.Next() {
		var msg models.InboxMessage
		if err := rows.Scan(
			&msg.ID, &msg.TenantID, &msg.ThreadID, &msg.FromUserID,
			&msg.Subject, &msg.Body, &msg.IsRead, &msg.IsArchived,
			&msg.ReadAt, &msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan sent: %w", err)
		}
		messages = append(messages, msg)
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	var nextCursor string
	if hasMore && len(messages) > 0 {
		nextCursor = messages[len(messages)-1].ID
	}

	return &models.PaginatedResponse{
		Messages:   messages,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// GetMessage returns a single message by ID, with recipients.
func (r *MessageRepository) GetMessage(ctx context.Context, msgID string) (*models.Message, error) {
	var msg models.Message
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, thread_id, parent_id, from_user_id, subject, body,
		        is_deleted_by_sender, created_at
		 FROM messages WHERE id = $1`, msgID,
	).Scan(&msg.ID, &msg.TenantID, &msg.ThreadID, &msg.ParentID,
		&msg.FromUserID, &msg.Subject, &msg.Body, &msg.IsDeletedBySender, &msg.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query message: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, message_id, recipient_user_id, tenant_id, is_read, is_archived, read_at, created_at
		 FROM message_recipients WHERE message_id = $1`, msgID,
	)
	if err != nil {
		return nil, fmt.Errorf("query recipients: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var r models.Recipient
		if err := rows.Scan(&r.ID, &r.MessageID, &r.RecipientUserID, &r.TenantID,
			&r.IsRead, &r.IsArchived, &r.ReadAt, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recipient: %w", err)
		}
		msg.Recipients = append(msg.Recipients, r)
	}

	return &msg, nil
}

// GetThread returns all messages in a thread ordered by created_at.
func (r *MessageRepository) GetThread(ctx context.Context, threadID string) ([]models.Message, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, thread_id, parent_id, from_user_id, subject, body,
		        is_deleted_by_sender, created_at
		 FROM messages WHERE thread_id = $1 ORDER BY created_at ASC`, threadID,
	)
	if err != nil {
		return nil, fmt.Errorf("query thread: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.TenantID, &msg.ThreadID, &msg.ParentID,
			&msg.FromUserID, &msg.Subject, &msg.Body, &msg.IsDeletedBySender, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan thread message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// MarkRead marks a message as read for a specific user.
func (r *MessageRepository) MarkRead(ctx context.Context, msgID, userID string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE message_recipients SET is_read = true, read_at = NOW()
		 WHERE message_id = $1 AND recipient_user_id = $2`,
		msgID, userID,
	)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message not found or not a recipient")
	}
	return nil
}

// MarkUnread marks a message as unread for a specific user.
func (r *MessageRepository) MarkUnread(ctx context.Context, msgID, userID string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE message_recipients SET is_read = false, read_at = NULL
		 WHERE message_id = $1 AND recipient_user_id = $2`,
		msgID, userID,
	)
	if err != nil {
		return fmt.Errorf("mark unread: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message not found or not a recipient")
	}
	return nil
}

// SoftDelete archives a message for the recipient, or marks deleted by sender.
func (r *MessageRepository) SoftDelete(ctx context.Context, msgID, userID string) error {
	// Try recipient archive first
	result, err := r.db.ExecContext(ctx,
		`UPDATE message_recipients SET is_archived = true
		 WHERE message_id = $1 AND recipient_user_id = $2`,
		msgID, userID,
	)
	if err != nil {
		return fmt.Errorf("archive: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows > 0 {
		return nil
	}

	// If not a recipient, try sender delete
	result, err = r.db.ExecContext(ctx,
		`UPDATE messages SET is_deleted_by_sender = true
		 WHERE id = $1 AND from_user_id = $2`,
		msgID, userID,
	)
	if err != nil {
		return fmt.Errorf("sender delete: %w", err)
	}
	rows, _ = result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message not found")
	}
	return nil
}

// GetUnreadCount returns the count of unread messages for a user.
func (r *MessageRepository) GetUnreadCount(ctx context.Context, tenantID, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM message_recipients
		 WHERE tenant_id = $1 AND recipient_user_id = $2 AND is_read = false AND is_archived = false`,
		tenantID, userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread: %w", err)
	}
	return count, nil
}

// UserCanAccessMessage checks if a user is a sender or recipient of a message.
func (r *MessageRepository) UserCanAccessMessage(ctx context.Context, msgID, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM messages WHERE id = $1 AND from_user_id = $2
			UNION
			SELECT 1 FROM message_recipients WHERE message_id = $1 AND recipient_user_id = $2
		)`, msgID, userID,
	).Scan(&exists)
	return exists, err
}

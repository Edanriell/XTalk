package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/yourusername/connect/message-service/domain/entities"
	"github.com/yourusername/connect/message-service/domain/repositories"
	"github.com/yourusername/connect/message-service/domain/valueobjects"
)

// PostgresMessageRepository implements MessageRepository using PostgreSQL
type PostgresMessageRepository struct {
	db *sql.DB
}

// NewPostgresMessageRepository creates a new PostgreSQL message repository
func NewPostgresMessageRepository(db *sql.DB) repositories.MessageRepository {
	return &PostgresMessageRepository{db: db}
}

// Save creates or updates a message
func (r *PostgresMessageRepository) Save(ctx context.Context, message *entities.Message) error {
	metadataJSON, err := json.Marshal(message.Metadata())
	if err != nil {
		return err
	}

	query := `
		INSERT INTO messages (id, chat_id, sender_id, message_type, content, metadata, is_read, created_at, read_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			is_read = EXCLUDED.is_read,
			read_at = EXCLUDED.read_at,
			deleted_at = EXCLUDED.deleted_at
	`

	_, err = r.db.ExecContext(ctx, query,
		message.ID(),
		message.ChatID(),
		message.SenderID(),
		message.MessageType().String(),
		message.Content(),
		metadataJSON,
		message.IsRead(),
		message.CreatedAt(),
		message.ReadAt(),
		message.DeletedAt(),
	)

	return err
}

// FindByID retrieves a message by ID
func (r *PostgresMessageRepository) FindByID(ctx context.Context, messageID string) (*entities.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, message_type, content, metadata, is_read, created_at, read_at, deleted_at
		FROM messages
		WHERE id = $1
	`

	var (
		id, chatID, senderID, msgTypeStr, content string
		metadataJSON                              []byte
		isRead                                    bool
		createdAt                                 sql.NullTime
		readAt, deletedAt                         sql.NullTime
		metadata                                  map[string]string
	)

	err := r.db.QueryRowContext(ctx, query, messageID).Scan(
		&id, &chatID, &senderID, &msgTypeStr, &content, &metadataJSON, &isRead, &createdAt, &readAt, &deletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entities.ErrMessageNotFound
		}
		return nil, err
	}

	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		metadata = make(map[string]string)
	}

	msgType, err := valueobjects.NewMessageType(msgTypeStr)
	if err != nil {
		return nil, err
	}

	var readAtPtr, deletedAtPtr *sql.NullTime
	if readAt.Valid {
		readAtPtr = &readAt
	}
	if deletedAt.Valid {
		deletedAtPtr = &deletedAt
	}

	message := entities.ReconstructMessage(
		id, chatID, senderID, msgType, content, metadata, isRead,
		createdAt.Time,
		timePtr(readAtPtr),
		timePtr(deletedAtPtr),
	)

	return message, nil
}

// FindByChatID retrieves all messages for a chat room
func (r *PostgresMessageRepository) FindByChatID(ctx context.Context, chatID string, limit, offset int) ([]*entities.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, message_type, content, metadata, is_read, created_at, read_at, deleted_at
		FROM messages
		WHERE chat_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, chatID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*entities.Message
	for rows.Next() {
		var (
			id, chatID, senderID, msgTypeStr, content string
			metadataJSON                              []byte
			isRead                                    bool
			createdAt                                 sql.NullTime
			readAt, deletedAt                         sql.NullTime
			metadata                                  map[string]string
		)

		err := rows.Scan(&id, &chatID, &senderID, &msgTypeStr, &content, &metadataJSON, &isRead, &createdAt, &readAt, &deletedAt)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			metadata = make(map[string]string)
		}

		msgType, err := valueobjects.NewMessageType(msgTypeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid message type for message %s: %w", id, err)
		}

		var readAtPtr, deletedAtPtr *sql.NullTime
		if readAt.Valid {
			readAtPtr = &readAt
		}
		if deletedAt.Valid {
			deletedAtPtr = &deletedAt
		}

		message := entities.ReconstructMessage(
			id, chatID, senderID, msgType, content, metadata, isRead,
			createdAt.Time,
			timePtr(readAtPtr),
			timePtr(deletedAtPtr),
		)

		messages = append(messages, message)
	}

	return messages, rows.Err()
}

// Delete removes a message (soft delete)
func (r *PostgresMessageRepository) Delete(ctx context.Context, messageID string) error {
	query := `UPDATE messages SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, messageID)
	return err
}

// MarkAsRead marks a message as read
func (r *PostgresMessageRepository) MarkAsRead(ctx context.Context, messageID string) error {
	query := `UPDATE messages SET is_read = true, read_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, messageID)
	return err
}

// CountUnreadByChatID counts unread messages in a chat
func (r *PostgresMessageRepository) CountUnreadByChatID(ctx context.Context, chatID string, userID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM messages
		WHERE chat_id = $1 AND sender_id != $2 AND is_read = false AND deleted_at IS NULL
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, chatID, userID).Scan(&count)
	return count, err
}

// FindUnreadByChatID retrieves unread messages for a user in a chat
func (r *PostgresMessageRepository) FindUnreadByChatID(ctx context.Context, chatID string, userID string) ([]*entities.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, message_type, content, metadata, is_read, created_at, read_at, deleted_at
		FROM messages
		WHERE chat_id = $1 AND sender_id != $2 AND is_read = false AND deleted_at IS NULL
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, chatID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*entities.Message
	for rows.Next() {
		var (
			id, chatID, senderID, msgTypeStr, content string
			metadataJSON                              []byte
			isRead                                    bool
			createdAt                                 sql.NullTime
			readAt, deletedAt                         sql.NullTime
			metadata                                  map[string]string
		)

		err := rows.Scan(&id, &chatID, &senderID, &msgTypeStr, &content, &metadataJSON, &isRead, &createdAt, &readAt, &deletedAt)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			metadata = make(map[string]string)
		}

		msgType, err := valueobjects.NewMessageType(msgTypeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid message type for message %s: %w", id, err)
		}

		var readAtPtr, deletedAtPtr *sql.NullTime
		if readAt.Valid {
			readAtPtr = &readAt
		}
		if deletedAt.Valid {
			deletedAtPtr = &deletedAt
		}

		message := entities.ReconstructMessage(
			id, chatID, senderID, msgType, content, metadata, isRead,
			createdAt.Time,
			timePtr(readAtPtr),
			timePtr(deletedAtPtr),
		)

		messages = append(messages, message)
	}

	return messages, rows.Err()
}

// Helper function to convert sql.NullTime to *time.Time
func timePtr(nt *sql.NullTime) *time.Time {
	if nt == nil || !nt.Valid {
		return nil
	}
	t := nt.Time
	return &t
}

package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/yourusername/connect/chat-service/domain/entities"
	"github.com/yourusername/connect/chat-service/domain/repositories"
	"github.com/yourusername/connect/chat-service/domain/valueobjects"
)

// PostgresChatRepository implements repositories.ChatRepository
type PostgresChatRepository struct {
	db *sql.DB
}

// NewPostgresChatRepository creates a new PostgresChatRepository
func NewPostgresChatRepository(db *sql.DB) repositories.ChatRepository {
	return &PostgresChatRepository{db: db}
}

// Save creates or updates a chat
func (r *PostgresChatRepository) Save(ctx context.Context, chat *entities.Chat) error {
	query := `
		INSERT INTO chats (id, participant1_id, participant2_id, status, match_score, created_at, updated_at, ended_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at,
			ended_at = EXCLUDED.ended_at
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		chat.ID(),
		chat.Participant1(),
		chat.Participant2(),
		chat.Status().String(),
		chat.MatchScore(),
		chat.CreatedAt(),
		chat.UpdatedAt(),
		chat.EndedAt(),
	)

	return err
}

// FindByID retrieves a chat by ID
func (r *PostgresChatRepository) FindByID(ctx context.Context, id string) (*entities.Chat, error) {
	query := `
		SELECT id, participant1_id, participant2_id, status, match_score, created_at, updated_at, ended_at
		FROM chats
		WHERE id = $1
	`

	var (
		chatID, participant1, participant2, statusStr string
		matchScore                                    float64
		createdAt, updatedAt                          time.Time
		endedAt                                       sql.NullTime
	)

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&chatID,
		&participant1,
		&participant2,
		&statusStr,
		&matchScore,
		&createdAt,
		&updatedAt,
		&endedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entities.ErrChatNotFound
		}
		return nil, err
	}

	status, err := valueobjects.NewChatStatus(statusStr)
	if err != nil {
		return nil, err
	}

	var endedAtPtr *time.Time
	if endedAt.Valid {
		endedAtPtr = &endedAt.Time
	}

	return entities.ReconstructChat(
		chatID,
		participant1,
		participant2,
		status,
		matchScore,
		createdAt,
		updatedAt,
		endedAtPtr,
	), nil
}

// FindByParticipants retrieves a chat by participant IDs
func (r *PostgresChatRepository) FindByParticipants(ctx context.Context, participant1, participant2 string) (*entities.Chat, error) {
	query := `
		SELECT id, participant1_id, participant2_id, status, match_score, created_at, updated_at, ended_at
		FROM chats
		WHERE (participant1_id = $1 AND participant2_id = $2) OR (participant1_id = $2 AND participant2_id = $1)
		ORDER BY created_at DESC
		LIMIT 1
	`

	var (
		chatID, p1, p2, statusStr string
		matchScore                float64
		createdAt, updatedAt      time.Time
		endedAt                   sql.NullTime
	)

	err := r.db.QueryRowContext(ctx, query, participant1, participant2).Scan(
		&chatID,
		&p1,
		&p2,
		&statusStr,
		&matchScore,
		&createdAt,
		&updatedAt,
		&endedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entities.ErrChatNotFound
		}
		return nil, err
	}

	status, err := valueobjects.NewChatStatus(statusStr)
	if err != nil {
		return nil, err
	}

	var endedAtPtr *time.Time
	if endedAt.Valid {
		endedAtPtr = &endedAt.Time
	}

	return entities.ReconstructChat(
		chatID,
		p1,
		p2,
		status,
		matchScore,
		createdAt,
		updatedAt,
		endedAtPtr,
	), nil
}

// FindActiveByUser retrieves active chat for a user
func (r *PostgresChatRepository) FindActiveByUser(ctx context.Context, userID string) (*entities.Chat, error) {
	query := `
		SELECT id, participant1_id, participant2_id, status, match_score, created_at, updated_at, ended_at
		FROM chats
		WHERE (participant1_id = $1 OR participant2_id = $1) AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`

	var (
		chatID, p1, p2, statusStr string
		matchScore                float64
		createdAt, updatedAt      time.Time
		endedAt                   sql.NullTime
	)

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&chatID,
		&p1,
		&p2,
		&statusStr,
		&matchScore,
		&createdAt,
		&updatedAt,
		&endedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entities.ErrChatNotFound
		}
		return nil, err
	}

	status, err := valueobjects.NewChatStatus(statusStr)
	if err != nil {
		return nil, err
	}

	var endedAtPtr *time.Time
	if endedAt.Valid {
		endedAtPtr = &endedAt.Time
	}

	return entities.ReconstructChat(
		chatID,
		p1,
		p2,
		status,
		matchScore,
		createdAt,
		updatedAt,
		endedAtPtr,
	), nil
}

// FindByUser retrieves all chats for a user
func (r *PostgresChatRepository) FindByUser(ctx context.Context, userID string, limit, offset int) ([]*entities.Chat, error) {
	query := `
		SELECT id, participant1_id, participant2_id, status, match_score, created_at, updated_at, ended_at
		FROM chats
		WHERE participant1_id = $1 OR participant2_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []*entities.Chat

	for rows.Next() {
		var (
			chatID, p1, p2, statusStr string
			matchScore                float64
			createdAt, updatedAt      time.Time
			endedAt                   sql.NullTime
		)

		err := rows.Scan(
			&chatID,
			&p1,
			&p2,
			&statusStr,
			&matchScore,
			&createdAt,
			&updatedAt,
			&endedAt,
		)

		if err != nil {
			return nil, err
		}

		status, err := valueobjects.NewChatStatus(statusStr)
		if err != nil {
			return nil, err
		}

		var endedAtPtr *time.Time
		if endedAt.Valid {
			endedAtPtr = &endedAt.Time
		}

		chat := entities.ReconstructChat(
			chatID,
			p1,
			p2,
			status,
			matchScore,
			createdAt,
			updatedAt,
			endedAtPtr,
		)

		chats = append(chats, chat)
	}

	return chats, rows.Err()
}

// Delete removes a chat
func (r *PostgresChatRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM chats WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ExistsActiveChat checks if user has an active chat
func (r *PostgresChatRepository) ExistsActiveChat(ctx context.Context, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM chats WHERE (participant1_id = $1 OR participant2_id = $1) AND status = 'active')`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&exists)
	return exists, err
}

// CountByUser returns the total number of chats a user participates in.
func (r *PostgresChatRepository) CountByUser(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM chats WHERE participant1_id = $1 OR participant2_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}

package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/yourusername/connect/matching-service/domain/entities"
	"github.com/yourusername/connect/matching-service/domain/repositories"
	"github.com/yourusername/connect/matching-service/domain/valueobjects"
)

// PostgresMatchHistoryRepository implements MatchHistoryRepository using PostgreSQL
type PostgresMatchHistoryRepository struct {
	db                 *sql.DB
	recentMatchWindow  time.Duration
}

// NewPostgresMatchHistoryRepository creates a new PostgreSQL match history repository
func NewPostgresMatchHistoryRepository(db *sql.DB) repositories.MatchHistoryRepository {
	return &PostgresMatchHistoryRepository{db: db, recentMatchWindow: 24 * time.Hour}
}

// Save saves a match
func (r *PostgresMatchHistoryRepository) Save(ctx context.Context, match *entities.Match) error {
	query := `
		INSERT INTO match_history (id, user1_id, user2_id, match_score, chat_id, status, created_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			chat_id = EXCLUDED.chat_id,
			status = EXCLUDED.status,
			completed_at = EXCLUDED.completed_at
	`

	_, err := r.db.ExecContext(ctx, query,
		match.ID(),
		match.User1ID(),
		match.User2ID(),
		match.MatchScore().Value(),
		match.ChatID(),
		match.Status(),
		match.CreatedAt(),
		match.CompletedAt(),
	)

	return err
}

// FindByID retrieves a match by ID
func (r *PostgresMatchHistoryRepository) FindByID(ctx context.Context, matchID string) (*entities.Match, error) {
	query := `
		SELECT id, user1_id, user2_id, match_score, chat_id, status, created_at, completed_at
		FROM match_history
		WHERE id = $1
	`

	var (
		id, user1ID, user2ID, chatID, status string
		matchScore                           float64
		createdAt                            sql.NullTime
		completedAt                          sql.NullTime
	)

	err := r.db.QueryRowContext(ctx, query, matchID).Scan(
		&id, &user1ID, &user2ID, &matchScore, &chatID, &status, &createdAt, &completedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entities.ErrMatchNotFound
		}
		return nil, err
	}

	score, _ := valueobjects.NewMatchScore(matchScore)

	var completedAtPtr *time.Time
	if completedAt.Valid {
		completedAtPtr = &completedAt.Time
	}

	return entities.ReconstructMatch(
		id, user1ID, user2ID, score, chatID, status, createdAt.Time, completedAtPtr,
	), nil
}

// FindByUserID retrieves all matches for a user
func (r *PostgresMatchHistoryRepository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*entities.Match, error) {
	query := `
		SELECT id, user1_id, user2_id, match_score, chat_id, status, created_at, completed_at
		FROM match_history
		WHERE user1_id = $1 OR user2_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*entities.Match
	for rows.Next() {
		var (
			id, user1ID, user2ID, chatID, status string
			matchScore                           float64
			createdAt                            sql.NullTime
			completedAt                          sql.NullTime
		)

		err := rows.Scan(&id, &user1ID, &user2ID, &matchScore, &chatID, &status, &createdAt, &completedAt)
		if err != nil {
			continue
		}

		score, _ := valueobjects.NewMatchScore(matchScore)

		var completedAtPtr *time.Time
		if completedAt.Valid {
			completedAtPtr = &completedAt.Time
		}

		match := entities.ReconstructMatch(
			id, user1ID, user2ID, score, chatID, status, createdAt.Time, completedAtPtr,
		)

		matches = append(matches, match)
	}

	return matches, rows.Err()
}

// FindActiveByUserID retrieves active matches for a user
func (r *PostgresMatchHistoryRepository) FindActiveByUserID(ctx context.Context, userID string) ([]*entities.Match, error) {
	query := `
		SELECT id, user1_id, user2_id, match_score, chat_id, status, created_at, completed_at
		FROM match_history
		WHERE (user1_id = $1 OR user2_id = $1) AND status = 'active'
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*entities.Match
	for rows.Next() {
		var (
			id, user1ID, user2ID, chatID, status string
			matchScore                           float64
			createdAt                            sql.NullTime
			completedAt                          sql.NullTime
		)

		err := rows.Scan(&id, &user1ID, &user2ID, &matchScore, &chatID, &status, &createdAt, &completedAt)
		if err != nil {
			continue
		}

		score, _ := valueobjects.NewMatchScore(matchScore)

		var completedAtPtr *time.Time
		if completedAt.Valid {
			completedAtPtr = &completedAt.Time
		}

		match := entities.ReconstructMatch(
			id, user1ID, user2ID, score, chatID, status, createdAt.Time, completedAtPtr,
		)

		matches = append(matches, match)
	}

	return matches, rows.Err()
}

// HasRecentMatch checks if two users have recently matched (within the configured window)
func (r *PostgresMatchHistoryRepository) HasRecentMatch(ctx context.Context, user1ID, user2ID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM match_history
			WHERE ((user1_id = $1 AND user2_id = $2) OR (user1_id = $2 AND user2_id = $1))
			AND created_at > NOW() - $3::interval
		)
	`

	// Format duration as PostgreSQL-compatible interval (e.g. "86400 seconds").
	intervalStr := fmt.Sprintf("%d seconds", int(r.recentMatchWindow.Seconds()))

	var exists bool
	err := r.db.QueryRowContext(ctx, query, user1ID, user2ID, intervalStr).Scan(&exists)
	return exists, err
}

package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"XTalk/services/matching/domain/entities"
	"XTalk/services/matching/domain/repositories"
	"XTalk/services/matching/domain/valueobjects"
)

// PostgresMatchingQueueRepository implements MatchingQueueRepository using PostgreSQL
type PostgresMatchingQueueRepository struct {
	db *sql.DB
}

// NewPostgresMatchingQueueRepository creates a new PostgreSQL matching queue repository
func NewPostgresMatchingQueueRepository(db *sql.DB) repositories.MatchingQueueRepository {
	return &PostgresMatchingQueueRepository{db: db}
}

// AddToQueue adds a candidate to the matching queue
func (r *PostgresMatchingQueueRepository) AddToQueue(ctx context.Context, candidate *entities.MatchingCandidate) error {
	prefs := candidate.Preferences()
	interestsJSON, err := json.Marshal(prefs.Interests())
	if err != nil {
		return fmt.Errorf("marshal interests: %w", err)
	}

	query := `
		INSERT INTO matching_queue (user_id, age, min_age, max_age, interests, gender, location, joined_at, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.db.ExecContext(ctx, query,
		candidate.UserID(),
		prefs.Age(),
		prefs.AgeRange().Min(),
		prefs.AgeRange().Max(),
		interestsJSON,
		prefs.Gender(),
		prefs.Location(),
		candidate.JoinedAt(),
		candidate.Priority(),
	)

	return err
}

// RemoveFromQueue atomically removes a candidate from the queue.
// Returns true if a row was actually deleted, false if no matching row existed.
func (r *PostgresMatchingQueueRepository) RemoveFromQueue(ctx context.Context, userID string) (bool, error) {
	query := `DELETE FROM matching_queue WHERE user_id = $1 RETURNING user_id`
	var deleted string
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&deleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// FindInQueue finds a candidate in the queue
func (r *PostgresMatchingQueueRepository) FindInQueue(ctx context.Context, userID string) (*entities.MatchingCandidate, error) {
	query := `
		SELECT user_id, age, min_age, max_age, interests, gender, location, joined_at, priority
		FROM matching_queue
		WHERE user_id = $1
	`

	var (
		uid, gender, location         string
		age, minAge, maxAge, priority int
		interestsJSON                 []byte
		joinedAt                      sql.NullTime
	)

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&uid, &age, &minAge, &maxAge, &interestsJSON, &gender, &location, &joinedAt, &priority,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entities.ErrCandidateNotFound
		}
		return nil, err
	}

	var interests []string
	if err := json.Unmarshal(interestsJSON, &interests); err != nil {
		return nil, fmt.Errorf("unmarshal interests: %w", err)
	}

	ageRange, _ := valueobjects.NewAgeRange(minAge, maxAge)
	prefs, _ := valueobjects.NewPreferences(age, ageRange, interests, gender, location)

	return entities.ReconstructMatchingCandidate(uid, prefs, joinedAt.Time, priority), nil
}

// GetAllCandidates retrieves all candidates in the queue
func (r *PostgresMatchingQueueRepository) GetAllCandidates(ctx context.Context) ([]*entities.MatchingCandidate, error) {
	query := `
		SELECT user_id, age, min_age, max_age, interests, gender, location, joined_at, priority
		FROM matching_queue
		ORDER BY priority DESC, joined_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []*entities.MatchingCandidate
	for rows.Next() {
		var (
			uid, gender, location         string
			age, minAge, maxAge, priority int
			interestsJSON                 []byte
			joinedAt                      sql.NullTime
		)

		err := rows.Scan(&uid, &age, &minAge, &maxAge, &interestsJSON, &gender, &location, &joinedAt, &priority)
		if err != nil {
			return nil, fmt.Errorf("scan candidate row: %w", err)
		}

		var interests []string
		if err := json.Unmarshal(interestsJSON, &interests); err != nil {
			return nil, fmt.Errorf("unmarshal interests for user %s: %w", uid, err)
		}

		ageRange, _ := valueobjects.NewAgeRange(minAge, maxAge)
		prefs, _ := valueobjects.NewPreferences(age, ageRange, interests, gender, location)

		candidate := entities.ReconstructMatchingCandidate(uid, prefs, joinedAt.Time, priority)
		candidates = append(candidates, candidate)
	}

	return candidates, rows.Err()
}

// IsInQueue checks if a user is in the queue
func (r *PostgresMatchingQueueRepository) IsInQueue(ctx context.Context, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM matching_queue WHERE user_id = $1)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&exists)
	return exists, err
}

// UpdatePriority updates a candidate's priority
func (r *PostgresMatchingQueueRepository) UpdatePriority(ctx context.Context, userID string, priority int) error {
	query := `UPDATE matching_queue SET priority = $1 WHERE user_id = $2`
	_, err := r.db.ExecContext(ctx, query, priority, userID)
	return err
}

// UpdateCandidate replaces the preferences for a candidate already in the queue.
func (r *PostgresMatchingQueueRepository) UpdateCandidate(ctx context.Context, candidate *entities.MatchingCandidate) error {
	prefs := candidate.Preferences()
	interestsJSON, err := json.Marshal(prefs.Interests())
	if err != nil {
		return fmt.Errorf("marshal interests: %w", err)
	}

	query := `
		UPDATE matching_queue
		SET age = $2, min_age = $3, max_age = $4, interests = $5, gender = $6, location = $7
		WHERE user_id = $1
	`

	res, err := r.db.ExecContext(ctx, query,
		candidate.UserID(),
		prefs.Age(),
		prefs.AgeRange().Min(),
		prefs.AgeRange().Max(),
		interestsJSON,
		prefs.Gender(),
		prefs.Location(),
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return entities.ErrCandidateNotFound
	}
	return nil
}

// FindCompatibleCandidates retrieves candidates pre-filtered by age range
// and gender compatibility at the DB level, ordered by priority/wait time.
func (r *PostgresMatchingQueueRepository) FindCompatibleCandidates(ctx context.Context, candidate *entities.MatchingCandidate, limit int) ([]*entities.MatchingCandidate, error) {
	prefs := candidate.Preferences()
	ageRange := prefs.AgeRange()

	query := `
		SELECT user_id, age, min_age, max_age, interests, gender, location, joined_at, priority
		FROM matching_queue
		WHERE user_id != $1
		  AND age BETWEEN $2 AND $3
		  AND $4 BETWEEN min_age AND max_age
		  AND ($5 = '' OR gender = $5)
		  AND ($6 = '' OR $6 = gender OR gender = '')
		ORDER BY priority DESC, joined_at ASC
		LIMIT $7
	`

	rows, err := r.db.QueryContext(ctx, query,
		candidate.UserID(),
		ageRange.Min(),
		ageRange.Max(),
		prefs.Age(),
		prefs.Gender(), // $5: candidate's preferred gender filter
		prefs.Gender(), // $6: other's gender preference filter
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []*entities.MatchingCandidate
	for rows.Next() {
		var (
			uid, gender, location         string
			age, minAge, maxAge, priority int
			interestsJSON                 []byte
			joinedAt                      sql.NullTime
		)

		err := rows.Scan(&uid, &age, &minAge, &maxAge, &interestsJSON, &gender, &location, &joinedAt, &priority)
		if err != nil {
			return nil, fmt.Errorf("scan compatible candidate row: %w", err)
		}

		var interests []string
		if err := json.Unmarshal(interestsJSON, &interests); err != nil {
			return nil, fmt.Errorf("unmarshal interests for user %s: %w", uid, err)
		}

		ar, _ := valueobjects.NewAgeRange(minAge, maxAge)
		p, _ := valueobjects.NewPreferences(age, ar, interests, gender, location)

		candidate := entities.ReconstructMatchingCandidate(uid, p, joinedAt.Time, priority)
		candidates = append(candidates, candidate)
	}

	return candidates, rows.Err()
}

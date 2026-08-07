package persistence

import (
	"XTalk/services/auth/application/interfaces"
	"XTalk/services/auth/domain/users"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// PostgresUserRepository implements users.UserRepository
type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) CreateWithEvent(
	ctx context.Context,
	user *users.User,
	event interfaces.UserRegisteredEvent,
) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin registration transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const insertUser = `
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	if _, err = tx.ExecContext(ctx, insertUser,
		user.ID(), user.Username(), user.Email().String(), user.PasswordHash(),
		user.CreatedAt(), user.UpdatedAt(), user.LastSeen(),
	); err != nil {
		return fmt.Errorf("insert auth user: %w", err)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal user registered event: %w", err)
	}
	const insertEvent = `
		INSERT INTO auth_outbox (aggregate_id, event_type, payload)
		VALUES ($1, 'auth.user_registered', $2)`
	if _, err = tx.ExecContext(ctx, insertEvent, user.ID(), payload); err != nil {
		return fmt.Errorf("insert registration outbox event: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit registration transaction: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) Save(ctx context.Context, user *users.User) error {
	query := `
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			updated_at = EXCLUDED.updated_at,
			last_seen = EXCLUDED.last_seen
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID(),
		user.Username(),
		user.Email().String(),
		user.PasswordHash(),
		user.CreatedAt(),
		user.UpdatedAt(),
		user.LastSeen(),
	)

	return err
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*users.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at, last_seen
		FROM users WHERE id = $1
	`

	var (
		userID, username, emailStr, passwordHash string
		createdAt, updatedAt, lastSeen           sql.NullTime
	)

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&userID, &username, &emailStr, &passwordHash,
		&createdAt, &updatedAt, &lastSeen,
	)

	if err == sql.ErrNoRows {
		return nil, users.ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	email, err := users.NewEmail(emailStr)
	if err != nil {
		return nil, err
	}

	user := users.ReconstructUser(userID, username, email, passwordHash, createdAt.Time, updatedAt.Time, lastSeen.Time)
	return user, nil
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email users.Email) (*users.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at, last_seen
		FROM users WHERE email = $1
	`

	var (
		userID, username, emailStr, passwordHash string
		createdAt, updatedAt, lastSeen           sql.NullTime
	)

	err := r.db.QueryRowContext(ctx, query, email.String()).Scan(
		&userID, &username, &emailStr, &passwordHash,
		&createdAt, &updatedAt, &lastSeen,
	)

	if err == sql.ErrNoRows {
		return nil, users.ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	user := users.ReconstructUser(userID, username, email, passwordHash, createdAt.Time, updatedAt.Time, lastSeen.Time)
	return user, nil
}

func (r *PostgresUserRepository) EmailExists(ctx context.Context, email users.Email) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, email.String()).Scan(&exists)

	return exists, err
}

func (r *PostgresUserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return users.ErrUserNotFound
	}

	return nil
}

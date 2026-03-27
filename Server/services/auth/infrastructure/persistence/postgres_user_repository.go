package persistence

import (
	"context"
	"database/sql"
)

// PostgresUserRepository implements repositories.UserRepository
type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) repositories.UserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Save(ctx context.Context, user *entities.User) error {
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

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*entities.User, error) {
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
		return nil, entities.ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	email, err := valueobjects.NewEmail(emailStr)
	if err != nil {
		return nil, err
	}

	user := entities.ReconstructUser(userID, username, email, passwordHash, createdAt.Time, updatedAt.Time, lastSeen.Time)
	return user, nil
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email valueobjects.Email) (*entities.User, error) {
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
		return nil, entities.ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	user := entities.ReconstructUser(userID, username, email, passwordHash, createdAt.Time, updatedAt.Time, lastSeen.Time)
	return user, nil
}

func (r *PostgresUserRepository) EmailExists(ctx context.Context, email valueobjects.Email) (bool, error) {
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
		return entities.ErrUserNotFound
	}

	return nil
}

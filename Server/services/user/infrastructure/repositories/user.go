package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"XTalk/services/user/domain/users"

	"github.com/lib/pq"
)

// PostgresUserRepository is the PostgreSQL adapter for the domain repository port.
type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// Create is idempotent by aggregate ID so an at-least-once registration event
// can safely be delivered again. Other unique conflicts remain domain errors.
func (r *PostgresUserRepository) Create(ctx context.Context, user *users.User) error {
	const query = `
		INSERT INTO users (
			id, username, email, age, gender, country, language, interests,
			status, bio, avatar_url, created_at, updated_at, last_seen, is_active
		) VALUES ($1,$2,$3,NULL,NULL,NULL,NULL,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query,
		user.ID(), user.Username(), user.Email().Value(), pq.Array(user.Interests()),
		user.Status().String(), user.Bio(), user.AvatarURL(), user.CreatedAt(),
		user.UpdatedAt(), user.LastSeen(), user.IsActive(),
	)
	if err != nil {
		return mapWriteError(err)
	}
	return nil
}

func (r *PostgresUserRepository) Save(ctx context.Context, user *users.User) error {
	const query = `
		UPDATE users SET
			username = $2, age = NULLIF($3, 0), gender = NULLIF($4, ''),
			country = NULLIF($5, ''), language = NULLIF($6, ''), interests = $7,
			status = $8, bio = $9, avatar_url = $10, updated_at = $11,
			last_seen = $12, is_active = $13
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		user.ID(), user.Username(), user.Age(), user.Gender(), user.Country(),
		user.Language(), pq.Array(user.Interests()), user.Status().String(),
		user.Bio(), user.AvatarURL(), user.UpdatedAt(), user.LastSeen(), user.IsActive(),
	)
	if err != nil {
		return mapWriteError(err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated row count: %w", err)
	}
	if rows == 0 {
		return users.ErrUserNotFound
	}
	return nil
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*users.User, error) {
	return r.findOne(ctx, selectUser+` WHERE id = $1`, id)
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email users.Email) (*users.User, error) {
	return r.findOne(ctx, selectUser+` WHERE email = $1`, email.Value())
}

const selectUser = `
	SELECT id, username, email, age, gender, country, language, interests,
	       status, bio, avatar_url, created_at, updated_at, last_seen, is_active
	FROM users`

func (r *PostgresUserRepository) findOne(ctx context.Context, query string, args ...any) (*users.User, error) {
	user, err := scanUser(r.db.QueryRowContext(ctx, query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, users.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return user, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(row scanner) (*users.User, error) {
	var (
		id, username, rawEmail         string
		age                            sql.NullInt32
		gender, country, language      sql.NullString
		interests                      []string
		rawStatus                      string
		bio, avatarURL                 sql.NullString
		createdAt, updatedAt, lastSeen sql.NullTime
		isActive                       bool
	)
	if err := row.Scan(
		&id, &username, &rawEmail, &age, &gender, &country, &language,
		pq.Array(&interests), &rawStatus, &bio, &avatarURL,
		&createdAt, &updatedAt, &lastSeen, &isActive,
	); err != nil {
		return nil, err
	}

	email, err := users.NewEmail(rawEmail)
	if err != nil {
		return nil, fmt.Errorf("invalid persisted email: %w", err)
	}
	status, err := users.NewStatus(rawStatus)
	if err != nil {
		return nil, fmt.Errorf("invalid persisted status: %w", err)
	}
	return users.ReconstructUser(
		id, username, email, int(age.Int32), gender.String, country.String,
		language.String, interests, status, bio.String, avatarURL.String,
		createdAt.Time, updatedAt.Time, lastSeen.Time, isActive,
	), nil
}

func mapWriteError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return users.ErrUserAlreadyExists
	}
	return fmt.Errorf("persist user: %w", err)
}

var _ users.UserRepository = (*PostgresUserRepository)(nil)

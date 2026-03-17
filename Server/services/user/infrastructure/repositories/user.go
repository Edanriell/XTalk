package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

// PostgresUserRepository implements repositories.UserRepository.
type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) repositories.UserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Save(ctx context.Context, user *entities.User) error {
	const query = `
		INSERT INTO users (id, username, email, age, gender, country, language,
		                    interests, status, bio, avatar_url,
		                    created_at, updated_at, last_seen, is_active)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (id) DO UPDATE SET
			username   = EXCLUDED.username,
			age        = EXCLUDED.age,
			gender     = EXCLUDED.gender,
			country    = EXCLUDED.country,
			language   = EXCLUDED.language,
			interests  = EXCLUDED.interests,
			status     = EXCLUDED.status,
			bio        = EXCLUDED.bio,
			avatar_url = EXCLUDED.avatar_url,
			updated_at = EXCLUDED.updated_at,
			last_seen  = EXCLUDED.last_seen,
			is_active  = EXCLUDED.is_active`

	_, err := r.db.ExecContext(ctx, query,
		user.ID(),
		user.Username(),
		user.Email().Value(),
		user.Age(),
		user.Gender(),
		user.Country(),
		user.Language(),
		pq.Array(user.Interests()),
		user.Status().String(),
		user.Bio(),
		user.AvatarURL(),
		user.CreatedAt(),
		user.UpdatedAt(),
		user.LastSeen(),
		user.IsActive(),
	)
	return err
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*entities.User, error) {
	return r.scanOne(ctx,
		`SELECT id,username,email,age,gender,country,language,interests,status,bio,avatar_url,created_at,updated_at,last_seen,is_active
		 FROM users WHERE id = $1`, id)
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email valueobjects.Email) (*entities.User, error) {
	return r.scanOne(ctx,
		`SELECT id,username,email,age,gender,country,language,interests,status,bio,avatar_url,created_at,updated_at,last_seen,is_active
		 FROM users WHERE email = $1`, email.Value())
}

func (r *PostgresUserRepository) FindByUsername(ctx context.Context, username string) (*entities.User, error) {
	return r.scanOne(ctx,
		`SELECT id,username,email,age,gender,country,language,interests,status,bio,avatar_url,created_at,updated_at,last_seen,is_active
		 FROM users WHERE username = $1`, username)
}

func (r *PostgresUserRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *PostgresUserRepository) ExistsByEmail(ctx context.Context, email valueobjects.Email) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email.Value()).Scan(&exists)
	return exists, err
}

func (r *PostgresUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`, username).Scan(&exists)
	return exists, err
}

func (r *PostgresUserRepository) FindActiveUsers(ctx context.Context, limit, offset int) ([]*entities.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id,username,email,age,gender,country,language,interests,status,bio,avatar_url,created_at,updated_at,last_seen,is_active
		 FROM users WHERE is_active = true ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*entities.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *PostgresUserRepository) UpdateStatus(ctx context.Context, id string, status valueobjects.Status) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET status = $1, last_seen = NOW(), updated_at = NOW() WHERE id = $2`,
		status.String(), id)
	return err
}

// --- helpers ---

// scanOne executes a query expected to return a single user row.
func (r *PostgresUserRepository) scanOne(ctx context.Context, query string, args ...any) (*entities.User, error) {
	row := r.db.QueryRowContext(ctx, query, args...)
	u, err := scanUserRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entities.ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanUserRow(s *sql.Row) (*entities.User, error) {
	var (
		id, username, emailStr string
		age                    sql.NullInt32
		gender, country        sql.NullString
		language               sql.NullString
		interests              []string
		statusStr              string
		bio, avatarURL         sql.NullString
		createdAt, updatedAt   sql.NullTime
		lastSeen               sql.NullTime
		isActive               bool
	)

	if err := s.Scan(&id, &username, &emailStr, &age, &gender, &country, &language,
		pq.Array(&interests), &statusStr, &bio, &avatarURL,
		&createdAt, &updatedAt, &lastSeen, &isActive); err != nil {
		return nil, err
	}

	return reconstruct(id, username, emailStr, int(age.Int32), gender.String, country.String, language.String,
		interests, statusStr, bio.String, avatarURL.String, createdAt, updatedAt, lastSeen, isActive)
}

func scanUser(rows *sql.Rows) (*entities.User, error) {
	var (
		id, username, emailStr string
		age                    sql.NullInt32
		gender, country        sql.NullString
		language               sql.NullString
		interests              []string
		statusStr              string
		bio, avatarURL         sql.NullString
		createdAt, updatedAt   sql.NullTime
		lastSeen               sql.NullTime
		isActive               bool
	)

	if err := rows.Scan(&id, &username, &emailStr, &age, &gender, &country, &language,
		pq.Array(&interests), &statusStr, &bio, &avatarURL,
		&createdAt, &updatedAt, &lastSeen, &isActive); err != nil {
		return nil, err
	}

	return reconstruct(id, username, emailStr, int(age.Int32), gender.String, country.String, language.String,
		interests, statusStr, bio.String, avatarURL.String, createdAt, updatedAt, lastSeen, isActive)
}

func reconstruct(
	id, username, emailStr string, age int,
	gender, country, language string,
	interests []string, statusStr, bio, avatarURL string,
	createdAt, updatedAt, lastSeen sql.NullTime, isActive bool,
) (*entities.User, error) {
	email, err := valueobjects.NewEmail(emailStr)
	if err != nil {
		return nil, err
	}
	status, err := valueobjects.NewStatus(statusStr)
	if err != nil {
		return nil, err
	}
	return entities.ReconstructUser(
		id, username, email, age, gender, country, language,
		interests, status, bio, avatarURL,
		createdAt.Time, updatedAt.Time, lastSeen.Time, isActive,
	), nil
}

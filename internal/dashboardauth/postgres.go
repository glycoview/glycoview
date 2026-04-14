package dashboardauth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const postgresSchemaSQL = `
CREATE TABLE IF NOT EXISTS app_users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  role TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS app_sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS app_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS app_sessions_user_id_idx ON app_sessions (user_id);
CREATE INDEX IF NOT EXISTS app_sessions_expires_at_idx ON app_sessions (expires_at);
`

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if _, err := pool.Exec(ctx, postgresSchemaSQL); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *PostgresStore) HasUsers(ctx context.Context) (bool, error) {
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM app_users`).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *PostgresStore) CreateUser(ctx context.Context, user User) (User, error) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO app_users (id, username, display_name, role, password_hash, active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, user.ID, user.Username, user.DisplayName, user.Role, user.PasswordHash, user.Active, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return User{}, ErrInvalidCredentials
		}
		return User{}, err
	}
	return user, nil
}

func (s *PostgresStore) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, username, display_name, role, password_hash, active, created_at, updated_at
		FROM app_users
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *PostgresStore) GetUserByUsername(ctx context.Context, username string) (User, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, username, display_name, role, password_hash, active, created_at, updated_at
		FROM app_users
		WHERE username=$1
	`, username)
	user, err := scanUser(row)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (User, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, username, display_name, role, password_hash, active, created_at, updated_at
		FROM app_users
		WHERE id=$1
	`, id)
	user, err := scanUser(row)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *PostgresStore) UpdateUser(ctx context.Context, user User) (User, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE app_users
		SET username=$2, display_name=$3, role=$4, password_hash=$5, active=$6, updated_at=$7
		WHERE id=$1
	`, user.ID, user.Username, user.DisplayName, user.Role, user.PasswordHash, user.Active, user.UpdatedAt)
	if err != nil {
		return User{}, err
	}
	if tag.RowsAffected() == 0 {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (s *PostgresStore) GetSetting(ctx context.Context, key string) (string, error) {
	row := s.pool.QueryRow(ctx, `SELECT value FROM app_settings WHERE key=$1`, key)
	var value string
	err := row.Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return value, err
}

func (s *PostgresStore) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, key, value)
	return err
}

func (s *PostgresStore) CreateSession(ctx context.Context, session Session) (Session, error) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO app_sessions (id, user_id, token_hash, expires_at, created_at, last_seen_at)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, session.ID, session.UserID, session.TokenHash, session.ExpiresAt, session.CreatedAt, session.LastSeenAt)
	return session, err
}

func (s *PostgresStore) GetSessionByHash(ctx context.Context, tokenHash string) (Session, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at, last_seen_at
		FROM app_sessions
		WHERE token_hash=$1
	`, tokenHash)
	var session Session
	err := row.Scan(&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &session.CreatedAt, &session.LastSeenAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	return session, err
}

func (s *PostgresStore) DeleteSessionByHash(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM app_sessions WHERE token_hash=$1`, tokenHash)
	return err
}

func (s *PostgresStore) DeleteSessionsByUserID(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM app_sessions WHERE user_id=$1`, userID)
	return err
}

func (s *PostgresStore) TouchSession(ctx context.Context, tokenHash string, seenAt time.Time) error {
	tag, err := s.pool.Exec(ctx, `UPDATE app_sessions SET last_seen_at=$2 WHERE token_hash=$1`, tokenHash, seenAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner userScanner) (User, error) {
	var user User
	err := scanner.Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.Role,
		&user.PasswordHash,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

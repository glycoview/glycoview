package goals

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("goal not found")

// Store is the goal persistence interface.
type Store interface {
	List(ctx context.Context, includeArchived bool) ([]Goal, error)
	Get(ctx context.Context, id string) (Goal, error)
	Create(ctx context.Context, g Goal) (Goal, error)
	Update(ctx context.Context, g Goal) (Goal, error)
	SetStatus(ctx context.Context, id string, status Status, completedAt *time.Time) (Goal, error)
	Delete(ctx context.Context, id string) error
}

// ─────────────────────────────────────────────────────────────────────────────
// Postgres
// ─────────────────────────────────────────────────────────────────────────────

const postgresSchemaSQL = `
CREATE TABLE IF NOT EXISTS goals (
    id TEXT PRIMARY KEY,
    created_by TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    predicate JSONB NOT NULL,
    start_date DATE NOT NULL,
    target_date DATE,
    status TEXT NOT NULL DEFAULT 'active',
    rationale TEXT NOT NULL DEFAULT '',
    action_plan TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS goals_status_idx ON goals (status) WHERE deleted_at IS NULL;
`

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, pool *pgxpool.Pool) (*PostgresStore, error) {
	if _, err := pool.Exec(ctx, postgresSchemaSQL); err != nil {
		return nil, err
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) List(ctx context.Context, includeArchived bool) ([]Goal, error) {
	q := `SELECT id, created_by, title, predicate, start_date, target_date, status, rationale, action_plan, created_at, updated_at, completed_at
	      FROM goals WHERE deleted_at IS NULL`
	if !includeArchived {
		q += ` AND status != 'archived'`
	}
	q += ` ORDER BY status = 'active' DESC, target_date ASC NULLS LAST, created_at DESC`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Goal, 0)
	for rows.Next() {
		var g Goal
		var predRaw []byte
		var startDate time.Time
		var targetDate *time.Time
		var completedAt *time.Time
		if err := rows.Scan(&g.ID, &g.CreatedBy, &g.Title, &predRaw, &startDate, &targetDate,
			&g.Status, &g.Rationale, &g.ActionPlan, &g.CreatedAt, &g.UpdatedAt, &completedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(predRaw, &g.Predicate); err != nil {
			return nil, err
		}
		g.StartDate = startDate.Format("2006-01-02")
		if targetDate != nil {
			g.TargetDate = targetDate.Format("2006-01-02")
		}
		g.CompletedAt = completedAt
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Goal, error) {
	q := `SELECT id, created_by, title, predicate, start_date, target_date, status, rationale, action_plan, created_at, updated_at, completed_at
	      FROM goals WHERE id = $1 AND deleted_at IS NULL`
	row := s.pool.QueryRow(ctx, q, id)
	var g Goal
	var predRaw []byte
	var startDate time.Time
	var targetDate *time.Time
	var completedAt *time.Time
	if err := row.Scan(&g.ID, &g.CreatedBy, &g.Title, &predRaw, &startDate, &targetDate,
		&g.Status, &g.Rationale, &g.ActionPlan, &g.CreatedAt, &g.UpdatedAt, &completedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return g, ErrNotFound
		}
		return g, err
	}
	if err := json.Unmarshal(predRaw, &g.Predicate); err != nil {
		return g, err
	}
	g.StartDate = startDate.Format("2006-01-02")
	if targetDate != nil {
		g.TargetDate = targetDate.Format("2006-01-02")
	}
	g.CompletedAt = completedAt
	return g, nil
}

func (s *PostgresStore) Create(ctx context.Context, g Goal) (Goal, error) {
	if g.ID == "" {
		g.ID = newID()
	}
	if g.Status == "" {
		g.Status = StatusActive
	}
	now := time.Now().UTC()
	g.CreatedAt = now
	g.UpdatedAt = now
	predRaw, err := json.Marshal(g.Predicate)
	if err != nil {
		return g, err
	}
	startDate, err := parseDate(g.StartDate)
	if err != nil {
		return g, err
	}
	targetDate := parseNullableDate(g.TargetDate)
	_, err = s.pool.Exec(ctx, `
		INSERT INTO goals (id, created_by, title, predicate, start_date, target_date, status, rationale, action_plan, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, g.ID, g.CreatedBy, g.Title, predRaw, startDate, targetDate, g.Status, g.Rationale, g.ActionPlan, g.CreatedAt, g.UpdatedAt)
	if err != nil {
		return g, err
	}
	return g, nil
}

func (s *PostgresStore) Update(ctx context.Context, g Goal) (Goal, error) {
	predRaw, err := json.Marshal(g.Predicate)
	if err != nil {
		return g, err
	}
	startDate, err := parseDate(g.StartDate)
	if err != nil {
		return g, err
	}
	targetDate := parseNullableDate(g.TargetDate)
	g.UpdatedAt = time.Now().UTC()
	cmd, err := s.pool.Exec(ctx, `
		UPDATE goals
		   SET title = $2, predicate = $3, start_date = $4, target_date = $5,
		       rationale = $6, action_plan = $7, updated_at = $8
		 WHERE id = $1 AND deleted_at IS NULL
	`, g.ID, g.Title, predRaw, startDate, targetDate, g.Rationale, g.ActionPlan, g.UpdatedAt)
	if err != nil {
		return g, err
	}
	if cmd.RowsAffected() == 0 {
		return g, ErrNotFound
	}
	return s.Get(ctx, g.ID)
}

func (s *PostgresStore) SetStatus(ctx context.Context, id string, status Status, completedAt *time.Time) (Goal, error) {
	now := time.Now().UTC()
	cmd, err := s.pool.Exec(ctx, `
		UPDATE goals SET status = $2, completed_at = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`, id, status, completedAt, now)
	if err != nil {
		return Goal{}, err
	}
	if cmd.RowsAffected() == 0 {
		return Goal{}, ErrNotFound
	}
	return s.Get(ctx, id)
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	now := time.Now().UTC()
	cmd, err := s.pool.Exec(ctx, `UPDATE goals SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`, id, now)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Memory
// ─────────────────────────────────────────────────────────────────────────────

type MemoryStore struct {
	goals map[string]Goal
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{goals: map[string]Goal{}}
}

func (s *MemoryStore) List(_ context.Context, includeArchived bool) ([]Goal, error) {
	out := make([]Goal, 0, len(s.goals))
	for _, g := range s.goals {
		if !includeArchived && g.Status == StatusArchived {
			continue
		}
		out = append(out, g)
	}
	return out, nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (Goal, error) {
	if g, ok := s.goals[id]; ok {
		return g, nil
	}
	return Goal{}, ErrNotFound
}

func (s *MemoryStore) Create(_ context.Context, g Goal) (Goal, error) {
	if g.ID == "" {
		g.ID = newID()
	}
	if g.Status == "" {
		g.Status = StatusActive
	}
	now := time.Now().UTC()
	g.CreatedAt = now
	g.UpdatedAt = now
	s.goals[g.ID] = g
	return g, nil
}

func (s *MemoryStore) Update(_ context.Context, g Goal) (Goal, error) {
	if _, ok := s.goals[g.ID]; !ok {
		return Goal{}, ErrNotFound
	}
	g.UpdatedAt = time.Now().UTC()
	s.goals[g.ID] = g
	return g, nil
}

func (s *MemoryStore) SetStatus(_ context.Context, id string, status Status, completedAt *time.Time) (Goal, error) {
	g, ok := s.goals[id]
	if !ok {
		return Goal{}, ErrNotFound
	}
	g.Status = status
	g.CompletedAt = completedAt
	g.UpdatedAt = time.Now().UTC()
	s.goals[id] = g
	return g, nil
}

func (s *MemoryStore) Delete(_ context.Context, id string) error {
	if _, ok := s.goals[id]; !ok {
		return ErrNotFound
	}
	delete(s.goals, id)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

func newID() string {
	var b [9]byte
	_, _ = rand.Read(b[:])
	return "goal_" + hex.EncodeToString(b[:])
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("startDate is required")
	}
	return time.ParseInLocation("2006-01-02", s, time.UTC)
}

func parseNullableDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, time.UTC); err == nil {
		return &t
	}
	return nil
}

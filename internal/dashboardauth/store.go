package dashboardauth

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSetupAlreadyDone   = errors.New("setup already completed")
	ErrSessionExpired     = errors.New("session expired")
	ErrRoleNotAllowed     = errors.New("role not allowed")
)

const (
	RoleAdmin  = "admin"
	RoleDoctor = "doctor"
)

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"displayName"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"-"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Session struct {
	ID         string
	UserID     string
	TokenHash  string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	LastSeenAt time.Time
}

type UserStore interface {
	HasUsers(ctx context.Context) (bool, error)
	CreateUser(ctx context.Context, user User) (User, error)
	ListUsers(ctx context.Context) ([]User, error)
	GetUserByUsername(ctx context.Context, username string) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	UpdateUser(ctx context.Context, user User) (User, error)
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	CreateSession(ctx context.Context, session Session) (Session, error)
	GetSessionByHash(ctx context.Context, tokenHash string) (Session, error)
	DeleteSessionByHash(ctx context.Context, tokenHash string) error
	DeleteSessionsByUserID(ctx context.Context, userID string) error
	TouchSession(ctx context.Context, tokenHash string, seenAt time.Time) error
}

func NormalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func NormalizeRole(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

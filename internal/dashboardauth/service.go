package dashboardauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const SessionCookieName = "bscout_session"
const SettingInstallAPISecret = "install_api_secret"

type UserSummary struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	Role        string    `json:"role"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Service struct {
	store      UserStore
	sessionTTL time.Duration
}

func NewService(store UserStore) *Service {
	return &Service{
		store:      store,
		sessionTTL: 30 * 24 * time.Hour,
	}
}

func (s *Service) SetupStatus(ctx context.Context) (bool, error) {
	return s.store.HasUsers(ctx)
}

func (s *Service) Bootstrap(ctx context.Context, username, password, displayName, configuredAPISecret string) (UserSummary, string, string, error) {
	hasUsers, err := s.store.HasUsers(ctx)
	if err != nil {
		return UserSummary{}, "", "", err
	}
	if hasUsers {
		return UserSummary{}, "", "", ErrSetupAlreadyDone
	}
	user, token, err := s.createUserAndSession(ctx, username, password, displayName, RoleAdmin)
	if err != nil {
		return UserSummary{}, "", "", err
	}
	apiSecret, err := s.EnsureInstallAPISecret(ctx, configuredAPISecret)
	if err != nil {
		return UserSummary{}, "", "", err
	}
	return user, token, apiSecret, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (UserSummary, string, error) {
	user, err := s.store.GetUserByUsername(ctx, NormalizeUsername(username))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return UserSummary{}, "", ErrInvalidCredentials
		}
		return UserSummary{}, "", err
	}
	if !user.Active {
		return UserSummary{}, "", ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return UserSummary{}, "", ErrInvalidCredentials
	}
	token, session, err := generateSession(user.ID, s.sessionTTL)
	if err != nil {
		return UserSummary{}, "", err
	}
	if _, err := s.store.CreateSession(ctx, session); err != nil {
		return UserSummary{}, "", err
	}
	return toSummary(user), token, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	return s.store.DeleteSessionByHash(ctx, hashToken(token))
}

func (s *Service) CurrentUser(ctx context.Context, token string) (UserSummary, error) {
	session, user, err := s.lookupSession(ctx, token)
	if err != nil {
		return UserSummary{}, err
	}
	if err := s.store.TouchSession(ctx, session.TokenHash, time.Now().UTC()); err != nil {
		return UserSummary{}, err
	}
	return toSummary(user), nil
}

func (s *Service) RequireRole(ctx context.Context, token, role string) (UserSummary, error) {
	user, err := s.CurrentUser(ctx, token)
	if err != nil {
		return UserSummary{}, err
	}
	if user.Role != NormalizeRole(role) && user.Role != RoleAdmin {
		return UserSummary{}, ErrRoleNotAllowed
	}
	return user, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]UserSummary, error) {
	users, err := s.store.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UserSummary, 0, len(users))
	for _, user := range users {
		out = append(out, toSummary(user))
	}
	return out, nil
}

func (s *Service) CreateUser(ctx context.Context, username, password, displayName, role string) (UserSummary, error) {
	user, err := s.createUser(ctx, username, password, displayName, role)
	if err != nil {
		return UserSummary{}, err
	}
	return toSummary(user), nil
}

func (s *Service) UpdateUser(ctx context.Context, id, displayName, role string, active *bool, password string) (UserSummary, error) {
	user, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return UserSummary{}, err
	}
	if strings.TrimSpace(displayName) != "" {
		user.DisplayName = strings.TrimSpace(displayName)
	}
	if strings.TrimSpace(role) != "" {
		role = NormalizeRole(role)
		if !isAllowedRole(role) {
			return UserSummary{}, ErrRoleNotAllowed
		}
		user.Role = role
	}
	if active != nil {
		user.Active = *active
	}
	if strings.TrimSpace(password) != "" {
		hash, err := hashPassword(password)
		if err != nil {
			return UserSummary{}, err
		}
		user.PasswordHash = hash
	}
	user.UpdatedAt = time.Now().UTC()
	updated, err := s.store.UpdateUser(ctx, user)
	if err != nil {
		return UserSummary{}, err
	}
	if !updated.Active {
		_ = s.store.DeleteSessionsByUserID(ctx, updated.ID)
	}
	return toSummary(updated), nil
}

func (s *Service) SetSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   int(s.sessionTTL.Seconds()),
		Expires:  time.Now().Add(s.sessionTTL),
	})
}

func (s *Service) CurrentInstallAPISecret(ctx context.Context) (string, error) {
	secret, err := s.store.GetSetting(ctx, SettingInstallAPISecret)
	if errors.Is(err, ErrNotFound) {
		return "", nil
	}
	return secret, err
}

func (s *Service) EnsureInstallAPISecret(ctx context.Context, configuredAPISecret string) (string, error) {
	current, err := s.store.GetSetting(ctx, SettingInstallAPISecret)
	if err == nil && strings.TrimSpace(current) != "" {
		return current, nil
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return "", err
	}
	secret := normalizeInstallAPISecret(configuredAPISecret)
	if secret == "" {
		secret, err = generateInstallAPISecret()
		if err != nil {
			return "", err
		}
	}
	if err := s.store.SetSetting(ctx, SettingInstallAPISecret, secret); err != nil {
		return "", err
	}
	return secret, nil
}

func (s *Service) ClearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func SessionTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func (s *Service) createUserAndSession(ctx context.Context, username, password, displayName, role string) (UserSummary, string, error) {
	created, err := s.createUser(ctx, username, password, displayName, role)
	if err != nil {
		return UserSummary{}, "", err
	}
	token, session, err := generateSession(created.ID, s.sessionTTL)
	if err != nil {
		return UserSummary{}, "", err
	}
	if _, err := s.store.CreateSession(ctx, session); err != nil {
		return UserSummary{}, "", err
	}
	return toSummary(created), token, nil
}

func normalizeInstallAPISecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "change-me") {
		return ""
	}
	return value
}

func generateInstallAPISecret() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *Service) createUser(ctx context.Context, username, password, displayName, role string) (User, error) {
	normalizedUsername := NormalizeUsername(username)
	if normalizedUsername == "" || len(normalizedUsername) < 3 {
		return User{}, fmt.Errorf("username must be at least 3 characters")
	}
	if len(strings.TrimSpace(password)) < 8 {
		return User{}, fmt.Errorf("password must be at least 8 characters")
	}
	role = NormalizeRole(role)
	if role == "" {
		role = RoleDoctor
	}
	if !isAllowedRole(role) {
		return User{}, ErrRoleNotAllowed
	}
	hash, err := hashPassword(password)
	if err != nil {
		return User{}, err
	}
	now := time.Now().UTC()
	if strings.TrimSpace(displayName) == "" {
		displayName = normalizedUsername
	}
	user := User{
		ID:           uuid.NewString(),
		Username:     normalizedUsername,
		DisplayName:  strings.TrimSpace(displayName),
		Role:         role,
		PasswordHash: hash,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	created, err := s.store.CreateUser(ctx, user)
	return created, err
}

func (s *Service) lookupSession(ctx context.Context, token string) (Session, User, error) {
	if strings.TrimSpace(token) == "" {
		return Session{}, User{}, ErrInvalidCredentials
	}
	session, err := s.store.GetSessionByHash(ctx, hashToken(token))
	if err != nil {
		return Session{}, User{}, ErrInvalidCredentials
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		_ = s.store.DeleteSessionByHash(ctx, session.TokenHash)
		return Session{}, User{}, ErrSessionExpired
	}
	user, err := s.store.GetUserByID(ctx, session.UserID)
	if err != nil {
		return Session{}, User{}, ErrInvalidCredentials
	}
	if !user.Active {
		return Session{}, User{}, ErrInvalidCredentials
	}
	return session, user, nil
}

func toSummary(user User) UserSummary {
	return UserSummary{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Active:      user.Active,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

func hashPassword(password string) (string, error) {
	value, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(value), nil
}

func generateSession(userID string, ttl time.Duration) (string, Session, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", Session{}, err
	}
	token := hex.EncodeToString(raw)
	now := time.Now().UTC()
	return token, Session{
		ID:         uuid.NewString(),
		UserID:     userID,
		TokenHash:  hashToken(token),
		ExpiresAt:  now.Add(ttl),
		CreatedAt:  now,
		LastSeenAt: now,
	}, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func isAllowedRole(role string) bool {
	switch NormalizeRole(role) {
	case RoleAdmin, RoleDoctor:
		return true
	default:
		return false
	}
}

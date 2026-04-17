package auth

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	nsauth "github.com/glycoview/nightscout-api/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrMissingCredentials = nsauth.ErrMissingCredentials
	ErrBadToken           = nsauth.ErrBadToken
	ErrBadJWT             = nsauth.ErrBadJWT
)

type Role = nsauth.Role
type Subject = nsauth.Subject
type Identity = nsauth.Identity

type Manager struct {
	mu           sync.RWMutex
	apiSecret    string
	apiSecretSHA string
	jwtSecret    []byte
	tokenTTL     time.Duration
	defaultRoles []string
	roles        map[string]Role
	subjects     map[string]Subject
}

func New(apiSecret string, defaultRoles []string, jwtSecret string) *Manager {
	if jwtSecret == "" {
		jwtSecret = apiSecret
	}
	sum := sha1.Sum([]byte(apiSecret))
	manager := &Manager{
		apiSecret:    apiSecret,
		apiSecretSHA: hex.EncodeToString(sum[:]),
		jwtSecret:    []byte(jwtSecret),
		tokenTTL:     12 * time.Hour,
		defaultRoles: append([]string(nil), defaultRoles...),
		roles:        map[string]Role{},
		subjects:     map[string]Subject{},
	}
	manager.bootstrap()
	return manager
}

func (m *Manager) bootstrap() {
	_ = m.CreateRole("admin", "*")
	_ = m.CreateRole("readable", "*:*:read")
	_ = m.CreateRole("apiAll", "api:*:*")
	_ = m.CreateRole("apiAdmin", "api:*:admin")
	_ = m.CreateRole("apiCreate", "api:*:create")
	_ = m.CreateRole("apiRead", "api:*:read")
	_ = m.CreateRole("apiUpdate", "api:*:update")
	_ = m.CreateRole("apiDelete", "api:*:delete")
	_ = m.CreateRole("denied", "")
}

func (m *Manager) CreateRole(name string, permissions ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.roles[name] = Role{Name: name, Permissions: append([]string(nil), permissions...)}
	return nil
}

func (m *Manager) CreateSubject(name string, roles []string) Subject {
	m.mu.Lock()
	defer m.mu.Unlock()
	subject := Subject{
		Name:        name,
		Roles:       append([]string(nil), roles...),
		AccessToken: uuid.NewString(),
	}
	m.subjects[subject.AccessToken] = subject
	return subject
}

func (m *Manager) UpdateAPISecret(apiSecret string) {
	apiSecret = strings.TrimSpace(apiSecret)
	if apiSecret == "" {
		return
	}
	sum := sha1.Sum([]byte(apiSecret))

	m.mu.Lock()
	defer m.mu.Unlock()

	reuseJWTSecret := string(m.jwtSecret) == m.apiSecret
	m.apiSecret = apiSecret
	m.apiSecretSHA = hex.EncodeToString(sum[:])
	if reuseJWTSecret {
		m.jwtSecret = []byte(apiSecret)
	}
}

func (m *Manager) IssueJWT(accessToken string) (string, error) {
	var identity Identity
	if accessToken == m.apiSecret || accessToken == m.apiSecretSHA {
		identity = m.identityForRoles("api-secret", []string{"admin"}, false)
	} else {
		m.mu.RLock()
		subject, ok := m.subjects[accessToken]
		m.mu.RUnlock()
		if !ok {
			return "", errors.New("unknown access token")
		}
		identity = m.identityForSubject(subject, false)
	}
	claims := jwt.MapClaims{
		"accessToken": accessToken,
		"sub":         identity.Name,
		"roles":       identity.Roles,
		"iat":         time.Now().Unix(),
		"exp":         time.Now().Add(m.tokenTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.jwtSecret)
}

func (m *Manager) AuthenticateRequest(r *http.Request) (*Identity, error) {
	if bearer := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer")); bearer != "" && bearer != r.Header.Get("Authorization") {
		return m.authenticateJWT(bearer)
	}
	if secret := strings.TrimSpace(r.Header.Get("api-secret")); secret != "" {
		return m.authenticateToken(secret)
	}
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		return m.authenticateToken(token)
	}
	if len(m.defaultRoles) == 0 {
		return nil, ErrMissingCredentials
	}
	identity := m.identityForRoles("default", m.defaultRoles, true)
	return &identity, nil
}

func (m *Manager) Require(permission string, allowDefault bool, next func(http.ResponseWriter, *http.Request, *Identity)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := m.AuthenticateRequest(r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"status": http.StatusUnauthorized, "message": authErrorMessage(err)})
			return
		}
		if identity.FromDefault && !allowDefault {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"status": http.StatusUnauthorized, "message": "Missing or bad access token or JWT"})
			return
		}
		if !m.HasPermission(*identity, permission) {
			if identity.FromDefault {
				writeJSON(w, http.StatusUnauthorized, map[string]any{"status": http.StatusUnauthorized, "message": "Missing or bad access token or JWT"})
				return
			}
			writeJSON(w, http.StatusForbidden, map[string]any{"status": http.StatusForbidden, "message": fmt.Sprintf("Missing permission %s", permission)})
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), identityContextKey{}, identity)), identity)
	}
}

func (m *Manager) RequireV1Write(permission string, next func(http.ResponseWriter, *http.Request, *Identity)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := m.AuthenticateExplicit(r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"status": http.StatusUnauthorized, "message": authErrorMessage(err)})
			return
		}
		if !m.HasPermission(*identity, permission) {
			writeJSON(w, http.StatusForbidden, map[string]any{"status": http.StatusForbidden, "message": fmt.Sprintf("Missing permission %s", permission)})
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), identityContextKey{}, identity)), identity)
	}
}

func (m *Manager) HasPermission(identity Identity, permission string) bool {
	for _, candidate := range identity.Permissions {
		if permissionMatch(candidate, permission) {
			return true
		}
	}
	return false
}

func IdentityFromContext(ctx context.Context) *Identity {
	if value, ok := ctx.Value(identityContextKey{}).(*Identity); ok {
		return value
	}
	return nil
}

func (m *Manager) authenticateToken(token string) (*Identity, error) {
	if token == m.apiSecret || token == m.apiSecretSHA {
		identity := m.identityForRoles("api-secret", []string{"admin"}, false)
		identity.AccessToken = token
		return &identity, nil
	}
	m.mu.RLock()
	subject, ok := m.subjects[token]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrBadToken
	}
	identity := m.identityForSubject(subject, false)
	return &identity, nil
}

func (m *Manager) authenticateJWT(tokenString string) (*Identity, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return m.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrBadJWT
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrBadJWT
	}
	accessToken, _ := claims["accessToken"].(string)
	if accessToken == m.apiSecret || accessToken == m.apiSecretSHA {
		identity := m.identityForRoles("api-secret", []string{"admin"}, false)
		return &identity, nil
	}
	m.mu.RLock()
	subject, ok := m.subjects[accessToken]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrBadJWT
	}
	identity := m.identityForSubject(subject, false)
	return &identity, nil
}

func (m *Manager) AuthenticateExplicit(r *http.Request) (*Identity, error) {
	if bearer := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer")); bearer != "" && bearer != r.Header.Get("Authorization") {
		return m.authenticateJWT(bearer)
	}
	if secret := strings.TrimSpace(r.Header.Get("api-secret")); secret != "" {
		return m.authenticateToken(secret)
	}
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		return m.authenticateToken(token)
	}
	return nil, ErrMissingCredentials
}

func (m *Manager) identityForSubject(subject Subject, fromDefault bool) Identity {
	return m.identityForRoles(subject.Name, subject.Roles, fromDefault)
}

func (m *Manager) identityForRoles(name string, roles []string, fromDefault bool) Identity {
	m.mu.RLock()
	defer m.mu.RUnlock()

	permissions := make([]string, 0, len(roles))
	for _, roleName := range roles {
		role, ok := m.roles[roleName]
		if !ok {
			continue
		}
		permissions = append(permissions, role.Permissions...)
	}
	return Identity{
		Name:        name,
		Roles:       append([]string(nil), roles...),
		Permissions: permissions,
		FromDefault: fromDefault,
	}
}

func permissionMatch(candidate, target string) bool {
	if candidate == "*" {
		return true
	}
	want := strings.Split(target, ":")
	have := strings.Split(candidate, ":")
	if len(have) != len(want) {
		return candidate == target
	}
	for i := range want {
		if have[i] == "*" {
			continue
		}
		if have[i] != want[i] {
			return false
		}
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = fmt.Fprint(w, marshalJSON(body))
}

func marshalJSON(v any) string {
	buf, _ := json.Marshal(v)
	return string(buf)
}

func authErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrBadToken), errors.Is(err, ErrBadJWT):
		return "Bad access token or JWT"
	default:
		return "Missing or bad access token or JWT"
	}
}

type identityContextKey struct{}

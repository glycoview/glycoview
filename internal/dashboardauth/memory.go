package dashboardauth

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	mu       sync.RWMutex
	users    map[string]User
	byUser   map[string]string
	settings map[string]string
	sessions map[string]Session
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:    map[string]User{},
		byUser:   map[string]string{},
		settings: map[string]string{},
		sessions: map[string]Session{},
	}
}

func (s *MemoryStore) HasUsers(_ context.Context) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users) > 0, nil
}

func (s *MemoryStore) CreateUser(_ context.Context, user User) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byUser[user.Username]; exists {
		return User{}, ErrInvalidCredentials
	}
	s.users[user.ID] = user
	s.byUser[user.Username] = user.ID
	return user, nil
}

func (s *MemoryStore) ListUsers(_ context.Context) ([]User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]User, 0, len(s.users))
	for _, user := range s.users {
		out = append(out, user)
	}
	return out, nil
}

func (s *MemoryStore) GetUserByUsername(_ context.Context, username string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byUser[username]
	if !ok {
		return User{}, ErrNotFound
	}
	user, ok := s.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (s *MemoryStore) GetUserByID(_ context.Context, id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (s *MemoryStore) UpdateUser(_ context.Context, user User) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.users[user.ID]
	if !ok {
		return User{}, ErrNotFound
	}
	if old.Username != user.Username {
		delete(s.byUser, old.Username)
		s.byUser[user.Username] = user.ID
	}
	s.users[user.ID] = user
	return user, nil
}

func (s *MemoryStore) GetSetting(_ context.Context, key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.settings[key]
	if !ok {
		return "", ErrNotFound
	}
	return value, nil
}

func (s *MemoryStore) SetSetting(_ context.Context, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings[key] = value
	return nil
}

func (s *MemoryStore) CreateSession(_ context.Context, session Session) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.TokenHash] = session
	return session, nil
}

func (s *MemoryStore) GetSessionByHash(_ context.Context, tokenHash string) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[tokenHash]
	if !ok {
		return Session{}, ErrNotFound
	}
	return session, nil
}

func (s *MemoryStore) DeleteSessionByHash(_ context.Context, tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, tokenHash)
	return nil
}

func (s *MemoryStore) DeleteSessionsByUserID(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, session := range s.sessions {
		if session.UserID == userID {
			delete(s.sessions, key)
		}
	}
	return nil
}

func (s *MemoryStore) TouchSession(_ context.Context, tokenHash string, seenAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[tokenHash]
	if !ok {
		return ErrNotFound
	}
	session.LastSeenAt = seenAt
	s.sessions[tokenHash] = session
	return nil
}

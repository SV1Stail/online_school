package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/SV1Stail/online_school/internal/security"
	"github.com/SV1Stail/online_school/internal/user"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrTooManyAttempts    = errors.New("too many login attempts")
)

type SessionRepository interface {
	CreateSession(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) error
	FindSession(ctx context.Context, tokenHash []byte, now time.Time) (*user.User, error)
	DeleteSession(ctx context.Context, tokenHash []byte) error
	DeleteUserSessions(ctx context.Context, userID string) error
}

type UserRepository interface {
	FindByLogin(ctx context.Context, login string) (*user.Credentials, error)
}

type LoginLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	Reset(ctx context.Context, key string) error
}

type Service struct {
	users    UserRepository
	sessions SessionRepository
	limiter  LoginLimiter
	ttl      time.Duration
	now      func() time.Time
}

type LoginResult struct {
	User      *user.User
	Token     string
	ExpiresAt time.Time
}

func NewService(
	users UserRepository,
	sessions SessionRepository,
	limiter LoginLimiter,
	ttl time.Duration,
) *Service {
	return &Service{users: users,
		sessions: sessions,
		limiter:  limiter, ttl: ttl,
		now: time.Now,
	}
}

func (s *Service) Login(ctx context.Context, login, password, clientKey string) (*LoginResult, error) {
	login = strings.ToLower(strings.TrimSpace(login))
	limitKey := clientKey + ":" + login
	allowed, err := s.limiter.Allow(ctx, limitKey)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrTooManyAttempts
	}

	credentials, err := s.users.FindByLogin(ctx, login)
	if errors.Is(err, user.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	valid, err := security.VerifyPassword(password, credentials.PasswordHash)
	if err != nil || !valid || !credentials.Active {
		return nil, ErrInvalidCredentials
	}

	token, tokenHash, err := newSessionToken()
	if err != nil {
		return nil, err
	}
	expiresAt := s.now().UTC().Add(s.ttl)
	if err := s.sessions.CreateSession(ctx, credentials.ID, tokenHash, expiresAt); err != nil {
		return nil, err
	}
	if err := s.limiter.Reset(ctx, limitKey); err != nil {
		return nil, err
	}

	return &LoginResult{User: &credentials.User, Token: token, ExpiresAt: expiresAt}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (*user.User, error) {
	if token == "" {
		return nil, ErrUnauthorized
	}
	return s.sessions.FindSession(ctx, hashToken(token), s.now().UTC())
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.sessions.DeleteSession(ctx, hashToken(token))
}

func newSessionToken() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, hashToken(token), nil
}

func hashToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

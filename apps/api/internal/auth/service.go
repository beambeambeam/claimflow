package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
)

const (
	sessionLifetime  = 8 * time.Hour
	sessionTokenSize = 32
	// dummyPassword is hashed once per Service. Login compares against it when the username is unknown.
	dummyPassword = "claimflow-dummy-password"
)

var (
	// ErrInvalidCredentials means the username is unknown or the password does not match. Callers must not
	// tell the two apart.
	ErrInvalidCredentials = errors.New("auth: invalid username or password")
	// ErrNoSession means the token belongs to no unexpired session of a user who is not banned.
	ErrNoSession = errors.New("auth: no active session")
)

// User is the signed-in user of a session.
type User struct {
	ID       uuid.UUID
	Username string
	Name     string
	Role     string
}

// Service holds the auth rules: credential check, session start, and session lookup (SDS §6).
type Service struct {
	queries   *db.Queries
	dummyHash []byte
	now       func() time.Time
}

// NewService returns a Service that keeps sessions in queries. bcryptCost is the cost of every hash.
// now is the clock for session and ban times; tests inject a fixed one.
func NewService(queries *db.Queries, bcryptCost int, now func() time.Time) Service {
	dummyHash, err := bcrypt.GenerateFromPassword([]byte(dummyPassword), bcryptCost)
	if err != nil { // coverage-ignore
		panic(fmt.Sprintf("auth: hash dummy password: %v", err))
	}

	return Service{queries: queries, dummyHash: dummyHash, now: now}
}

// Login checks the credential and starts a session. It returns the user and the token for the cookie.
// The database keeps only the token's SHA-256 digest.
func (s Service) Login(ctx context.Context, username, password string) (User, string, error) {
	credential, err := s.queries.GetCredentialForLogin(ctx, strings.ToLower(username))
	if errors.Is(err, pgx.ErrNoRows) {
		// Spend the same bcrypt time as a wrong password, so the response time does not reveal the username.
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))

		return User{}, "", ErrInvalidCredentials
	}
	if err != nil { // coverage-ignore
		return User{}, "", fmt.Errorf("get credential: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(credential.PasswordHash), []byte(password)); err != nil {
		return User{}, "", ErrInvalidCredentials
	}

	token, err := s.startSession(ctx, credential.UserID)
	if err != nil { // coverage-ignore
		return User{}, "", err
	}

	return User{ID: credential.UserID, Username: credential.Username, Name: credential.Name, Role: credential.Role}, token, nil
}

// SessionUser returns the user of the session that token names.
func (s Service) SessionUser(ctx context.Context, token string) (User, error) {
	digest := sha256.Sum256([]byte(token))

	user, err := s.queries.GetSessionUser(ctx, digest[:])
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNoSession
	}
	if err != nil { // coverage-ignore
		return User{}, fmt.Errorf("get session user: %w", err)
	}

	return User{ID: user.ID, Username: user.Username, Name: user.Name, Role: user.Role}, nil
}

// startSession stores a new session for userID and returns the token that goes in the cookie.
func (s Service) startSession(ctx context.Context, userID uuid.UUID) (string, error) {
	raw := make([]byte, sessionTokenSize)
	if _, err := rand.Read(raw); err != nil { // coverage-ignore
		return "", fmt.Errorf("session token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(token))

	err := s.queries.CreateSession(ctx, db.CreateSessionParams{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    userID,
		Token:     digest[:],
		ExpiresAt: pgtype.Timestamptz{Time: s.now().Add(sessionLifetime), Valid: true},
	})
	if err != nil { // coverage-ignore
		return "", fmt.Errorf("create session: %w", err)
	}

	return token, nil
}

// Package auth implements the sign-in and current-user operations (F-01).
//
// GetMe is a stub until T3 implements it.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

const (
	sessionLifetime  = 8 * time.Hour
	sessionTokenSize = 32
	// dummyPassword is hashed once per Handler. Login compares against it when the username is unknown.
	dummyPassword = "claimflow-dummy-password"
)

var errInvalidCredentials = &httperr.Error{
	Status: http.StatusUnauthorized,
	Code:   api.ProblemCodeUnauthenticated,
	Title:  "Invalid username or password.",
}

// Handler implements the auth operations of api.StrictServerInterface.
type Handler struct {
	queries      *db.Queries
	cookieSecure bool
	dummyHash    []byte
}

// NewHandler returns a Handler that keeps sessions in queries. When cookieSecure is set, the session
// cookie gets the Secure attribute and the __Host- prefix. bcryptCost is the cost of every hash.
func NewHandler(queries *db.Queries, cookieSecure bool, bcryptCost int) Handler {
	dummyHash, err := bcrypt.GenerateFromPassword([]byte(dummyPassword), bcryptCost)
	if err != nil { // coverage-ignore
		panic(fmt.Sprintf("auth: hash dummy password: %v", err))
	}

	return Handler{queries: queries, cookieSecure: cookieSecure, dummyHash: dummyHash}
}

// Login checks the credential and starts a session. The cookie carries the token. The database keeps
// only its SHA-256 digest.
func (h Handler) Login(ctx context.Context, request api.LoginRequestObject) (api.LoginResponseObject, error) {
	if !validLogin(request.Body.Username, request.Body.Password) {
		return nil, httperr.ErrValidation
	}

	password := []byte(request.Body.Password)

	credential, err := h.queries.GetCredentialForLogin(ctx, strings.ToLower(request.Body.Username))
	if errors.Is(err, pgx.ErrNoRows) {
		// Spend the same bcrypt time as a wrong password, so the response time does not reveal the username.
		_ = bcrypt.CompareHashAndPassword(h.dummyHash, password)

		return nil, errInvalidCredentials
	}
	if err != nil { // coverage-ignore
		return nil, fmt.Errorf("get credential: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(credential.PasswordHash), password); err != nil {
		return nil, errInvalidCredentials
	}

	token, err := h.startSession(ctx, credential.UserID)
	if err != nil { // coverage-ignore
		return nil, err
	}

	cookie := h.sessionCookie(token)

	return api.Login200JSONResponse{
		Body: api.CurrentUserEnvelope{Data: api.CurrentUser{
			Id:          credential.UserID.String(),
			Name:        credential.Name,
			Username:    credential.Username,
			Role:        api.CurrentUserRole(credential.Role),
			Permissions: roles[api.CurrentUserRole(credential.Role)],
		}},
		Headers: api.Login200ResponseHeaders{SetCookie: &cookie},
	}, nil
}

// startSession stores a new session for userID and returns the token that goes in the cookie.
func (h Handler) startSession(ctx context.Context, userID uuid.UUID) (string, error) {
	raw := make([]byte, sessionTokenSize)
	if _, err := rand.Read(raw); err != nil { // coverage-ignore
		return "", fmt.Errorf("session token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(token))

	err := h.queries.CreateSession(ctx, db.CreateSessionParams{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    userID,
		Token:     digest[:],
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(sessionLifetime), Valid: true},
	})
	if err != nil { // coverage-ignore
		return "", fmt.Errorf("create session: %w", err)
	}

	return token, nil
}

// sessionCookie builds the Set-Cookie value for token. Attributes follow F-01 §1.5.
func (h Handler) sessionCookie(token string) string {
	name, secure := "sid", ""
	if h.cookieSecure {
		name, secure = "__Host-sid", "Secure; "
	}

	return fmt.Sprintf("%s=%s; HttpOnly; %sSameSite=Lax; Path=/; Max-Age=%d",
		name, token, secure, int(sessionLifetime/time.Second))
}

// validLogin reports whether a login request is within the bounds of LoginRequest:
// username 1–50 characters, password 1–72 bytes.
func validLogin(username, password string) bool {
	length := utf8.RuneCountInString(username)

	return length >= 1 && length <= 50 && len(password) >= 1 && len(password) <= 72
}

// GetMe returns the user of the current session. Not implemented yet.
func (Handler) GetMe(context.Context, api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	return nil, httperr.ErrNotImplemented
}

// Package auth implements the sign-in and current-user operations (F-01).
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

	"github.com/gin-gonic/gin"
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

var (
	errInvalidCredentials = &httperr.Error{
		Status: http.StatusUnauthorized,
		Code:   api.ProblemCodeUnauthenticated,
		Title:  "Invalid username or password.",
	}
	errUnauthenticated = &httperr.Error{
		Status: http.StatusUnauthorized,
		Code:   api.ProblemCodeUnauthenticated,
		Title:  "Authentication required.",
	}
)

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
	if fields := loginFieldErrors(request.Body.Username, request.Body.Password); len(fields) > 0 {
		return nil, httperr.InvalidFields(fields)
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
		Body:    api.CurrentUserEnvelope{Data: currentUser(credential.UserID, credential.Name, credential.Username, credential.Role)},
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

// cookieName returns the session cookie name. The __Host- prefix requires Secure, so it is used only with it.
func (h Handler) cookieName() string {
	if h.cookieSecure {
		return "__Host-sid"
	}

	return "sid"
}

// sessionCookie builds the Set-Cookie value for token. Attributes follow F-01 §1.5.
func (h Handler) sessionCookie(token string) string {
	secure := ""
	if h.cookieSecure {
		secure = "Secure; "
	}

	return fmt.Sprintf("%s=%s; HttpOnly; %sSameSite=Lax; Path=/; Max-Age=%d",
		h.cookieName(), token, secure, int(sessionLifetime/time.Second))
}

// currentUser builds the body that describes a user, with the permissions of their role.
func currentUser(id uuid.UUID, name, username, role string) api.CurrentUser {
	return api.CurrentUser{
		Id:          id.String(),
		Name:        name,
		Username:    username,
		Role:        api.CurrentUserRole(role),
		Permissions: roles[api.CurrentUserRole(role)],
	}
}

// loginFieldErrors names each field of a login request that breaks its bound in LoginRequest:
// username 1–50 characters, password 1–72 bytes. It returns nil when both fields are within bounds.
func loginFieldErrors(username, password string) []api.FieldError {
	var fields []api.FieldError

	switch length := utf8.RuneCountInString(username); {
	case length == 0:
		fields = append(fields, api.FieldError{Field: "username", Rule: "required", Message: "Enter your username."})
	case length > 50:
		fields = append(fields, api.FieldError{Field: "username", Rule: "length", Message: "Username must be 1–50 characters."})
	}

	switch {
	case password == "":
		fields = append(fields, api.FieldError{Field: "password", Rule: "required", Message: "Enter your password."})
	case len(password) > 72:
		fields = append(fields, api.FieldError{Field: "password", Rule: "length", Message: "Password must be 1–72 bytes."})
	}

	return fields
}

// GetMe returns the user of the session in the request cookie. The strict server passes the gin context
// as ctx, and gin exposes the request through it.
func (h Handler) GetMe(ctx context.Context, _ api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	request := ctx.Value(gin.ContextRequestKey).(*http.Request)

	cookie, err := request.Cookie(h.cookieName())
	if err != nil {
		return nil, errUnauthenticated
	}

	digest := sha256.Sum256([]byte(cookie.Value))

	user, err := h.queries.GetSessionUser(ctx, digest[:])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errUnauthenticated
	}
	if err != nil { // coverage-ignore
		return nil, fmt.Errorf("get session user: %w", err)
	}

	return api.GetMe200JSONResponse{Data: currentUser(user.ID, user.Name, user.Username, user.Role)}, nil
}

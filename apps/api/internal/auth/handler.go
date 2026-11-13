// Package auth implements sign-in, sessions, and the current user (F-01).
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
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

// Handler implements the auth operations of api.StrictServerInterface. It maps requests to Service calls and
// Service errors to problem responses.
type Handler struct {
	service      Service
	cookieSecure bool
}

// NewHandler returns a Handler over service. When cookieSecure is set, the session cookie gets the Secure
// attribute and the __Host- prefix.
func NewHandler(service Service, cookieSecure bool) Handler {
	return Handler{service: service, cookieSecure: cookieSecure}
}

// Login checks the request's bounds, then logs in and sets the session cookie.
func (h Handler) Login(ctx context.Context, request api.LoginRequestObject) (api.LoginResponseObject, error) {
	if fields := loginFieldErrors(request.Body.Username, request.Body.Password); len(fields) > 0 {
		return nil, fmt.Errorf("login fields: %w", httperr.InvalidFields(fields))
	}

	user, token, err := h.service.Login(ctx, request.Body.Username, request.Body.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		return nil, errInvalidCredentials
	}
	if err != nil { // coverage-ignore
		return nil, err
	}

	cookie := sessionCookie(h.cookieSecure, token)

	return api.Login200JSONResponse{
		Body:    api.CurrentUserEnvelope{Data: currentUser(user)},
		Headers: api.Login200ResponseHeaders{SetCookie: &cookie},
	}, nil
}

// GetMe returns the user that the Session middleware found for the request.
func (h Handler) GetMe(ctx context.Context, _ api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	user, ok := UserFrom(ctx)
	if !ok {
		return nil, errUnauthenticated
	}

	return api.GetMe200JSONResponse{Data: currentUser(user)}, nil
}

// Logout ends the session of the request and clears the session cookie.
func (h Handler) Logout(ctx context.Context, _ api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	sessionID, ok := SessionIDFrom(ctx)
	if !ok {
		return nil, errUnauthenticated
	}

	if err := h.service.Logout(ctx, sessionID); err != nil { // coverage-ignore
		return nil, err
	}

	cookie := clearedCookie(h.cookieSecure)

	return api.Logout204Response{Headers: api.Logout204ResponseHeaders{SetCookie: &cookie}}, nil
}

// cookieName returns the session cookie name. The __Host- prefix requires Secure, so it is used only with it.
func cookieName(secure bool) string {
	if secure {
		return "__Host-sid"
	}

	return "sid"
}

// sessionCookie builds the Set-Cookie value for token. Attributes follow F-01 §1.5.
func sessionCookie(secure bool, token string) string {
	return setCookie(secure, token, int(sessionLifetime/time.Second))
}

// clearedCookie builds the Set-Cookie value that removes the session cookie: same name, empty value, Max-Age=0.
func clearedCookie(secure bool) string {
	return setCookie(secure, "", 0)
}

// setCookie builds the Set-Cookie value of the session cookie with the given value and Max-Age in seconds.
func setCookie(secure bool, value string, maxAge int) string {
	secureAttr := ""
	if secure {
		secureAttr = "Secure; "
	}

	return fmt.Sprintf("%s=%s; HttpOnly; %sSameSite=Lax; Path=/; Max-Age=%d", cookieName(secure), value, secureAttr, maxAge)
}

// currentUser builds the body that describes a user, with the permissions of their role.
func currentUser(user User) api.CurrentUser {
	return api.CurrentUser{
		Id:          user.ID.String(),
		Name:        user.Name,
		Username:    user.Username,
		Role:        api.CurrentUserRole(user.Role),
		Permissions: permissionsByRole[api.CurrentUserRole(user.Role)],
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

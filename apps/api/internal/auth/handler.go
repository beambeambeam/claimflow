// Package auth implements the sign-in and current-user operations (F-01).
//
// The operations are stubs until T2 (login) and T3 (current user) implement them.
package auth

import (
	"context"
	"unicode/utf8"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

// Handler implements the auth operations of api.StrictServerInterface.
type Handler struct{}

// Login rejects a request outside the username and password bounds. The rest is not implemented yet.
func (Handler) Login(_ context.Context, request api.LoginRequestObject) (api.LoginResponseObject, error) {
	if !validLogin(request.Body.Username, request.Body.Password) {
		return nil, httperr.ErrValidation
	}

	return nil, httperr.ErrNotImplemented
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

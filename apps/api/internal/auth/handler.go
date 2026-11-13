// Package auth implements the sign-in and current-user operations (F-01).
//
// The operations are stubs until T2 (login) and T3 (current user) implement them.
package auth

import (
	"context"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

// Handler implements the auth operations of api.StrictServerInterface.
type Handler struct{}

// Login checks the username and password and starts a session. Not implemented yet.
func (Handler) Login(context.Context, api.LoginRequestObject) (api.LoginResponseObject, error) {
	return nil, httperr.ErrNotImplemented
}

// GetMe returns the user of the current session. Not implemented yet.
func (Handler) GetMe(context.Context, api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	return nil, httperr.ErrNotImplemented
}

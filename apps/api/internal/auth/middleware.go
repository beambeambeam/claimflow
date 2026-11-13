package auth

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

type (
	userKey    struct{}
	sessionKey struct{}
)

// Session puts the user of the request's session cookie in the request context (SDS §6). A request with no
// cookie, or with a token of no active session, continues without a user, because login and health are
// public. Operations that need a user read it with UserFrom and answer 401 when it is missing.
// The router must set ContextWithFallback, so that handlers given the gin context see the value.
func Session(service Service, cookieSecure bool) gin.HandlerFunc {
	name := cookieName(cookieSecure)

	return func(c *gin.Context) {
		cookie, err := c.Request.Cookie(name)
		if err != nil {
			c.Next()

			return
		}

		user, sessionID, err := service.SessionUser(c.Request.Context(), cookie.Value)
		if err != nil && !errors.Is(err, ErrNoSession) { // coverage-ignore
			httperr.Report(c, err)
			c.Abort()

			return
		}

		if err == nil {
			ctx := context.WithValue(c.Request.Context(), userKey{}, user)
			c.Request = c.Request.WithContext(context.WithValue(ctx, sessionKey{}, sessionID))
		}

		c.Next()
	}
}

// SessionIDFrom returns the id of the session that Session stored in ctx.
func SessionIDFrom(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(sessionKey{}).(uuid.UUID)

	return id, ok
}

// UserFrom returns the user that Session stored in ctx.
func UserFrom(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userKey{}).(User)

	return user, ok
}

// RequireSession answers 401 UNAUTHENTICATED, without calling the handler, when a request to a protected
// route has no session user. protected holds gin routes as "METHOD /full/path". It runs after Session and
// before the request is validated (F-01 §3.4, step 3), so each operation needs no check of its own.
func RequireSession(protected map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := UserFrom(c.Request.Context()); !ok && protected[c.Request.Method+" "+c.FullPath()] {
			httperr.Report(c, errUnauthenticated)
			c.Abort()

			return
		}

		c.Next()
	}
}

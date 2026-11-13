package auth

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

type userKey struct{}

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

		user, err := service.SessionUser(c.Request.Context(), cookie.Value)
		if err != nil && !errors.Is(err, ErrNoSession) { // coverage-ignore
			httperr.Report(c, err)
			c.Abort()

			return
		}

		if err == nil {
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), userKey{}, user))
		}

		c.Next()
	}
}

// UserFrom returns the user that Session stored in ctx.
func UserFrom(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userKey{}).(User)

	return user, ok
}

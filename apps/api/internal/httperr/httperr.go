// Package httperr turns errors into RFC 9457 problem responses (SDS §4.2).
//
// Handlers report an error with c.Error. Middleware then writes one problem for it.
// An *Error keeps its status, code, and title. Any other error, and any panic,
// becomes a 500 whose body has no internal text. The error itself is logged
// under the same request ID that the response carries.
package httperr

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
)

const (
	contentType = "application/problem+json"
	// RequestIDHeader carries the request ID on every response.
	RequestIDHeader = "X-Request-Id"
	requestIDKey    = "httperr.requestID"
)

// Error is an error with a known problem response.
type Error struct {
	Status int
	Code   api.ProblemCode
	Title  string
}

func (e *Error) Error() string {
	return e.Title
}

var (
	// ErrNotFound is the problem for a path with no operation.
	ErrNotFound = &Error{Status: http.StatusNotFound, Code: api.ProblemCodeNotFound, Title: "Not found."}
	// ErrUnavailable is the problem when a dependency, such as the database, cannot be reached.
	ErrUnavailable = &Error{Status: http.StatusServiceUnavailable, Code: api.ProblemCodeUnavailable, Title: "Service unavailable."}

	errInternal = &Error{Status: http.StatusInternalServerError, Code: api.ProblemCodeInternal, Title: "Something went wrong."}
)

// RequestID gives each request a new UUIDv7 ID and sends it in X-Request-Id.
// The server never trusts an incoming request ID.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.Must(uuid.NewV7()).String()
		c.Set(requestIDKey, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// RequestIDFrom returns the request ID that RequestID set.
func RequestIDFrom(c *gin.Context) string {
	return c.GetString(requestIDKey)
}

// Handle writes a problem for the first error a handler reported, and for any panic.
func Handle(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.LogAttrs(c.Request.Context(), slog.LevelError, "panic recovered",
					slog.String("requestId", RequestIDFrom(c)),
					slog.Any("panic", recovered),
				)
				write(c, errInternal)
				c.Abort()
			}
		}()

		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		err := c.Errors[0].Err

		var problem *Error
		if !errors.As(err, &problem) {
			logger.LogAttrs(c.Request.Context(), slog.LevelError, "internal error",
				slog.String("requestId", RequestIDFrom(c)),
				slog.String("error", err.Error()),
			)

			problem = errInternal
		}

		write(c, problem)
	}
}

// NotFound reports ErrNotFound. Use it as the router's NoRoute handler.
func NotFound(c *gin.Context) {
	_ = c.Error(ErrNotFound)
}

// Report records err so Handle writes its problem. It fits the generated
// strict server's error hooks.
func Report(c *gin.Context, err error) {
	_ = c.Error(err)
}

func write(c *gin.Context, problem *Error) {
	body := api.Problem{
		Type:      api.ProblemTypeAboutBlank,
		Title:     problem.Title,
		Status:    problem.Status,
		Code:      problem.Code,
		Instance:  c.Request.URL.Path,
		RequestId: RequestIDFrom(c),
	}
	c.Header("Content-Type", contentType)
	c.JSON(problem.Status, body)
}

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/claim"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

// defineCalendarDate makes the date format a real calendar date. The built-in format is a pattern, so it accepts
// 2026-02-31. kin-openapi keeps formats in a process-wide table, so the definition is made once however many routers
// are built.
var defineCalendarDate = sync.OnceFunc(func() {
	openapi3.DefineStringFormatValidator("date", openapi3.NewCallbackValidator(func(value string) error {
		_, err := time.Parse(time.DateOnly, value)
		if err != nil {
			return fmt.Errorf("not a calendar date: %w", err)
		}

		return nil
	}))
})

// maxRequestBody caps the JSON body that requestValidation reads. A claim is at most 20 items of a few hundred bytes.
const maxRequestBody = 1 << 20

// validatedOperations lists the operations whose JSON request body is checked against the contract schema before the
// handler runs (SDS §1.3, "OpenAPI request validation"), with the message of each field. The others check their own
// bounds in the handler. A shape error is a 400 that names every broken field; business limits are the module's 422.
// The key is the operation name that oapi-codegen gives the embedded spec, as in operationPermissions.
var validatedOperations = map[string]map[string]string{
	"CreateClaim": claim.ShapeMessages,
}

// defaultShapeMessage is the message of a field that has no entry in its operation's messages.
const defaultShapeMessage = "Enter a valid value."

// indexPattern matches an array index in a field path, such as [0].
var indexPattern = regexp.MustCompile(`\[\d+\]`)

// requestValidation checks the JSON body of each operation in validated against its schema in spec, and answers a
// body that does not match with a 400 whose errors[] has one entry per broken field: field path such as
// items[0].amount, the schema keyword that failed as the rule, and the message of that field. A body that is not JSON,
// or whose whole value has the wrong type, is a 400 with no field. It runs after the session check and before the
// route permission, as the contract's workflow says.
func requestValidation(spec *openapi3.T, validated map[string]map[string]string) gin.HandlerFunc {
	defineCalendarDate()

	type check struct {
		schema   *openapi3.Schema
		messages map[string]string
	}

	checks := map[string]check{}

	for path, item := range spec.Paths.Map() {
		for method, operation := range item.Operations() {
			messages, ok := validated[operation.OperationID]
			if !ok {
				continue
			}

			schema := operation.RequestBody.Value.Content.Get("application/json").Schema.Value
			checks[ginRoute(method, path)] = check{schema: schema, messages: messages}
		}
	}

	return func(c *gin.Context) {
		found, ok := checks[c.Request.Method+" "+c.FullPath()]
		if !ok {
			c.Next()

			return
		}

		raw, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBody))
		if err != nil {
			reject(c, httperr.ErrValidation)

			return
		}

		c.Request.Body = io.NopCloser(bytes.NewReader(raw))

		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			reject(c, httperr.ErrValidation)

			return
		}

		err = found.schema.VisitJSON(value, openapi3.MultiErrors())
		if err == nil {
			c.Next()

			return
		}

		if fields := shapeErrors(err, found.messages); len(fields) > 0 {
			reject(c, httperr.InvalidFields(fields))

			return
		}

		reject(c, httperr.ErrValidation)
	}
}

// reject reports problem and stops the request.
func reject(c *gin.Context, problem *httperr.Error) {
	httperr.Report(c, problem)
	c.Abort()
}

// shapeErrors turns the schema errors of err into one entry per field, in field order. An error that names no field
// is dropped.
func shapeErrors(err error, messages map[string]string) []api.FieldError {
	var fields []api.FieldError

	for _, found := range flatten(err) {
		field := fieldPath(found.JSONPointer())
		if field == "" {
			continue
		}

		message, ok := messages[indexPattern.ReplaceAllString(field, "[]")]
		if !ok {
			message = defaultShapeMessage
		}

		fields = append(fields, api.FieldError{Field: field, Rule: found.SchemaField, Message: message})
	}

	slices.SortStableFunc(fields, func(a, b api.FieldError) int { return strings.Compare(a.Field, b.Field) })

	return slices.CompactFunc(fields, func(a, b api.FieldError) bool { return a.Field == b.Field })
}

// flatten returns the schema errors inside err, which may be a MultiError of schema errors and nested MultiErrors.
func flatten(err error) []*openapi3.SchemaError {
	var multi openapi3.MultiError
	if errors.As(err, &multi) {
		var found []*openapi3.SchemaError
		for _, inner := range multi {
			found = append(found, flatten(inner)...)
		}

		return found
	}

	var schemaError *openapi3.SchemaError
	if errors.As(err, &schemaError) {
		return []*openapi3.SchemaError{schemaError}
	}

	return nil
}

// fieldPath writes a JSON pointer as the field path of a problem: items[0].amount for [items 0 amount].
func fieldPath(pointer []string) string {
	var path strings.Builder

	for _, part := range pointer {
		if _, err := strconv.Atoi(part); err == nil {
			path.WriteString("[" + part + "]")

			continue
		}

		if path.Len() > 0 {
			path.WriteString(".")
		}

		path.WriteString(part)
	}

	return path.String()
}

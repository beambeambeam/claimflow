package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// A body that is not JSON, or whose whole value has the wrong type, is a 400 that names no field.
func TestRequestValidationWithoutAField(t *testing.T) {
	for _, body := range []string{`{"title": `, `[]`, ``} {
		if got := badRequestErrors(t, body); len(got) != 0 {
			t.Fatalf("body %q: errors = %+v, want none", body, got)
		}
	}
}

// Every broken field is named at once, in field order, and a wrong type or a missing member names its own path.
func TestRequestValidationNamesEveryField(t *testing.T) {
	body := `{"purpose": 7, "items": [1, {"category": "TRAVEL", "description": "d", "expenseDate": "2026-11-03", "amount": 1.5}]}`

	want := []fieldError{
		{Field: "items[0]", Rule: "type", Message: "Enter the line item."},
		{Field: "items[1].amount", Rule: "type", Message: "Enter an amount with 2 decimals, e.g. 1250.00."},
		{Field: "items[1].receiptRef", Rule: "required", Message: "Enter the receipt reference (up to 50 characters)."},
		{Field: "purpose", Rule: "type", Message: "Enter a purpose (up to 1,000 characters)."},
		{Field: "title", Rule: "required", Message: "Enter a title (up to 120 characters)."},
	}

	if got := badRequestErrors(t, body); !slices.Equal(got, want) {
		t.Fatalf("errors = %+v, want %+v", got, want)
	}
}

// failingBody is a request body that cannot be read.
type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("read refused") }

func TestRequestValidationUnreadableBody(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/claims", failingBody{})
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "sid", Value: token, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	decodeProblem(t, recorder, http.StatusBadRequest, "BAD_REQUEST")

	if got := problemErrors(t, recorder); len(got) != 0 {
		t.Fatalf("errors = %+v, want none", got)
	}
}

// A field with no message of its own gets the default, and an error that is not a schema error names no field.
func TestShapeErrorsDefaults(t *testing.T) {
	if got := shapeErrors(errors.New("not a schema error"), nil); len(got) != 0 {
		t.Fatalf("shapeErrors = %+v, want none", got)
	}

	schema := openapi3.NewObjectSchema().WithProperty("x", openapi3.NewStringSchema())

	got := shapeErrors(schema.VisitJSON(map[string]any{"x": 1}, openapi3.MultiErrors()), nil)
	if len(got) != 1 || got[0].Field != "x" || got[0].Message != defaultShapeMessage {
		t.Fatalf("shapeErrors = %+v, want x with the default message", got)
	}

	if got := fieldPath([]string{"items", "0", "amount"}); got != "items[0].amount" {
		t.Fatalf("fieldPath = %q, want items[0].amount", got)
	}
}

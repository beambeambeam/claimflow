package claim

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/auth"
	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

// locationPrefix is the path that a claim's Location header continues with its id (SDS §4.1).
const locationPrefix = "/api/v1/claims/"

var errUnauthenticated = &httperr.Error{
	Status: http.StatusUnauthorized,
	Code:   api.ProblemCodeUnauthenticated,
	Title:  "Authentication required.",
}

// Handler implements the claim operations of api.StrictServerInterface. It maps requests to Service calls and
// Service errors to problem responses.
type Handler struct {
	service Service
}

// NewHandler returns a Handler over service.
func NewHandler(service Service) Handler {
	return Handler{service: service}
}

// CreateClaim submits the claim of the request body for the session user. The route permission claim:create has
// already been checked. ErrForbidden, which is BR-15 here, becomes 403. The identity comes from the session only.
func (h Handler) CreateClaim(ctx context.Context, request api.CreateClaimRequestObject) (api.CreateClaimResponseObject, error) {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return nil, errUnauthenticated
	}

	created, err := h.service.Create(ctx, Actor{ID: user.ID}, input(request.Body), true)
	if errors.Is(err, ErrForbidden) {
		return nil, httperr.ErrForbidden
	}

	var invalid ValidationError
	if errors.As(err, &invalid) {
		return nil, fmt.Errorf("claim rules: %w", httperr.ValidationFailed(fieldErrors(invalid)))
	}

	if err != nil {
		return nil, err
	}

	return api.CreateClaim201JSONResponse{
		Body: api.ClaimEnvelope{Data: claimBody(&created)},
		Headers: api.CreateClaim201ResponseHeaders{
			ETag:     fmt.Sprintf("%q", strconv.Itoa(created.Version)),
			Location: locationPrefix + created.ID.String(),
		},
	}, nil
}

// fieldErrors maps the broken rules of a ValidationError to the problem's errors[] entries.
func fieldErrors(invalid ValidationError) []api.FieldError {
	fields := make([]api.FieldError, 0, len(invalid.Fields))
	for _, field := range invalid.Fields {
		fields = append(fields, api.FieldError{Field: field.Field, Rule: field.Rule, Message: field.Message})
	}

	return fields
}

// input maps the request body to the module's input. The body has no total, and any total in the JSON is ignored.
func input(body *api.CreateClaimJSONRequestBody) Input {
	items := make([]LineItemInput, 0, len(body.Items))
	for _, item := range body.Items {
		items = append(items, LineItemInput{
			Category:    db.ExpenseCategory(item.Category),
			Description: item.Description,
			ExpenseDate: item.ExpenseDate.Time,
			Amount:      item.Amount,
			ReceiptRef:  item.ReceiptRef,
		})
	}

	return Input{Title: body.Title, Purpose: body.Purpose, Items: items}
}

// claimBody maps a claim to the contract's Claim. Times are sent in UTC.
func claimBody(claim *Claim) api.Claim {
	body := api.Claim{
		Id:               claim.ID.String(),
		ClaimNo:          claim.ClaimNo,
		Status:           api.ClaimStatus(claim.Status),
		Title:            claim.Title,
		Purpose:          claim.Purpose,
		Total:            claim.Total,
		ReturnCount:      claim.ReturnCount,
		Version:          claim.Version,
		Requester:        memberBody(claim.Requester),
		FirstSubmittedAt: utc(claim.FirstSubmittedAt),
		SubmittedAt:      utc(claim.SubmittedAt),
		DecidedAt:        utc(claim.DecidedAt),
		CreatedAt:        claim.CreatedAt.UTC(),
		UpdatedAt:        claim.UpdatedAt.UTC(),
		Items:            make([]api.LineItem, 0, len(claim.Items)),
		History:          make([]api.HistoryEntry, 0, len(claim.History)),
		AllowedActions:   claim.AllowedActions,
	}

	if claim.AssignedReviewer != nil {
		reviewer := memberBody(*claim.AssignedReviewer)
		body.AssignedReviewer = &reviewer
	}

	for _, item := range claim.Items {
		body.Items = append(body.Items, api.LineItem{
			LineNo:      item.LineNo,
			Category:    api.ExpenseCategory(item.Category),
			Description: item.Description,
			ExpenseDate: openapi_types.Date{Time: item.ExpenseDate},
			Amount:      item.Amount,
			ReceiptRef:  item.ReceiptRef,
		})
	}

	for i := range claim.History {
		entry := &claim.History[i]

		historyEntry := api.HistoryEntry{
			Id:        entry.ID,
			Action:    api.ClaimAction(entry.Action),
			ToStatus:  api.ClaimStatus(entry.ToStatus),
			Actor:     memberBody(entry.Actor),
			Comment:   entry.Comment,
			CreatedAt: entry.CreatedAt.UTC(),
		}

		if entry.FromStatus != nil {
			from := api.ClaimStatus(*entry.FromStatus)
			historyEntry.FromStatus = &from
		}

		body.History = append(body.History, historyEntry)
	}

	return body
}

func memberBody(member Member) api.MemberSummary {
	return api.MemberSummary{Id: member.ID.String(), Name: member.Name, Position: member.Position}
}

// utc returns t in UTC, or nil for a missing time.
func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	value := t.UTC()

	return &value
}

package claim

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/auth"
	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

// locationPrefix is the path that a claim's Location header continues with its id (SDS §4.1).
const locationPrefix = "/api/v1/claims/"

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

var errUnauthenticated = &httperr.Error{
	Status: http.StatusUnauthorized,
	Code:   api.ProblemCodeUnauthenticated,
	Title:  "Authentication required.",
}

var errClaimNotFound = &httperr.Error{
	Status: http.StatusNotFound,
	Code:   api.ProblemCodeNotFound,
	Title:  "Claim not found.",
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

// ListClaims returns one page of the session user's claims. A missing or unknown scope, a page below 1, and a page
// size outside 1–100 are 400. The route permission depends on the scope, so the handler checks claim:list for
// scope=mine after it reads the scope: a Finance Officer gets 403 (F03-AC-04).
func (h Handler) ListClaims(ctx context.Context, request api.ListClaimsRequestObject) (api.ListClaimsResponseObject, error) {
	params := request.Params
	page, pageSize := defaultPage, defaultPageSize

	if params.Page != nil {
		page = *params.Page
	}

	if params.PageSize != nil {
		pageSize = *params.PageSize
	}

	if params.Scope != api.ListScopeMine || page < 1 || pageSize < 1 || pageSize > maxPageSize {
		return nil, httperr.ErrValidation
	}

	if err := auth.Authorize(ctx, "claim", "list"); err != nil {
		return nil, fmt.Errorf("authorize: %w", err)
	}

	user, ok := auth.UserFrom(ctx)
	if !ok { // coverage-ignore
		return nil, errUnauthenticated
	}

	result, err := h.service.ListMine(ctx, Actor{ID: user.ID}, page, pageSize)
	if err != nil { // coverage-ignore
		return nil, err
	}

	body := api.ClaimPage{Data: make([]api.ClaimSummary, 0, len(result.Claims)), Page: page, PageSize: pageSize, Total: result.Total}
	for i := range result.Claims {
		summary := &result.Claims[i]
		body.Data = append(body.Data, api.ClaimSummary{
			Id:          summary.ID.String(),
			ClaimNo:     summary.ClaimNo,
			Status:      api.ClaimStatus(summary.Status),
			Title:       summary.Title,
			Total:       summary.Total,
			Requester:   memberBody(summary.Requester),
			SubmittedAt: utc(summary.SubmittedAt),
			CreatedAt:   summary.CreatedAt.UTC(),
		})
	}

	return api.ListClaims200JSONResponse(body), nil
}

// GetClaim returns a claim that the session user may see, with the actions the claim policy allows. A malformed id is
// 400, a missing claim 404, and a claim the user may not see 403 with no claim data (F03-AC-06, F03-AC-08).
func (h Handler) GetClaim(ctx context.Context, request api.GetClaimRequestObject) (api.GetClaimResponseObject, error) {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return nil, errUnauthenticated
	}

	id, err := uuid.Parse(request.Id)
	if err != nil {
		return nil, httperr.ErrValidation
	}

	found, err := h.service.Get(ctx, Actor{ID: user.ID}, id)
	if errors.Is(err, ErrNotFound) {
		return nil, errClaimNotFound
	}

	if errors.Is(err, ErrForbidden) {
		return nil, httperr.ErrForbidden
	}

	if err != nil { // coverage-ignore
		return nil, err
	}

	return api.GetClaim200JSONResponse{
		Body:    api.ClaimEnvelope{Data: claimBody(&found)},
		Headers: api.GetClaim200ResponseHeaders{ETag: fmt.Sprintf("%q", strconv.Itoa(found.Version))},
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

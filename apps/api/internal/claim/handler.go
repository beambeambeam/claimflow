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

// ListClaims returns one page of claims in the scope: the session user's own (mine), or the SUBMITTED claims assigned
// to them (review). A missing or unknown scope, a page below 1, and a page size outside 1–100 are 400. The route
// permission depends on the scope, so the handler checks it after it reads the scope: claim:list for mine, which a
// Finance Officer lacks (F03-AC-04), and review:list for review, which a requester lacks (F04-AC-02).
func (h Handler) ListClaims(ctx context.Context, request api.ListClaimsRequestObject) (api.ListClaimsResponseObject, error) {
	params := request.Params
	page, pageSize := defaultPage, defaultPageSize

	if params.Page != nil {
		page = *params.Page
	}

	if params.PageSize != nil {
		pageSize = *params.PageSize
	}

	if page < 1 || pageSize < 1 || pageSize > maxPageSize {
		return nil, httperr.ErrValidation
	}

	resource, list := "claim", h.service.ListMine

	switch params.Scope {
	case api.ListScopeMine:
	case api.ListScopeReview:
		resource, list = "review", h.service.ListReview
	default:
		return nil, httperr.ErrValidation
	}

	if err := auth.Authorize(ctx, resource, "list"); err != nil {
		return nil, fmt.Errorf("authorize: %w", err)
	}

	user, ok := auth.UserFrom(ctx)
	if !ok { // coverage-ignore
		return nil, errUnauthenticated
	}

	result, err := list(ctx, Actor{ID: user.ID}, page, pageSize)
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

// ApproveClaim approves the claim for the session user. The checks run in the order of F-04 §1.4.
func (h Handler) ApproveClaim(ctx context.Context, request api.ApproveClaimRequestObject) (api.ApproveClaimResponseObject, error) {
	decided, err := h.decide(ctx, request.Id, request.Params.IfMatch, ActionApprove, request.Body)
	if err != nil {
		return nil, err
	}

	return api.ApproveClaim200JSONResponse{DecidedJSONResponse: decidedBody(&decided)}, nil
}

// RejectClaim rejects the claim for the session user. The checks run in the order of F-04 §1.4.
func (h Handler) RejectClaim(ctx context.Context, request api.RejectClaimRequestObject) (api.RejectClaimResponseObject, error) {
	decided, err := h.decide(ctx, request.Id, request.Params.IfMatch, ActionReject, request.Body)
	if err != nil {
		return nil, err
	}

	return api.RejectClaim200JSONResponse{DecidedJSONResponse: decidedBody(&decided)}, nil
}

// decide reads If-Match, the claim id, and the comment of a decision request, takes the decision, and maps the
// module's errors to problems. A missing If-Match is 428 and a malformed one 400, both before the claim is read
// (F-04 §1.4, step 1).
func (h Handler) decide(ctx context.Context, rawID string, ifMatch *string, action string, body *api.ClaimDecision) (Claim, error) {
	version, err := parseVersion(ifMatch)
	if err != nil {
		return Claim{}, err
	}

	user, ok := auth.UserFrom(ctx)
	if !ok {
		return Claim{}, errUnauthenticated
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		return Claim{}, httperr.ErrValidation
	}

	var comment string
	if body != nil && body.Comment != nil {
		comment = *body.Comment
	}

	decided, err := h.service.Act(ctx, Actor{ID: user.ID}, id, version, action, comment)

	switch {
	case errors.Is(err, ErrNotFound):
		return Claim{}, errClaimNotFound
	case errors.Is(err, ErrForbidden):
		return Claim{}, httperr.ErrForbidden
	case errors.Is(err, ErrConflict):
		return Claim{}, httperr.ErrConflict
	case err != nil:
		return Claim{}, err
	}

	return decided, nil
}

// parseVersion reads the claim version from an If-Match value such as "3". A missing header is 428. A value that is not
// a quoted version number is 400: no quotes, a weak validator, a sign or a fraction, or more than 31 bits.
func parseVersion(ifMatch *string) (int, error) {
	if ifMatch == nil {
		return 0, httperr.ErrPreconditionRequired
	}

	quoted := *ifMatch
	if len(quoted) < 3 || quoted[0] != '"' || quoted[len(quoted)-1] != '"' {
		return 0, httperr.ErrValidation
	}

	version, err := strconv.ParseUint(quoted[1:len(quoted)-1], 10, 31)
	if err != nil {
		return 0, httperr.ErrValidation
	}

	return int(version), nil
}

// decidedBody is the 200 response of a decision: the claim and its new ETag.
func decidedBody(decided *Claim) api.DecidedJSONResponse {
	return api.DecidedJSONResponse{
		Body:    api.ClaimEnvelope{Data: claimBody(decided)},
		Headers: api.DecidedResponseHeaders{ETag: fmt.Sprintf("%q", strconv.Itoa(decided.Version))},
	}
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

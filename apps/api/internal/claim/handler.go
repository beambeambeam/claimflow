package claim

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/auth"
	"github.com/beambeambeam/claimflow/apps/api/internal/daterange"
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

// CreateClaim submits or saves the claim of the request body for the session user. The route permission claim:create has
// already been checked. ErrForbidden, which is BR-15 here, becomes 403. The identity comes from the session only.
func (h Handler) CreateClaim(ctx context.Context, request api.CreateClaimRequestObject) (api.CreateClaimResponseObject, error) {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return nil, httperr.ErrUnauthenticated
	}

	created, err := h.service.Create(ctx, Actor{ID: user.ID, Role: user.Role}, input(request.Body), request.Params.Draft == nil || !*request.Params.Draft)
	if err != nil {
		return nil, problem(err)
	}

	return api.CreateClaim201JSONResponse{
		Body: api.ClaimEnvelope{Data: claimBody(&created)},
		Headers: api.CreateClaim201ResponseHeaders{
			ETag:     etag(created.Version),
			Location: locationPrefix + created.ID.String(),
		},
	}, nil
}

// listScopes maps each scope of the contract to the module's scope and the resource of its route permission:
// claim:list for mine, which a Finance Officer lacks (F03-AC-04), review:list for review, which a requester lacks
// (F04-AC-02), and finance:list for finance, which only a Finance Officer has (F-08 §1.8). all has no resource: it needs
// only a session (F-09 §1.7).
var listScopes = map[api.ListClaimsParamsScope]struct {
	scope    Scope
	resource string
}{
	api.ListScopeMine:    {scope: ScopeMine, resource: "claim"},
	api.ListScopeReview:  {scope: ScopeReview, resource: "review"},
	api.ListScopeFinance: {scope: ScopeFinance, resource: "finance"},
	api.ListScopeAll:     {scope: ScopeAll},
}

// ListClaims returns one page of claims in the scope: the session user's own (mine), the SUBMITTED claims assigned to
// them (review), every PENDING_FINANCE claim (finance), or every claim they can see, narrowed by the filters (all). A
// missing or unknown scope, a page below 1, and a page size outside 1–100 are 400. A bad filter is a 400 that names
// each bad parameter (F-09 §1.3). The route permission depends on the scope, so the handler checks it after it reads
// the scope.
func (h Handler) ListClaims(ctx context.Context, request api.ListClaimsRequestObject) (api.ListClaimsResponseObject, error) {
	params := request.Params
	q := Query{Page: defaultPage, PageSize: defaultPageSize}

	if params.Page != nil {
		q.Page = *params.Page
	}

	if params.PageSize != nil {
		q.PageSize = *params.PageSize
	}

	scope, ok := listScopes[params.Scope]
	if !ok || q.Page < 1 || q.PageSize < 1 || q.PageSize > maxPageSize {
		return nil, httperr.ErrValidation
	}

	filters, invalid := listFilters(&params)
	if len(invalid) > 0 {
		return nil, fmt.Errorf("list filters: %w", httperr.InvalidFields(invalid))
	}

	q.Scope, q.Filters = scope.scope, filters

	user, ok := auth.UserFrom(ctx)
	if !ok {
		return nil, httperr.ErrUnauthenticated
	}

	if scope.resource != "" {
		if _, err := auth.Authorize(ctx, scope.resource, "list"); err != nil {
			return nil, fmt.Errorf("authorize: %w", err)
		}
	}

	result, err := h.service.List(ctx, Actor{ID: user.ID, Role: user.Role}, &q)
	if err != nil { // coverage-ignore
		return nil, err
	}

	body := api.ClaimPage{Data: make([]api.ClaimSummary, 0, len(result.Claims)), Page: q.Page, PageSize: q.PageSize, Total: result.Total}
	for i := range result.Claims {
		body.Data = append(body.Data, summaryBody(&result.Claims[i]))
	}

	return api.ListClaims200JSONResponse(body), nil
}

// The messages of the filter checks (F-09 §1.3).
const (
	scopeOnlyMessage = "Filters are only available for scope=all."
	maxQueryLength   = 120
)

// listFilters checks the filter parameters of a list request and returns them as Filters, or one field error per bad
// parameter, in parameter order (F-09 §1.3). A filter with a scope other than all is an error, so a client never gets
// an unfiltered list it thinks is filtered. A from later than to is an error on to.
func listFilters(params *api.ListClaimsParams) (Filters, []api.FieldError) {
	invalid := []api.FieldError{}

	if params.Scope != api.ListScopeAll {
		for _, set := range []struct {
			field string
			isSet bool
		}{
			{"status", params.Status != nil},
			{"category", params.Category != nil},
			{"from", params.From != nil},
			{"to", params.To != nil},
			{"q", params.Q != nil},
		} {
			if set.isSet {
				invalid = append(invalid, api.FieldError{Field: set.field, Rule: "scope", Message: scopeOnlyMessage})
			}
		}

		return Filters{}, invalid
	}

	var filters Filters

	if params.Status != nil {
		if params.Status.Valid() {
			status := db.ClaimStatus(*params.Status)
			filters.Status = &status
		} else {
			invalid = append(invalid, api.FieldError{Field: "status", Rule: "enum", Message: "Unknown status."})
		}
	}

	if params.Category != nil {
		if params.Category.Valid() {
			category := db.ExpenseCategory(*params.Category)
			filters.Category = &category
		} else {
			invalid = append(invalid, api.FieldError{Field: "category", Rule: "enum", Message: "Unknown category."})
		}
	}

	var dateProblems []api.FieldError
	filters.From, filters.To, dateProblems = daterange.Parse(params.From, params.To)
	invalid = append(invalid, dateProblems...)

	if params.Q != nil {
		filters.Q = strings.TrimSpace(*params.Q)
		if utf8.RuneCountInString(filters.Q) > maxQueryLength {
			invalid = append(invalid, api.FieldError{Field: "q", Rule: "maxLength", Message: "Search text must be at most 120 characters."})
		}
	}

	if len(invalid) > 0 {
		return Filters{}, invalid
	}

	return filters, nil
}

// GetClaim returns a claim that the session user may see, with the actions the claim policy allows. A malformed id is
// 400, a missing claim 404, and a claim the user may not see 403 with no claim data (F03-AC-06, F03-AC-08).
func (h Handler) GetClaim(ctx context.Context, request api.GetClaimRequestObject) (api.GetClaimResponseObject, error) {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return nil, httperr.ErrUnauthenticated
	}

	id, err := uuid.Parse(request.Id)
	if err != nil {
		return nil, httperr.ErrValidation
	}

	found, err := h.service.Get(ctx, Actor{ID: user.ID, Role: user.Role}, id)
	if err != nil {
		return nil, problem(err)
	}

	return api.GetClaim200JSONResponse{
		Body:    api.ClaimEnvelope{Data: claimBody(&found)},
		Headers: api.GetClaim200ResponseHeaders{ETag: etag(found.Version)},
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

// FinalApproveClaim makes the final approval of a PENDING_FINANCE claim for the session user. The checks run in the
// order of F-08 §1.4.
func (h Handler) FinalApproveClaim(ctx context.Context, request api.FinalApproveClaimRequestObject) (api.FinalApproveClaimResponseObject, error) {
	decided, err := h.decide(ctx, request.Id, request.Params.IfMatch, ActionFinalApprove, request.Body)
	if err != nil {
		return nil, err
	}

	return api.FinalApproveClaim200JSONResponse{DecidedJSONResponse: decidedBody(&decided)}, nil
}

// FinalRejectClaim makes the final rejection of a PENDING_FINANCE claim for the session user. The checks run in the
// order of F-08 §1.4.
func (h Handler) FinalRejectClaim(ctx context.Context, request api.FinalRejectClaimRequestObject) (api.FinalRejectClaimResponseObject, error) {
	decided, err := h.decide(ctx, request.Id, request.Params.IfMatch, ActionFinalReject, request.Body)
	if err != nil {
		return nil, err
	}

	return api.FinalRejectClaim200JSONResponse{DecidedJSONResponse: decidedBody(&decided)}, nil
}

// ReturnClaim returns the claim to its requester for the session user. The checks run in the order of F-04 §1.4.
func (h Handler) ReturnClaim(ctx context.Context, request api.ReturnClaimRequestObject) (api.ReturnClaimResponseObject, error) {
	returned, err := h.decide(ctx, request.Id, request.Params.IfMatch, ActionReturn, request.Body)
	if err != nil {
		return nil, err
	}

	return api.ReturnClaim200JSONResponse{DecidedJSONResponse: decidedBody(&returned)}, nil
}

// WithdrawClaim withdraws the claim of the session user, with an optional comment. The checks run in the order of F-05 §1.4.
func (h Handler) WithdrawClaim(ctx context.Context, request api.WithdrawClaimRequestObject) (api.WithdrawClaimResponseObject, error) {
	var comment *string
	if request.Body != nil {
		comment = request.Body.Comment
	}

	withdrawn, err := h.decide(ctx, request.Id, request.Params.IfMatch, ActionWithdraw, &api.ClaimDecision{Comment: comment})
	if err != nil {
		return nil, err
	}

	return api.WithdrawClaim200JSONResponse{DecidedJSONResponse: decidedBody(&withdrawn)}, nil
}

// decide reads If-Match, the claim id, and the comment of a decision request, takes the decision, and maps the
// module's errors to problems. A missing If-Match is 428 and a malformed one 400, both before the claim is read
// (F-04 §1.4, step 1).
func (h Handler) decide(ctx context.Context, rawID string, ifMatch *string, action Action, body *api.ClaimDecision) (Claim, error) {
	actor, id, version, err := mutationTarget(ctx, rawID, ifMatch)
	if err != nil {
		return Claim{}, err
	}

	var comment string
	if body != nil && body.Comment != nil {
		comment = *body.Comment
	}

	decided, err := h.service.Act(ctx, actor, id, version, action, comment)
	if err != nil {
		return Claim{}, actionProblem(err, action)
	}

	return decided, nil
}

// UpdateClaim replaces a draft or returned claim's content.
func (h Handler) UpdateClaim(ctx context.Context, request api.UpdateClaimRequestObject) (api.UpdateClaimResponseObject, error) {
	actor, id, version, err := mutationTarget(ctx, request.Id, request.Params.IfMatch)
	if err != nil {
		return nil, err
	}

	updated, err := h.service.Edit(ctx, actor, id, version, input(request.Body))
	if err != nil {
		return nil, actionProblem(err, ActionEdit)
	}

	return api.UpdateClaim200JSONResponse{ClaimChangedJSONResponse: changedBody(&updated)}, nil
}

// DeleteClaim removes a draft and its line items.
func (h Handler) DeleteClaim(ctx context.Context, request api.DeleteClaimRequestObject) (api.DeleteClaimResponseObject, error) {
	actor, id, version, err := mutationTarget(ctx, request.Id, request.Params.IfMatch)
	if err != nil {
		return nil, err
	}

	if err := h.service.Delete(ctx, actor, id, version); err != nil {
		return nil, actionProblem(err, ActionDelete)
	}

	return api.DeleteClaim204Response{}, nil
}

// SubmitClaim submits the claim for review.
func (h Handler) SubmitClaim(ctx context.Context, request api.SubmitClaimRequestObject) (api.SubmitClaimResponseObject, error) {
	submitted, err := h.decide(ctx, request.Id, request.Params.IfMatch, ActionSubmit, nil)
	if err != nil {
		return nil, err
	}

	return api.SubmitClaim200JSONResponse{ClaimChangedJSONResponse: changedBody(&submitted)}, nil
}

// mutationTarget reads If-Match before looking up the claim.
func mutationTarget(ctx context.Context, rawID string, ifMatch *string) (Actor, uuid.UUID, int, error) {
	version, invalid := parseVersion(ifMatch)
	if invalid != nil {
		return Actor{}, uuid.Nil, 0, invalid
	}

	user, ok := auth.UserFrom(ctx)
	if !ok {
		return Actor{}, uuid.Nil, 0, httperr.ErrUnauthenticated
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		return Actor{}, uuid.Nil, 0, httperr.ErrValidation
	}

	return Actor{ID: user.ID, Role: user.Role}, id, version, nil
}

var transitionErrors = map[Action]FieldError{
	ActionApprove:      transitionError,
	ActionReject:       transitionError,
	ActionReturn:       {Field: "status", Rule: ruleTransition, Message: "Only a submitted claim can be returned."},
	ActionEdit:         {Field: "status", Rule: "BR-10", Message: "Only a draft or a returned claim can be edited."},
	ActionDelete:       {Field: "status", Rule: "BR-10", Message: "Only a draft can be deleted."},
	ActionSubmit:       {Field: "status", Rule: ruleTransition, Message: "Only a draft or a returned claim can be submitted."},
	ActionWithdraw:     {Field: "status", Rule: "BR-16", Message: "Only a submitted or returned claim can be withdrawn."},
	ActionFinalApprove: finalTransitionError,
	ActionFinalReject:  finalTransitionError,
}

// finalTransitionError is the field error of a final decision on a claim that is not PENDING_FINANCE (F-08 §1.5).
var finalTransitionError = FieldError{Field: "status", Rule: ruleTransition, Message: "Only a claim pending finance can be finally approved or rejected."}

// returnLimitError is the field error of ErrReturnLimit: the claim was already returned MaxReturns times (BR-14).
var returnLimitError = FieldError{Field: "returnCount", Rule: "BR-14", Message: fmt.Sprintf("This claim has already been returned %d times.", MaxReturns)}

func actionProblem(err error, action Action) error {
	if errors.Is(err, ErrReturnLimit) {
		return fmt.Errorf("claim action: %w", httperr.InvalidTransition(apiFieldError(returnLimitError)))
	}

	if errors.Is(err, ErrInvalidTransition) {
		return fmt.Errorf("claim action: %w", httperr.InvalidTransition(apiFieldError(transitionErrors[action])))
	}

	return problem(err)
}

func changedBody(claim *Claim) api.ClaimChangedJSONResponse {
	return api.ClaimChangedJSONResponse{
		Body:    api.ClaimEnvelope{Data: claimBody(claim)},
		Headers: api.ClaimChangedResponseHeaders{ETag: etag(claim.Version)},
	}
}

// moduleProblems maps each error of the module to its problem response (SDS §4.2).
var moduleProblems = []struct {
	err     error
	problem error
}{
	{ErrNotFound, errClaimNotFound},
	{ErrForbidden, httperr.ErrForbidden},
	{ErrConflict, httperr.ErrConflict},
}

// problem maps a module error to its problem response. ErrForbidden is also BR-15 on Create. A ValidationError
// becomes 422 with its fields. Any other error is left as it is, which the server answers with 500.
func problem(err error) error {
	var invalid ValidationError
	if errors.As(err, &invalid) {
		return fmt.Errorf("claim rules: %w", httperr.ValidationFailed(fieldErrors(invalid)))
	}

	for _, known := range moduleProblems {
		if errors.Is(err, known.err) {
			return known.problem
		}
	}

	return err
}

// etag is the ETag of a claim version, the version number in quotes, which If-Match sends back.
func etag(version int) string {
	return strconv.Quote(strconv.Itoa(version))
}

// ValidateIfMatch checks the version header ahead of request body validation.
func ValidateIfMatch(values []string) *httperr.Error {
	switch len(values) {
	case 0:
		return httperr.ErrPreconditionRequired
	case 1:
		_, invalid := parseVersion(&values[0])
		return invalid
	default:
		return httperr.ErrValidation
	}
}

// parseVersion reads the claim version from an If-Match value such as "3". A missing header is 428. A value that is not
// a quoted version number is 400: no quotes, a weak validator, a sign or a fraction, or more than 31 bits.
func parseVersion(ifMatch *string) (int, *httperr.Error) {
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
		Headers: api.DecidedResponseHeaders{ETag: etag(decided.Version)},
	}
}

// fieldErrors maps the broken rules of a ValidationError to the problem's errors[] entries.
func fieldErrors(invalid ValidationError) []api.FieldError {
	fields := make([]api.FieldError, 0, len(invalid.Fields))
	for _, field := range invalid.Fields {
		fields = append(fields, apiFieldError(field))
	}

	return fields
}

// apiFieldError maps one field error of the module to the contract's.
func apiFieldError(field FieldError) api.FieldError {
	return api.FieldError{Field: field.Field, Rule: field.Rule, Message: field.Message}
}

// input maps the request body to the module's input. The body has no total, and any total in the JSON is ignored.
func input(body *api.ClaimInput) Input {
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
		AllowedActions:   make([]string, 0, len(claim.AllowedActions)),
	}

	if claim.AssignedReviewer != nil {
		reviewer := memberBody(*claim.AssignedReviewer)
		body.AssignedReviewer = &reviewer
	}

	for i := range claim.Items {
		body.Items = append(body.Items, lineItemBody(&claim.Items[i]))
	}

	for i := range claim.History {
		body.History = append(body.History, historyBody(&claim.History[i]))
	}

	for _, action := range claim.AllowedActions {
		body.AllowedActions = append(body.AllowedActions, string(action))
	}

	return body
}

// summaryBody maps a list entry to the contract's ClaimSummary.
func summaryBody(summary *Summary) api.ClaimSummary {
	return api.ClaimSummary{
		Id:          summary.ID.String(),
		ClaimNo:     summary.ClaimNo,
		Status:      api.ClaimStatus(summary.Status),
		Title:       summary.Title,
		Total:       summary.Total,
		Requester:   memberBody(summary.Requester),
		SubmittedAt: utc(summary.SubmittedAt),
		CreatedAt:   summary.CreatedAt.UTC(),
	}
}

// lineItemBody maps a line item to the contract's LineItem.
func lineItemBody(item *LineItem) api.LineItem {
	return api.LineItem{
		LineNo:      item.LineNo,
		Category:    api.ExpenseCategory(item.Category),
		Description: item.Description,
		ExpenseDate: openapi_types.Date{Time: item.ExpenseDate},
		Amount:      item.Amount,
		ReceiptRef:  item.ReceiptRef,
	}
}

// historyBody maps a history entry to the contract's HistoryEntry.
func historyBody(entry *HistoryEntry) api.HistoryEntry {
	body := api.HistoryEntry{
		Id:        entry.ID,
		Action:    api.ClaimAction(entry.Action),
		ToStatus:  api.ClaimStatus(entry.ToStatus),
		Actor:     memberBody(entry.Actor),
		Comment:   entry.Comment,
		CreatedAt: entry.CreatedAt.UTC(),
	}

	if entry.FromStatus != nil {
		from := api.ClaimStatus(*entry.FromStatus)
		body.FromStatus = &from
	}

	return body
}

// memberBody maps a member to the contract's MemberSummary. The module has no contract types, so this copy is the seam.
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

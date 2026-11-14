package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// decide sends POST /claims/{id}/{action} with the session cookie. An empty ifMatch sends no If-Match header, and an
// empty body sends no body.
func decide(t *testing.T, router http.Handler, token, id, action, ifMatch, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/claims/"+id+"/"+action, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}

	request.AddCookie(&http.Cookie{Name: "sid", Value: token, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

// decodeClaim decodes a 200 claim envelope.
func decodeClaim(t *testing.T, recorder *httptest.ResponseRecorder) claimResponse {
	t.Helper()

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
	}

	var body claimResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode claim: %v", err)
	}

	return body
}

// openClaim opens the claim as token.
func openClaim(t *testing.T, router http.Handler, token, id string) claimResponse {
	t.Helper()

	return decodeClaim(t, getWithSession(t, router, "/api/v1/claims/"+id, token))
}

// storedState is the part of a claim that a refused decision must leave as it was.
type storedState struct {
	Status    string
	Version   int
	Decided   bool
	History   int
	UpdatedAt time.Time
}

func readState(t *testing.T, pool *pgxpool.Pool, id string) storedState {
	t.Helper()

	var state storedState
	if err := pool.QueryRow(t.Context(), `
		SELECT c.status::text, c.version, c.decided_at IS NOT NULL, c.updated_at,
		       (SELECT count(*)::int FROM claim_history h WHERE h.claim_id = c.id)
		FROM claims c WHERE c.id = $1`, uuid.MustParse(id)).
		Scan(&state.Status, &state.Version, &state.Decided, &state.UpdatedAt, &state.History); err != nil {
		t.Fatalf("read claim %s: %v", id, err)
	}

	return state
}

// submittedBy has username submit threeItemsBody and returns the claim id.
func submittedBy(t *testing.T, router http.Handler, username string) string {
	t.Helper()

	recorder := postClaim(t, router, loginAs(t, router, username), threeItemsBody())
	if recorder.Code != http.StatusCreated {
		t.Fatalf("%s submit status = %d, want 201; body = %s", username, recorder.Code, recorder.Body)
	}

	var body claimResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	return body.Data.ID
}

// F04-TEST-01 (F04-AC-01): rev.malee's queue holds only her SUBMITTED claims, earliest submittedAt first. The ids run
// the other way round, so the order cannot come from the id.
func TestF04TEST01ReviewQueueHoldsAssignedSubmittedClaimsOldestFirst(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "rev.malee")

	t1 := time.Date(2026, time.November, 1, 3, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)

	later := storedClaim(t, pool, "req.anan", "SUBMITTED", t2)
	earlier := storedClaim(t, pool, "req.ploy", "SUBMITTED", t1)
	storedClaim(t, pool, "req.anan", "APPROVED", t1)
	storedClaim(t, pool, "req.anan", "REJECTED", t1)
	storedClaim(t, pool, "req.niran", "SUBMITTED", t1) // assigned to rev.somchai
	storedClaim(t, pool, "rev.malee", "SUBMITTED", t1) // her own, assigned to rev.prasert

	page := listClaims(t, router, token, "?scope=review")

	if page.Total != 2 || page.Page != 1 || page.PageSize != 20 || len(page.Data) != 2 {
		t.Fatalf("page = %+v, want total 2, page 1, pageSize 20, 2 claims", page)
	}

	want := []struct{ id, submittedAt, requester string }{
		{earlier.String(), "2026-11-01T03:00:00Z", "Ploy"},
		{later.String(), "2026-11-02T03:00:00Z", "Anan K."},
	}
	for i, claim := range page.Data {
		if claim.ID != want[i].id || claim.Status != "SUBMITTED" || claim.SubmittedAt != want[i].submittedAt ||
			claim.Requester.Name != want[i].requester {
			t.Fatalf("data[%d] = %+v, want claim %s submitted %s by %s", i, claim, want[i].id, want[i].submittedAt, want[i].requester)
		}
	}
}

// Claims submitted at the same instant keep a stable order: id ascending.
func TestReviewQueueBreaksSubmittedAtTiesByIDAscending(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)

	at := time.Date(2026, time.November, 1, 3, 0, 0, 0, time.UTC)
	first := storedClaim(t, pool, "req.anan", "SUBMITTED", at)
	second := storedClaim(t, pool, "req.ploy", "SUBMITTED", at)

	page := listClaims(t, router, loginAs(t, router, "rev.malee"), "?scope=review")
	if len(page.Data) != 2 || page.Data[0].ID != first.String() || page.Data[1].ID != second.String() {
		t.Fatalf("data = %+v, want %s then %s", page.Data, first, second)
	}
}

// The review scope pages like the mine scope: defaults, a page past the last one, and the same bounds.
func TestReviewQueuePaginationAndBounds(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "rev.malee")

	at := time.Date(2026, time.November, 1, 3, 0, 0, 0, time.UTC)
	for i := range 25 {
		storedClaim(t, pool, "req.anan", "SUBMITTED", at.Add(time.Duration(i)*time.Minute))
	}

	pages := []struct {
		query                string
		page, pageSize, size int
	}{
		{"?scope=review", 1, 20, 20},
		{"?scope=review&page=2", 2, 20, 5},
		{"?scope=review&page=9", 9, 20, 0},
	}
	for _, tc := range pages {
		page := listClaims(t, router, token, tc.query)
		if page.Page != tc.page || page.PageSize != tc.pageSize || len(page.Data) != tc.size || page.Total != 25 {
			t.Fatalf("GET /claims%s = page %d, pageSize %d, %d claims, total %d; want page %d, pageSize %d, %d claims, total 25",
				tc.query, page.Page, page.PageSize, len(page.Data), page.Total, tc.page, tc.pageSize, tc.size)
		}
	}

	if body := getWithSession(t, router, "/api/v1/claims?scope=review&page=9", token).Body.String(); !strings.Contains(body, `"data":[]`) {
		t.Fatalf("body = %s, want data: []", body)
	}

	for _, query := range []string{"?scope=review&pageSize=101", "?scope=review&page=0"} {
		decodeProblem(t, getWithSession(t, router, "/api/v1/claims"+query, token), http.StatusBadRequest, "BAD_REQUEST")
	}
}

// F04-TEST-02 (F04-AC-02, NFR-02): a requester has no review:list, so the review scope is 403. fin.wanida's queue holds
// only the claims assigned to her: rev.prasert's, not rev.malee's.
func TestF04TEST02ReviewScopeIsForReviewersAndFinance(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)

	at := time.Date(2026, time.November, 1, 3, 0, 0, 0, time.UTC)
	prasert := storedClaim(t, pool, "rev.prasert", "SUBMITTED", at)
	storedClaim(t, pool, "rev.malee", "SUBMITTED", at)
	storedClaim(t, pool, "req.anan", "SUBMITTED", at)

	decodeProblem(t, getWithSession(t, router, "/api/v1/claims?scope=review", loginAs(t, router, "req.anan")), http.StatusForbidden, "FORBIDDEN")

	page := listClaims(t, router, loginAs(t, router, "fin.wanida"), "?scope=review")
	if page.Total != 1 || len(page.Data) != 1 || page.Data[0].ID != prasert.String() || page.Data[0].Requester.Name != "Prasert" {
		t.Fatalf("page = %+v, want only claim %s of Prasert", page, prasert)
	}
}

// F04-TEST-03 (F04-AC-03): the assigned reviewer opens a SUBMITTED claim and gets the full history and both actions.
func TestF04TEST03AssignedReviewerGetsApproveAndReject(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")

	claim := openClaim(t, router, loginAs(t, router, "rev.malee"), id).Data

	if claim.Status != "SUBMITTED" || len(claim.History) != 1 || claim.History[0].Action != "SUBMIT" ||
		!slices.Equal(claim.AllowedActions, []string{"approve", "reject"}) {
		t.Fatalf("claim = %+v, want SUBMITTED, one SUBMIT entry, actions [approve reject]", claim)
	}

	// The requester sees the same claim and has no action on it.
	if own := openClaim(t, router, loginAs(t, router, "req.anan"), id).Data; len(own.AllowedActions) != 0 {
		t.Fatalf("requester allowedActions = %v, want none", own.AllowedActions)
	}
}

// F04-TEST-04 (F04-AC-04, FR-06, FR-07, NFR-06): approving moves the claim to APPROVED, stamps the decision time, adds
// one to the version, sends the new ETag, and appends one APPROVE history entry with the comment. The claim leaves the
// reviewer's queue.
func TestF04TEST04ApproveSetsStatusVersionHistoryAndLeavesTheQueue(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	token := loginAs(t, router, "rev.malee")

	recorder := decide(t, router, token, id, "approve", `"0"`, `{"comment": "OK"}`)
	claim := decodeClaim(t, recorder).Data

	if got := recorder.Header().Get("ETag"); got != `"1"` {
		t.Fatalf("ETag = %q, want \"1\"", got)
	}

	if claim.Status != "APPROVED" || claim.Version != 1 || claim.DecidedAt != "2026-11-05T03:00:00Z" {
		t.Fatalf("claim = %+v, want APPROVED, version 1, decided 2026-11-05T03:00:00Z", claim)
	}

	if len(claim.History) != 2 {
		t.Fatalf("history = %+v, want 2 entries", claim.History)
	}

	last := claim.History[1]
	if last.Action != "APPROVE" || last.FromStatus == nil || *last.FromStatus != "SUBMITTED" || last.ToStatus != "APPROVED" ||
		last.Actor.Name != "Malee S." || last.Comment == nil || *last.Comment != "OK" {
		t.Fatalf("last history = %+v, want APPROVE SUBMITTED -> APPROVED by Malee S. with comment OK", last)
	}

	if claim.AllowedActions == nil || len(claim.AllowedActions) != 0 {
		t.Fatalf("allowedActions = %v, want an empty list", claim.AllowedActions)
	}

	if state := readState(t, pool, id); state.Status != "APPROVED" || state.Version != 1 || !state.Decided || state.History != 2 {
		t.Fatalf("stored = %+v, want APPROVED, version 1, decided, 2 history rows", state)
	}

	if queue := listClaims(t, router, token, "?scope=review"); queue.Total != 0 || len(queue.Data) != 0 {
		t.Fatalf("queue = %+v, want empty", queue)
	}
}

// An approval needs no comment: with no body, the history comment is null.
func TestApproveWithoutCommentStoresNoComment(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")

	claim := decodeClaim(t, decide(t, router, loginAs(t, router, "rev.malee"), id, "approve", `"0"`, "")).Data

	if claim.Status != "APPROVED" || len(claim.History) != 2 || claim.History[1].Comment != nil {
		t.Fatalf("claim = %+v, want APPROVED with a null comment", claim)
	}
}

// F04-TEST-08 (F04-AC-07, BR-01): rev.malee cannot decide her own claim, which is assigned to rev.prasert. The claim is
// unchanged.
func TestF04TEST08ReviewerCannotDecideOwnClaim(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "rev.malee")
	token := loginAs(t, router, "rev.malee")
	before := readState(t, pool, id)

	for _, action := range []string{"approve", "reject"} {
		decodeProblem(t, decide(t, router, token, id, action, `"0"`, `{"comment": "Fine by me"}`), http.StatusForbidden, "FORBIDDEN")
	}

	if after := readState(t, pool, id); after != before {
		t.Fatalf("stored = %+v after the refused decisions, want %+v", after, before)
	}
}

// F04-TEST-11 (F04-AC-08, BR-02, NFR-02): rev.somchai is not the assigned reviewer of req.anan's claim. Both decisions
// are 403 with no claim data, and the claim is unchanged.
func TestF04TEST11UnassignedReviewerCannotDecide(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	token := loginAs(t, router, "rev.somchai")
	before := readState(t, pool, id)

	for _, action := range []string{"approve", "reject"} {
		recorder := decide(t, router, token, id, action, `"0"`, `{"comment": "Not mine to decide"}`)
		decodeProblem(t, recorder, http.StatusForbidden, "FORBIDDEN")

		for _, name := range []string{"data", "claimNo", "items"} {
			if strings.Contains(recorder.Body.String(), `"`+name+`"`) {
				t.Fatalf("%s: body has %q: %s", action, name, recorder.Body)
			}
		}

		if strings.Contains(recorder.Body.String(), "Bangkok workshop travel") {
			t.Fatalf("%s: body holds the claim title: %s", action, recorder.Body)
		}
	}

	if after := readState(t, pool, id); after != before {
		t.Fatalf("stored = %+v after the refused decisions, want %+v", after, before)
	}
}

// F04-TEST-12 (F04-AC-09, BR-02, NFR-02): the requester cannot approve their own claim, and another requester cannot
// approve it either.
func TestF04TEST12RequestersCannotApprove(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	before := readState(t, pool, id)

	for _, username := range []string{"req.anan", "req.ploy"} {
		decodeProblem(t, decide(t, router, loginAs(t, router, username), id, "approve", `"0"`, ""), http.StatusForbidden, "FORBIDDEN")
	}

	if after := readState(t, pool, id); after != before {
		t.Fatalf("stored = %+v after the refused decisions, want %+v", after, before)
	}
}

// F04-TEST-13 (F04-AC-10, FR-06, ADR-0001): fin.wanida approves rev.prasert's 40,000.00 claim in one stage.
func TestF04TEST13FinanceOfficerApprovesDepartmentHeadClaim(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)

	body := claimBodyWith(itemJSON("ACCOMMODATION", "2026-11-03", "20000.00") + "," + itemJSON("ACCOMMODATION", "2026-11-04", "20000.00"))

	submitted := postClaim(t, router, loginAs(t, router, "rev.prasert"), body)
	if submitted.Code != http.StatusCreated {
		t.Fatalf("submit status = %d, want 201; body = %s", submitted.Code, submitted.Body)
	}

	var created claimResponse
	if err := json.Unmarshal(submitted.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if created.Data.Total != "40000.00" || created.Data.AssignedReviewer.Name != "Wanida" {
		t.Fatalf("claim = %+v, want 40000.00 assigned to Wanida", created.Data)
	}

	claim := decodeClaim(t, decide(t, router, loginAs(t, router, "fin.wanida"), created.Data.ID, "approve", `"0"`, "")).Data
	if claim.Status != "APPROVED" || claim.Version != 1 {
		t.Fatalf("claim = %+v, want APPROVED, version 1", claim)
	}
}

// F04-TEST-16 (F04-AC-13, BR-13): a well-formed If-Match that is not the current version is 409 CONFLICT, a missing
// header is 428, and a malformed one is 400. The claim is unchanged each time. The header is checked before the claim
// is read, so an unknown claim without it is 428 too.
func TestF04TEST16IfMatchIsRequiredAndMustBeCurrent(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	token := loginAs(t, router, "rev.malee")
	before := readState(t, pool, id)

	decodeProblem(t, decide(t, router, token, id, "approve", `"1"`, ""), http.StatusConflict, "CONFLICT")
	decodeProblem(t, decide(t, router, token, id, "approve", "", ""), http.StatusPreconditionRequired, "PRECONDITION_REQUIRED")

	for _, malformed := range []string{"abc", `W/"0"`, `"-1"`, `""`, `"0`, `"1.5"`, `"99999999999"`} {
		decodeProblem(t, decide(t, router, token, id, "approve", malformed, ""), http.StatusBadRequest, "BAD_REQUEST")
	}

	unknown := uuid.Must(uuid.NewV7()).String()
	decodeProblem(t, decide(t, router, token, unknown, "approve", "", ""), http.StatusPreconditionRequired, "PRECONDITION_REQUIRED")

	if after := readState(t, pool, id); after != before {
		t.Fatalf("stored = %+v after the refused decisions, want %+v", after, before)
	}
}

// An unknown claim is 404 and a malformed id is 400, for both decisions.
func TestDecisionOnUnknownAndMalformedIDs(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "rev.malee")

	for _, action := range []string{"approve", "reject"} {
		unknown := uuid.Must(uuid.NewV7()).String()
		decodeProblem(t, decide(t, router, token, unknown, action, `"0"`, ""), http.StatusNotFound, "NOT_FOUND")
		decodeProblem(t, decide(t, router, token, "abc", action, `"0"`, ""), http.StatusBadRequest, "BAD_REQUEST")
	}
}

// A body that is not JSON is a 400 before any decision.
func TestDecisionWithMalformedBodyIs400(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")

	decodeProblem(t, decide(t, router, loginAs(t, router, "rev.malee"), id, "approve", `"0"`, `{"comment": `), http.StatusBadRequest, "BAD_REQUEST")
}

// F04-TEST-05 (F04-AC-05, FR-06, FR-07): rejecting with a reason moves the claim to REJECTED, appends one REJECT
// history entry with the reason, and the requester sees the reason when they open the claim.
func TestF04TEST05RejectStoresTheReasonAndTheRequesterSeesIt(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")

	recorder := decide(t, router, loginAs(t, router, "rev.malee"), id, "reject", `"0"`, `{"comment": "Missing receipt"}`)
	claim := decodeClaim(t, recorder).Data

	if got := recorder.Header().Get("ETag"); got != `"1"` {
		t.Fatalf("ETag = %q, want \"1\"", got)
	}

	if claim.Status != "REJECTED" || claim.Version != 1 || claim.DecidedAt != "2026-11-05T03:00:00Z" || len(claim.History) != 2 {
		t.Fatalf("claim = %+v, want REJECTED, version 1, decided, 2 history entries", claim)
	}

	opened := openClaim(t, router, loginAs(t, router, "req.anan"), id).Data
	if opened.Status != "REJECTED" || len(opened.History) != 2 {
		t.Fatalf("requester sees %+v, want REJECTED with 2 history entries", opened)
	}

	last := opened.History[1]
	if last.Action != "REJECT" || last.FromStatus == nil || *last.FromStatus != "SUBMITTED" || last.ToStatus != "REJECTED" ||
		last.Actor.Name != "Malee S." || last.Comment == nil || *last.Comment != "Missing receipt" {
		t.Fatalf("last history = %+v, want REJECT SUBMITTED -> REJECTED by Malee S. with the reason", last)
	}

	if opened.AllowedActions == nil || len(opened.AllowedActions) != 0 {
		t.Fatalf("allowedActions = %v, want an empty list", opened.AllowedActions)
	}
}

// F04-TEST-07 (F04-AC-06, BR-04): a missing, blank, short, or long reason is 422 with one comment error, and the claim
// is unchanged. An approve comment over 500 characters is 422 too.
func TestF04TEST07CommentRuleIs422WithTheFieldError(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	token := loginAs(t, router, "rev.malee")
	before := readState(t, pool, id)

	cases := []struct {
		name    string
		action  string
		body    string
		message string
	}{
		{"reject without a body", "reject", "", "Enter a reason (5 to 500 characters)."},
		{"reject with an empty comment", "reject", `{"comment": ""}`, "Enter a reason (5 to 500 characters)."},
		{"reject with blanks", "reject", `{"comment": "   "}`, "Enter a reason (5 to 500 characters)."},
		{"reject with 4 characters", "reject", `{"comment": "abcd"}`, "Reason must be at least 5 characters."},
		{"reject with 501 characters", "reject", `{"comment": "` + strings.Repeat("a", 501) + `"}`, "Comment cannot be more than 500 characters."},
		{"approve with 501 characters", "approve", `{"comment": "` + strings.Repeat("a", 501) + `"}`, "Comment cannot be more than 500 characters."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := decide(t, router, token, id, tc.action, `"0"`, tc.body)
			decodeProblem(t, recorder, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

			want := []fieldError{{Field: "comment", Rule: "BR-04", Message: tc.message}}
			if got := problemErrors(t, recorder); !slices.Equal(got, want) {
				t.Fatalf("errors = %+v, want %+v", got, want)
			}

			if after := readState(t, pool, id); after != before {
				t.Fatalf("stored = %+v after the refused decision, want %+v", after, before)
			}
		})
	}
}

// F04-TEST-14 (F04-AC-10, BR-03): approving or rejecting a claim that is no longer SUBMITTED, with its current
// version, is 409 INVALID_TRANSITION with the status error. The claim is unchanged.
func TestF04TEST14DecidedClaimCannotBeDecidedAgain(t *testing.T) {
	const message = "Only a submitted claim can be approved or rejected."

	for _, first := range []struct{ action, body, status string }{
		{"approve", "", "APPROVED"},
		{"reject", `{"comment": "Missing receipt"}`, "REJECTED"},
	} {
		pool := pgtest.New(t)
		router := seededClaimRouter(t, pool)
		id := submittedBy(t, router, "req.anan")
		token := loginAs(t, router, "rev.malee")

		decided := decodeClaim(t, decide(t, router, token, id, first.action, `"0"`, first.body)).Data
		if decided.Status != first.status {
			t.Fatalf("status = %s, want %s", decided.Status, first.status)
		}

		before := readState(t, pool, id)

		for _, again := range []string{"approve", "reject"} {
			recorder := decide(t, router, token, id, again, `"1"`, `{"comment": "Second decision"}`)
			decodeProblem(t, recorder, http.StatusConflict, "INVALID_TRANSITION")

			want := []fieldError{{Field: "status", Rule: "BR-03", Message: message}}
			if got := problemErrors(t, recorder); !slices.Equal(got, want) {
				t.Fatalf("%s after %s: errors = %+v, want %+v", again, first.action, got, want)
			}
		}

		if after := readState(t, pool, id); after != before {
			t.Fatalf("stored = %+v after the refused decisions, want %+v", after, before)
		}
	}
}

// A decision with a stale version on a claim that another reviewer decided meanwhile is 409 CONFLICT: the screen is
// out of date (BR-13), which is what the caller must learn first.
func TestDecisionWithStaleVersionOnDecidedClaimIsConflict(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	token := loginAs(t, router, "rev.malee")

	decodeClaim(t, decide(t, router, token, id, "approve", `"0"`, ""))
	before := readState(t, pool, id)

	decodeProblem(t, decide(t, router, token, id, "reject", `"0"`, `{"comment": "Missing receipt"}`), http.StatusConflict, "CONFLICT")

	if after := readState(t, pool, id); after != before {
		t.Fatalf("stored = %+v after the refused decision, want %+v", after, before)
	}
}

// The status rule comes before the comment rule: a decided claim with a blank reason is INVALID_TRANSITION.
func TestDecidedClaimWithBlankReasonIsInvalidTransition(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	token := loginAs(t, router, "rev.malee")

	decodeClaim(t, decide(t, router, token, id, "approve", `"0"`, ""))

	decodeProblem(t, decide(t, router, token, id, "reject", `"1"`, ""), http.StatusConflict, "INVALID_TRANSITION")
}

// F04-AC-14 at the HTTP seam: when the history insert fails, the decision is 500 and the claim is unchanged.
func TestDecisionHistoryFailureIs500AndChangesNothing(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	token := loginAs(t, router, "rev.malee")
	before := readState(t, pool, id)

	// A test-only trigger makes every history insert fail.
	if _, err := pool.Exec(t.Context(), `
		CREATE FUNCTION fail_history_insert() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'forced history failure'; END; $$;
		CREATE TRIGGER fail_history_insert BEFORE INSERT ON claim_history
		FOR EACH ROW EXECUTE FUNCTION fail_history_insert();`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	decodeProblem(t, decide(t, router, token, id, "approve", `"0"`, ""), http.StatusInternalServerError, "INTERNAL")

	if after := readState(t, pool, id); after != before {
		t.Fatalf("stored = %+v after the failed decision, want %+v", after, before)
	}
}

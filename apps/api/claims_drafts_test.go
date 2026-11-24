package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

const draftsPath = "/api/v1/claims?draft=true"

func draftRequest(t *testing.T, router http.Handler, token, method, path, version, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if version != "" {
		request.Header.Set("If-Match", version)
	}
	request.AddCookie(&http.Cookie{Name: "sid", Value: token, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

func expectDraftResponse(t *testing.T, rec *httptest.ResponseRecorder, status int) api.Claim {
	t.Helper()

	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, status, rec.Body)
	}

	var response api.ClaimEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode claim: %v", err)
	}

	return response.Data
}

func TestF05TEST01SaveDraftWithoutSubmission(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")
	rec := draftRequest(t, router, token, http.MethodPost, draftsPath, "", twoItemsBody(`"total": "1.00",`))
	got := expectDraftResponse(t, rec, http.StatusCreated)

	if got.Status != api.ClaimStatusDraft || got.Total != 125000 || got.Version != 0 || got.ClaimNo == "" ||
		got.AssignedReviewer != nil || got.FirstSubmittedAt != nil || got.SubmittedAt != nil ||
		got.History == nil || len(got.History) != 0 || rec.Header().Get("ETag") != `"0"` ||
		rec.Header().Get("Location") != "/api/v1/claims/"+got.Id {
		t.Fatalf("unexpected draft: %s", rec.Body)
	}
	if n := rowCount(t, pool, "SELECT count(*) FROM claim_history"); n != 0 {
		t.Fatalf("history rows = %d, want 0", n)
	}
}

func TestF05TEST03FinanceOfficerCannotSaveDraft(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "fin.wanida")
	rec := draftRequest(t, router, token, http.MethodPost, draftsPath, "", twoItemsBody(""))
	if rec.Code != http.StatusForbidden || rowCount(t, pool, "SELECT count(*) FROM claims") != 0 {
		t.Fatalf("finance draft: status %d, body %s", rec.Code, rec.Body)
	}
}

func TestF05TEST09EditRequiresOwnerAndEditableStatus(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	owner := loginAs(t, router, "req.anan")
	other := loginAs(t, router, "req.ploy")
	draft := expectDraftResponse(t, draftRequest(t, router, owner, http.MethodPost, draftsPath, "", twoItemsBody("")), http.StatusCreated)
	submitted := submittedBy(t, router, "req.anan")

	rec := draftRequest(t, router, other, http.MethodPut, "/api/v1/claims/"+draft.Id, `"0"`, twoItemsBody(""))
	decodeProblem(t, rec, http.StatusForbidden, "FORBIDDEN")
	rec = draftRequest(t, router, owner, http.MethodPut, "/api/v1/claims/"+submitted, `"0"`, twoItemsBody(""))
	expectTransition(t, rec, "BR-10", "Only a draft or a returned claim can be edited.")
}

func TestF05TEST19DeleteRequiresOwnerAndDraftStatus(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	owner := loginAs(t, router, "req.anan")
	other := loginAs(t, router, "req.ploy")
	draft := expectDraftResponse(t, draftRequest(t, router, owner, http.MethodPost, draftsPath, "", twoItemsBody("")), http.StatusCreated)
	submitted := submittedBy(t, router, "req.anan")

	rec := draftRequest(t, router, other, http.MethodDelete, "/api/v1/claims/"+draft.Id, `"0"`, "")
	decodeProblem(t, rec, http.StatusForbidden, "FORBIDDEN")
	rec = draftRequest(t, router, owner, http.MethodDelete, "/api/v1/claims/"+submitted, `"0"`, "")
	expectTransition(t, rec, "BR-10", "Only a draft can be deleted.")

	rec = draftRequest(t, router, owner, http.MethodDelete, "/api/v1/claims/"+draft.Id, `"0"`, "")
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body)
	}
	decodeProblem(t, getWithSession(t, router, "/api/v1/claims/"+draft.Id, owner), http.StatusNotFound, "NOT_FOUND")
	if n := rowCount(t, pool, "SELECT count(*) FROM line_items"); n != 3 {
		t.Fatalf("line items = %d, want only the submitted claim's 3 items", n)
	}
}

func expectTransition(t *testing.T, rec *httptest.ResponseRecorder, rule, message string) {
	t.Helper()

	decodeProblem(t, rec, http.StatusConflict, "INVALID_TRANSITION")
	if got := problemErrors(t, rec); len(got) != 1 || got[0] != (fieldError{Field: "status", Rule: rule, Message: message}) {
		t.Fatalf("transition errors = %+v", got)
	}
}

var draftHTTPNow = time.Date(2026, time.November, 25, 10, 0, 0, 0, time.FixedZone("ICT", 7*60*60))

func draftRouter(t *testing.T, pool *pgxpool.Pool) http.Handler {
	t.Helper()

	seedDemoUsers(t, pool)
	return newRouterAt(discardLogger(), testEnv(0), pool, func() time.Time { return draftHTTPNow })
}

func TestF05TEST05OnlyOwnerCanOpenDraft(t *testing.T) {
	pool := pgtest.New(t)
	router := draftRouter(t, pool)
	owner := loginAs(t, router, "req.anan")
	draft := expectDraftResponse(t, draftRequest(t, router, owner, http.MethodPost, draftsPath, "", twoItemsBody("")), http.StatusCreated)
	path := "/api/v1/claims/" + draft.Id
	opened := expectDraftResponse(t, getWithSession(t, router, path, owner), http.StatusOK)
	if !slices.Equal(opened.AllowedActions, []string{"edit", "delete", "submit"}) {
		t.Fatalf("allowed actions = %v", opened.AllowedActions)
	}

	mine := listClaims(t, router, owner, "?scope=mine")
	if mine.Total != 1 || mine.Data[0].ID != draft.Id || mine.Data[0].Status != "DRAFT" || mine.Data[0].SubmittedAt != "" {
		t.Fatalf("My Claims = %+v", mine)
	}

	for _, name := range []string{"req.ploy", "rev.malee", "fin.wanida"} {
		token := loginAs(t, router, name)
		rec := getWithSession(t, router, path, token)
		decodeProblem(t, rec, http.StatusForbidden, "FORBIDDEN")
		if strings.Contains(rec.Body.String(), draft.ClaimNo) || strings.Contains(rec.Body.String(), draft.Title) {
			t.Fatalf("claim data visible to %s: %s", name, rec.Body)
		}
		decodeProblem(t, draftRequest(t, router, token, http.MethodPost, path+"/submit", `"0"`, ""), http.StatusForbidden, "FORBIDDEN")
	}
}

func TestF05TEST07EditReplacesItemsAndRecalculatesTotal(t *testing.T) {
	pool := pgtest.New(t)
	router := draftRouter(t, pool)
	token := loginAs(t, router, "req.anan")
	draft := expectDraftResponse(t, draftRequest(t, router, token, http.MethodPost, draftsPath, "", twoItemsBody("")), http.StatusCreated)
	path := "/api/v1/claims/" + draft.Id
	body := `{"title":"Edited title","purpose":"Edited purpose","total":"1.00","items":[` + strings.Join([]string{
		itemJSON("OTHER", "2026-11-25", "100.00"),
		itemJSON("TRAVEL", "2026-11-26", "200.00"),
		itemJSON("MEALS", "2026-08-01", "300.00"),
	}, ",") + "]}"
	rec := draftRequest(t, router, token, http.MethodPut, path, `"0"`, body)
	got := expectDraftResponse(t, rec, http.StatusOK)
	if got.Title != "Edited title" || got.Purpose != "Edited purpose" ||
		got.Total != 60000 || got.Version != 1 || got.Status != api.ClaimStatusDraft ||
		len(got.Items) != 3 || len(got.History) != 0 || rec.Header().Get("ETag") != `"1"` {
		t.Fatalf("unexpected edit: %s", rec.Body)
	}
	for i, category := range []api.ExpenseCategory{"OTHER", "TRAVEL", "MEALS"} {
		if got.Items[i].LineNo != i+1 || got.Items[i].Category != category {
			t.Fatalf("edited items = %+v", got.Items)
		}
	}
	if stored := expectDraftResponse(t, getWithSession(t, router, path, token), http.StatusOK); !reflect.DeepEqual(stored, got) {
		t.Fatalf("stored claim differs from the edit response: %+v", stored)
	}

	bad := draftRequest(t, router, token, http.MethodPut, path, `"1"`, `{"purpose":"Missing title","items":[]}`)
	decodeProblem(t, bad, http.StatusBadRequest, "BAD_REQUEST")
	if fields := problemErrors(t, bad); len(fields) != 1 || fields[0].Field != "title" {
		t.Fatalf("edit shape errors = %+v", fields)
	}
}

func TestF05TEST10DraftMutationsRequireCurrentIfMatch(t *testing.T) {
	pool := pgtest.New(t)
	router := draftRouter(t, pool)
	token := loginAs(t, router, "req.anan")
	draft := expectDraftResponse(t, draftRequest(t, router, token, http.MethodPost, draftsPath, "", twoItemsBody("")), http.StatusCreated)
	path := "/api/v1/claims/" + draft.Id

	for _, op := range []struct{ method, suffix, body string }{
		{http.MethodPut, "", twoItemsBody("")},
		{http.MethodDelete, "", ""},
		{http.MethodPost, "/submit", ""},
	} {
		t.Run(op.method, func(t *testing.T) {
			for _, test := range []struct {
				version, code string
				status        int
			}{
				{`"1"`, "CONFLICT", http.StatusConflict},
				{"", "PRECONDITION_REQUIRED", http.StatusPreconditionRequired},
				{"abc", "BAD_REQUEST", http.StatusBadRequest},
			} {
				rec := draftRequest(t, router, token, op.method, path+op.suffix, test.version, op.body)
				decodeProblem(t, rec, test.status, test.code)
			}
			missing := "/api/v1/claims/00000000-0000-0000-0000-000000000001" + op.suffix
			decodeProblem(t, draftRequest(t, router, token, op.method, missing, "", op.body), http.StatusPreconditionRequired, "PRECONDITION_REQUIRED")
			decodeProblem(t, draftRequest(t, router, token, op.method, missing, `"0"`, op.body), http.StatusNotFound, "NOT_FOUND")
		})
	}

	if stored := expectDraftResponse(t, getWithSession(t, router, path, token), http.StatusOK); !reflect.DeepEqual(stored, draft) {
		t.Fatalf("refused mutation changed draft: %+v", stored)
	}
}

func TestF05TEST11SubmitDraftRecordsFirstSubmissionAndQueuesClaim(t *testing.T) {
	pool := pgtest.New(t)
	router := draftRouter(t, pool)
	owner := loginAs(t, router, "req.anan")
	draft := expectDraftResponse(t, draftRequest(t, router, owner, http.MethodPost, draftsPath, "", twoItemsBody("")), http.StatusCreated)
	path := "/api/v1/claims/" + draft.Id + "/submit"
	rec := draftRequest(t, router, owner, http.MethodPost, path, `"0"`, "")
	got := expectDraftResponse(t, rec, http.StatusOK)
	if got.Status != api.ClaimStatusSubmitted || got.Version != 1 || rec.Header().Get("ETag") != `"1"` ||
		got.FirstSubmittedAt == nil || !got.FirstSubmittedAt.Equal(draftHTTPNow) ||
		got.SubmittedAt == nil || !got.SubmittedAt.Equal(draftHTTPNow) ||
		got.AssignedReviewer == nil || got.AssignedReviewer.Name != "Malee S." || len(got.History) != 1 {
		t.Fatalf("unexpected submission: %s", rec.Body)
	}
	entry := got.History[0]
	if entry.FromStatus == nil || *entry.FromStatus != api.ClaimStatusDraft || entry.Action != api.ClaimActionSubmit ||
		entry.ToStatus != api.ClaimStatusSubmitted || entry.Actor.Id != got.Requester.Id || entry.Comment != nil ||
		!entry.CreatedAt.Equal(draftHTTPNow) {
		t.Fatalf("submission history = %+v", entry)
	}
	queue := listClaims(t, router, loginAs(t, router, "rev.malee"), "?scope=review")
	if queue.Total != 1 || queue.Data[0].ID != draft.Id {
		t.Fatalf("review queue = %+v", queue)
	}

	decodeProblem(t, draftRequest(t, router, owner, http.MethodPost, path, `"0"`, ""), http.StatusConflict, "CONFLICT")
	decodeProblem(t, draftRequest(t, router, owner, http.MethodPut, "/api/v1/claims/"+draft.Id, `"0"`, twoItemsBody("")), http.StatusConflict, "CONFLICT")
	decodeProblem(t, draftRequest(t, router, owner, http.MethodDelete, "/api/v1/claims/"+draft.Id, `"0"`, ""), http.StatusConflict, "CONFLICT")
}

func TestF05TEST13SubmitDraftReportsExpenseDateField(t *testing.T) {
	pool := pgtest.New(t)
	router := draftRouter(t, pool)
	owner := loginAs(t, router, "req.anan")
	body := claimBodyWith(itemJSON("TRAVEL", "2026-11-26", "250.00"))
	draft := expectDraftResponse(t, draftRequest(t, router, owner, http.MethodPost, draftsPath, "", body), http.StatusCreated)
	rec := draftRequest(t, router, owner, http.MethodPost, "/api/v1/claims/"+draft.Id+"/submit", `"0"`, "")
	decodeProblem(t, rec, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	want := []fieldError{{Field: "items[0].expenseDate", Rule: "BR-07", Message: "Expense date cannot be in the future."}}
	if got := problemErrors(t, rec); !slices.Equal(got, want) {
		t.Fatalf("errors = %+v, want %+v", got, want)
	}
}

func TestF05TEST16OnlyDraftOrReturnedCanBeSubmitted(t *testing.T) {
	pool := pgtest.New(t)
	router := draftRouter(t, pool)
	token := loginAs(t, router, "req.anan")
	for _, status := range []string{"SUBMITTED", "APPROVED", "REJECTED"} {
		t.Run(status, func(t *testing.T) {
			id := storedClaim(t, pool, "req.anan", status, draftHTTPNow).String()
			before := readState(t, pool, id)
			rec := draftRequest(t, router, token, http.MethodPost, "/api/v1/claims/"+id+"/submit", `"0"`, "")
			expectTransition(t, rec, "BR-03", "Only a draft or a returned claim can be submitted.")
			if got := readState(t, pool, id); got != before {
				t.Fatalf("refused submission changed claim: %+v", got)
			}
		})
	}
}

func TestF05TEST20HistoryPreventsDirectClaimDeletion(t *testing.T) {
	pool := pgtest.New(t)
	router := draftRouter(t, pool)
	token := loginAs(t, router, "req.anan")
	claim := expectDraftResponse(t, postClaim(t, router, token, twoItemsBody("")), http.StatusCreated)
	if _, err := pool.Exec(t.Context(), "DELETE FROM claims WHERE id = $1", claim.Id); err == nil {
		t.Fatal("direct SQL deleted a claim with history")
	}
	got := expectDraftResponse(t, getWithSession(t, router, "/api/v1/claims/"+claim.Id, token), http.StatusOK)
	if !reflect.DeepEqual(got, claim) {
		t.Fatalf("refused SQL delete changed claim: %+v", got)
	}
}

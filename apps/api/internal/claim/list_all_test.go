package claim

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// listed is a claim that a ScopeAll test stores: who asked for it, its status, title, line item categories, and times.
// firstSubmittedAt and submittedAt are nil for a claim that was never submitted.
type listed struct {
	requester        uuid.UUID
	status           db.ClaimStatus
	title            string
	categories       []db.ExpenseCategory
	createdAt        time.Time
	firstSubmittedAt *time.Time
	submittedAt      *time.Time
}

// storeListed inserts c with one line item per category (TRAVEL when none) and returns its id. A claim that is not a
// DRAFT is assigned to its requester's reviewer.
func storeListed(t *testing.T, pool *pgxpool.Pool, c *listed) uuid.UUID {
	t.Helper()

	if c.title == "" {
		c.title = "Listed claim"
	}
	if len(c.categories) == 0 {
		c.categories = []db.ExpenseCategory{db.ExpenseCategoryTRAVEL}
	}

	id := uuid.Must(uuid.NewV7())
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO claims (id, claim_no, requester_id, assigned_reviewer_id, title, purpose, status, total_satang,
		                    first_submitted_at, submitted_at, created_at, updated_at)
		SELECT $1, 'CLM-2026-' || lpad(nextval('claim_no_seq')::text, 6, '0'), u.id,
		       CASE WHEN $3::claim_status <> 'DRAFT' THEN u.reviewer_id END, $4, 'Purpose', $3, 10000, $5, $6, $7, $7
		FROM "user" u WHERE u.id = $2`,
		id, c.requester, c.status, c.title, c.firstSubmittedAt, c.submittedAt, c.createdAt); err != nil {
		t.Fatalf("insert claim %q: %v", c.title, err)
	}

	for i, category := range c.categories {
		if _, err := pool.Exec(t.Context(), `
			INSERT INTO line_items (id, claim_id, line_no, category, description, expense_date, amount_satang, receipt_ref)
			VALUES ($1, $2, $3, $4, 'Item', '2026-11-01', 100, 'REF')`,
			uuid.Must(uuid.NewV7()), id, i+1, category); err != nil {
			t.Fatalf("insert line item of %q: %v", c.title, err)
		}
	}

	return id
}

// submittedListed is a claim of requester in status, created and first submitted at at.
func submittedListed(requester uuid.UUID, status db.ClaimStatus, at time.Time) *listed {
	return &listed{requester: requester, status: status, createdAt: at, firstSubmittedAt: &at, submittedAt: &at}
}

// listAll returns the ids of every claim on the first page of ScopeAll for actor with filters, and the total.
func listAll(t *testing.T, service Service, actor Actor, filters Filters) (ids []uuid.UUID, total int) {
	t.Helper()

	page, err := service.List(t.Context(), actor, &Query{Scope: ScopeAll, Filters: filters, Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("List(ScopeAll, %+v): %v", filters, err)
	}

	ids = make([]uuid.UUID, 0, len(page.Claims))
	for i := range page.Claims {
		ids = append(ids, page.Claims[i].ID)
	}

	return ids, page.Total
}

// sameIDs reports whether got and want hold the same ids in any order.
func sameIDs(got, want []uuid.UUID) bool {
	got, want = slices.Clone(got), slices.Clone(want)
	compare := func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) }
	slices.SortFunc(got, compare)
	slices.SortFunc(want, compare)

	return slices.Equal(got, want)
}

// day returns the calendar date d of November 2026.
func day(d int) *time.Time {
	date := time.Date(2026, time.November, d, 0, 0, 0, 0, time.UTC)

	return &date
}

// F09-TEST-03 (F09-AC-03): from and to are inclusive Asia/Bangkok dates on first_submitted_at, or created_at for a claim
// that was never submitted.
func TestF09TEST03DateRangeUsesBangkokDates(t *testing.T) {
	pool := pgtest.New(t)
	reviewer := member(t, pool, "rev.malee", "reviewer", uuid.Nil)
	anan := member(t, pool, "req.anan", "requester", reviewer)
	service := newService(pool)

	lateOn5th := storeListed(t, pool, submittedListed(anan, db.ClaimStatusSUBMITTED, time.Date(2026, time.November, 5, 23, 59, 59, 0, bangkok)))
	startOf6th := storeListed(t, pool, submittedListed(anan, db.ClaimStatusSUBMITTED, time.Date(2026, time.November, 6, 0, 0, 0, 0, bangkok)))
	earlyOn5th := storeListed(t, pool, submittedListed(anan, db.ClaimStatusAPPROVED, time.Date(2026, time.November, 4, 17, 30, 0, 0, time.UTC)))

	first := time.Date(2026, time.November, 1, 10, 0, 0, 0, bangkok)
	latest := time.Date(2026, time.November, 6, 10, 0, 0, 0, bangkok)
	resubmitted := storeListed(t, pool, &listed{
		requester: anan, status: db.ClaimStatusSUBMITTED, createdAt: first, firstSubmittedAt: &first, submittedAt: &latest,
	})
	draft := storeListed(t, pool, &listed{
		requester: anan, status: db.ClaimStatusDRAFT, createdAt: time.Date(2026, time.November, 5, 9, 0, 0, 0, bangkok),
	})

	cases := []struct {
		name    string
		filters Filters
		want    []uuid.UUID
	}{
		{"from = to = 5 Nov", Filters{From: day(5), To: day(5)}, []uuid.UUID{lateOn5th, earlyOn5th, draft}},
		{"from 6 Nov", Filters{From: day(6)}, []uuid.UUID{startOf6th}},
		{"to 5 Nov", Filters{To: day(5)}, []uuid.UUID{lateOn5th, earlyOn5th, resubmitted, draft}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, total := listAll(t, service, Actor{ID: anan, Role: "requester"}, tc.filters)
			if !sameIDs(got, tc.want) || total != len(tc.want) {
				t.Fatalf("List = %v (total %d), want %v", got, total, tc.want)
			}
		})
	}
}

// F09-TEST-08 (F09-AC-07, SR-01): for each seed user, ScopeAll with no filters lists exactly the claims that Allowed
// marks visible, and with a status or category filter it lists none that Allowed hides.
func TestF09TEST08ListAgreesWithPolicyVisibility(t *testing.T) {
	pool := pgtest.New(t)
	wanida := member(t, pool, "fin.wanida", "finance", uuid.Nil)
	prasert := member(t, pool, "rev.prasert", "reviewer", wanida)
	malee := member(t, pool, "rev.malee", "reviewer", prasert)
	somchai := member(t, pool, "rev.somchai", "reviewer", prasert)
	actors := []Actor{
		{ID: wanida, Role: RoleFinance},
		{ID: prasert, Role: "reviewer"},
		{ID: malee, Role: "reviewer"},
		{ID: somchai, Role: "reviewer"},
		{ID: member(t, pool, "req.anan", "requester", malee), Role: "requester"},
		{ID: member(t, pool, "req.ploy", "requester", malee), Role: "requester"},
		{ID: member(t, pool, "req.niran", "requester", somchai), Role: "requester"},
	}
	service := newService(pool)

	statuses := []db.ClaimStatus{
		db.ClaimStatusDRAFT, db.ClaimStatusSUBMITTED, db.ClaimStatusRETURNED, db.ClaimStatusPENDINGFINANCE,
		db.ClaimStatusAPPROVED, db.ClaimStatusREJECTED, db.ClaimStatusWITHDRAWN,
	}
	var all []uuid.UUID
	at := time.Date(2026, time.November, 1, 9, 0, 0, 0, bangkok)
	for _, requester := range actors {
		for _, status := range statuses {
			// fin.wanida has no reviewer, so she can only hold a draft.
			if requester.ID == wanida && status != db.ClaimStatusDRAFT {
				continue
			}
			at = at.Add(time.Minute)
			claim := submittedListed(requester.ID, status, at)
			if status == db.ClaimStatusDRAFT {
				claim.firstSubmittedAt, claim.submittedAt = nil, nil
			}
			claim.categories = []db.ExpenseCategory{db.ExpenseCategoryMEALS}
			if status == db.ClaimStatusAPPROVED {
				claim.categories = []db.ExpenseCategory{db.ExpenseCategoryTRAVEL, db.ExpenseCategoryMEALS}
			}
			all = append(all, storeListed(t, pool, claim))
		}
	}

	approved, meals := db.ClaimStatusAPPROVED, db.ExpenseCategoryMEALS
	for _, actor := range actors {
		visible := map[uuid.UUID]bool{}
		var want []uuid.UUID
		for _, id := range all {
			if _, err := service.Get(t.Context(), actor, id); err == nil {
				visible[id] = true
				want = append(want, id)
			}
		}

		got, total := listAll(t, service, actor, Filters{})
		if !sameIDs(got, want) || total != len(want) {
			t.Fatalf("actor %v: List = %v (total %d), want the %d visible claims %v", actor, got, total, len(want), want)
		}

		for _, filters := range []Filters{{Status: &approved}, {Category: &meals}, {Status: &approved, Category: &meals}} {
			got, _ := listAll(t, service, actor, filters)
			for _, id := range got {
				if !visible[id] {
					t.Fatalf("actor %v: List(%+v) holds %s, which Allowed hides", actor, filters, id)
				}
			}
		}
	}
}

// F09-TEST-02 (F09-AC-02): status matches exactly, and category matches a claim with any line item in it, once.
func TestF09TEST02StatusAndCategoryFilters(t *testing.T) {
	pool := pgtest.New(t)
	reviewer := member(t, pool, "rev.malee", "reviewer", uuid.Nil)
	anan := member(t, pool, "req.anan", "requester", reviewer)
	service := newService(pool)

	// Every claim but the mixed one has only a SUPPLIES item.
	supplies := func(status db.ClaimStatus, at time.Time) *listed {
		claim := submittedListed(anan, status, at)
		claim.categories = []db.ExpenseCategory{db.ExpenseCategorySUPPLIES}
		return claim
	}
	at := time.Date(2026, time.November, 2, 9, 0, 0, 0, bangkok)
	approved := storeListed(t, pool, supplies(db.ClaimStatusAPPROVED, at))
	mixed := submittedListed(anan, db.ClaimStatusSUBMITTED, at.Add(time.Hour))
	mixed.categories = []db.ExpenseCategory{db.ExpenseCategoryMEALS, db.ExpenseCategoryTRAVEL, db.ExpenseCategoryMEALS}
	mixedID := storeListed(t, pool, mixed)
	storeListed(t, pool, supplies(db.ClaimStatusREJECTED, at.Add(2*time.Hour)))
	storeListed(t, pool, supplies(db.ClaimStatusRETURNED, at.Add(3*time.Hour)))

	status, meals, travel := db.ClaimStatusAPPROVED, db.ExpenseCategoryMEALS, db.ExpenseCategoryTRAVEL
	cases := []struct {
		name    string
		filters Filters
		want    []uuid.UUID
	}{
		{"status APPROVED", Filters{Status: &status}, []uuid.UUID{approved}},
		{"category MEALS", Filters{Category: &meals}, []uuid.UUID{mixedID}},
		{"category TRAVEL", Filters{Category: &travel}, []uuid.UUID{mixedID}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, total := listAll(t, service, Actor{ID: anan, Role: "requester"}, tc.filters)
			if !sameIDs(got, tc.want) || total != len(tc.want) {
				t.Fatalf("List = %v (total %d), want %v", got, total, tc.want)
			}
		})
	}
}

// F09-TEST-04 (F09-AC-04): q is a trimmed, case-insensitive title substring, and %, _, and \ match literally.
func TestF09TEST04TitleSearchIsLiteral(t *testing.T) {
	pool := pgtest.New(t)
	reviewer := member(t, pool, "rev.malee", "reviewer", uuid.Nil)
	anan := member(t, pool, "req.anan", "requester", reviewer)
	service := newService(pool)

	ids := map[string]uuid.UUID{}
	at := time.Date(2026, time.November, 2, 9, 0, 0, 0, bangkok)
	for i, title := range []string{"Taxi to client site", "100% refund", "Hotel_A", `Path C:\temp`} {
		claim := submittedListed(anan, db.ClaimStatusSUBMITTED, at.Add(time.Duration(i)*time.Hour))
		claim.title = title
		ids[title] = storeListed(t, pool, claim)
	}

	for _, tc := range []struct{ q, title string }{
		{" TAXI ", "Taxi to client site"}, {"100%", "100% refund"}, {"%", "100% refund"}, {"_", "Hotel_A"}, {`\`, `Path C:\temp`},
	} {
		got, total := listAll(t, service, Actor{ID: anan, Role: "requester"}, Filters{Q: tc.q})
		if total != 1 || len(got) != 1 || got[0] != ids[tc.title] {
			t.Fatalf("q %q: List = %v (total %d), want only %q", tc.q, got, total, tc.title)
		}
	}

	if got, total := listAll(t, service, Actor{ID: anan, Role: "requester"}, Filters{Q: "   "}); total != 4 || len(got) != 4 {
		t.Fatalf("blank q: List = %v (total %d), want all 4 claims", got, total)
	}
}

// F09-TEST-05 (F09-AC-05): filters combine with AND, and total counts the filtered set on every page.
func TestF09TEST05FiltersCombineAndTotalIsFiltered(t *testing.T) {
	pool := pgtest.New(t)
	reviewer := member(t, pool, "rev.malee", "reviewer", uuid.Nil)
	anan := member(t, pool, "req.anan", "requester", reviewer)
	service := newService(pool)

	inRange := time.Date(2026, time.November, 5, 9, 0, 0, 0, bangkok)
	match := func(at time.Time) *listed {
		claim := submittedListed(anan, db.ClaimStatusAPPROVED, at)
		claim.title = "Taxi to client site"
		claim.categories = []db.ExpenseCategory{db.ExpenseCategoryTRAVEL}
		return claim
	}
	older := storeListed(t, pool, match(inRange))
	newer := storeListed(t, pool, match(inRange.Add(time.Hour)))

	wrongStatus := match(inRange)
	wrongStatus.status = db.ClaimStatusREJECTED
	wrongCategory := match(inRange)
	wrongCategory.categories = []db.ExpenseCategory{db.ExpenseCategoryMEALS}
	wrongDate := match(inRange.AddDate(0, 0, 3))
	wrongTitle := match(inRange)
	wrongTitle.title = "Hotel"
	for _, claim := range []*listed{wrongStatus, wrongCategory, wrongDate, wrongTitle} {
		storeListed(t, pool, claim)
	}

	status, category := db.ClaimStatusAPPROVED, db.ExpenseCategoryTRAVEL
	filters := Filters{Status: &status, Category: &category, From: day(4), To: day(6), Q: "taxi"}
	for page, want := range []uuid.UUID{newer, older} {
		got, err := service.List(t.Context(), Actor{ID: anan, Role: "requester"}, &Query{Scope: ScopeAll, Filters: filters, Page: page + 1, PageSize: 1})
		if err != nil {
			t.Fatalf("List page %d: %v", page+1, err)
		}
		if got.Total != 2 || len(got.Claims) != 1 || got.Claims[0].ID != want {
			t.Fatalf("page %d = %+v, want only %s of total 2", page+1, got, want)
		}
	}
}

// F02-TEST-14 (F02-AC-09, BR-15): a Finance Officer has no assigned reviewer, so POST /claims gets 403 and stores nothing.
func TestF02TEST14FinanceOfficerCannotCreateClaim(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "fin.wanida")

	decodeProblem(t, postClaim(t, router, token, twoItemsBody("")), http.StatusForbidden, "FORBIDDEN")

	if got := rowCount(t, pool, "SELECT count(*) FROM claims"); got != 0 {
		t.Fatalf("claims = %d, want 0", got)
	}
}

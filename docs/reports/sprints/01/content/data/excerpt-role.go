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

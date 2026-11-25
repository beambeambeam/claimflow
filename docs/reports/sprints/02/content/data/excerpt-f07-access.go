// F07-TEST-04 (F07-AC-04, FR-11, NFR-02): only the owner can withdraw. The assigned reviewer gets 403, and so does another
// requester, with no claim data. The claim is unchanged.
func TestF07TEST04OnlyTheOwnerCanWithdraw(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	before := readState(t, pool, id)

	for _, username := range []string{"rev.malee", "req.ploy"} {
		recorder := decide(t, router, loginAs(t, router, username), id, "withdraw", `"0"`, "")
		decodeProblem(t, recorder, http.StatusForbidden, "FORBIDDEN")

		for _, leaked := range []string{`"data"`, "claimNo", "Bangkok workshop travel"} {
			if strings.Contains(recorder.Body.String(), leaked) {
				t.Fatalf("%s: body holds %q: %s", username, leaked, recorder.Body)
			}
		}
	}

	if after := readState(t, pool, id); after != before {
		t.Fatalf("stored = %+v after the refused withdrawals, want %+v", after, before)
	}
}

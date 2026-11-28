// F06-TEST-07 (F06-AC-05, BR-14): a claim that was returned MaxReturns times and resubmitted cannot be returned again:
// the refusal is an invalid transition for the return limit. The reviewer can still approve it.
func TestF06TEST07FourthReturnIsRefusedAndApproveStillWorks(t *testing.T) {
	pool := pgtest.New(t)
	created, reviewer := submittedClaim(t, pool)
	service := newService(pool)

	current := created
	for range MaxReturns {
		current = returnAndResubmit(t, pool, reviewer, &current)
	}

	if current.Status != db.ClaimStatusSUBMITTED || current.ReturnCount != MaxReturns {
		t.Fatalf("claim = %s with return count %d, want SUBMITTED with %d", current.Status, current.ReturnCount, MaxReturns)
	}

	_, err := service.Act(t.Context(), reviewer, current.ID, current.Version, ActionReturn, "One more thing.")
	if !errors.Is(err, ErrInvalidTransition) || !errors.Is(err, ErrReturnLimit) {
		t.Fatalf("Act return error = %v, want ErrInvalidTransition for the return limit", err)
	}

	after, err := service.Get(t.Context(), reviewer, current.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if after.Status != db.ClaimStatusSUBMITTED || after.ReturnCount != MaxReturns || after.Version != current.Version {
		t.Fatalf("claim = %+v after the refused return, want it unchanged", after)
	}

	approved, err := service.Act(t.Context(), reviewer, current.ID, current.Version, ActionApprove, "")
	if err != nil || approved.Status != db.ClaimStatusAPPROVED {
		t.Fatalf("Act approve = %s, %v; want APPROVED", approved.Status, err)
	}
}

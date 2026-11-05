# 02: Reject, invalid transitions, and concurrent decisions

**What to build:** A reviewer rejects a claim with a required reason of 5–500 characters, entered in a dialog. A decision on a claim that is no longer `Submitted` is refused and writes no history. When two decisions race, exactly one wins; the other reviewer sees "This claim was changed by someone else – reloaded" and the claim refetches.

**Blocked by:** F-04/01 Review Queue and approve

**Status:** ready-for-agent

- [ ] Transition table adds `SUBMITTED + REJECT → REJECTED`; `decided_at` is set for both decisions.
- [ ] Reason rule (BR-04): blank, 4, 5, 500, 501 characters tested; violations → 422 with the field and rule.
- [ ] A decision on `APPROVED` or `REJECTED` → 409 `INVALID_TRANSITION` with rule BR-03, and no history row (BR-03).
- [ ] Two goroutines `Act` on the same claim with the same version → one success, one `ErrConflict`, one history row (BR-13).
- [ ] Forcing the history insert to fail leaves the status unchanged (NFR-06).
- [ ] `POST /claims/{id}/reject` with `If-Match`.
- [ ] Web: Reject dialog with required reason and field message; 409 toast and refetch; requester sees the reason on a rejected claim.

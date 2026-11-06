# 02: Finance Queue and final decisions

**What to build:** A Finance Officer opens the Finance Queue and sees every `PendingFinance` claim, oldest first. On the claim detail they final-approve (→ `Approved`) or final-reject with a required reason (→ `Rejected`); there is no Return. Both stages appear in history. From now on a Finance Officer can see every non-draft claim in the department.

**Blocked by:** F-08/01 High-value approval goes to Pending finance

**Status:** ready-for-agent

- [ ] Transition table adds `PENDING_FINANCE + FINAL_APPROVE → APPROVED` and `+ FINAL_REJECT → REJECTED`.
- [ ] Policy adds `final-approve`, `final-reject` (role `finance`, not own, `PENDING_FINANCE`); Finance sees every non-draft claim, never a draft.
- [ ] `GET /claims?scope=finance` requires `finance:list`; reviewer → 403.
- [ ] `POST /claims/{id}/final-approve` `{ comment? }` and `/final-reject` `{ comment }` with `If-Match`; BR-04 on final reject.
- [ ] Reviewer or requester final decision → 403; final decision on `SUBMITTED` → BR-03; two concurrent final decisions → one 409 `CONFLICT`.
- [ ] Policy matrix gains Finance rows for every status.
- [ ] Web: Finance Queue route and header link; final-decision panel without Return.
- [ ] E2E: 15,000.00 claim → reviewer approves → Finance final-approves → requester sees `Approved` with both decisions in history.

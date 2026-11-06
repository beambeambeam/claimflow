# F-05: Drafts

Status: ready-for-agent Sprint: 2 Requirements: FR-03, FR-07, FR-08, BR-03, BR-05, BR-06, BR-07, BR-08, BR-10, BR-12, BR-15, BR-17, NFR-08 Verifies: SR-01, SR-04, SR-06 (new paths) Depends on: F-02 Design: SDS §3.2 `SAVE_DRAFT`, `SUBMIT`, `DELETE`; §3.3; §4.3 `POST /claims?draft=true`, `PUT`, `DELETE`, `submit`; §5.2; §7.1 Edit Claim

## Problem Statement

A requester often collects receipts over several days. Today they must enter the whole claim in one sitting or lose their work. They need to save a claim, come back to it, change it, and submit it when it is ready, or throw it away if it is no longer needed.

## Solution

A requester can save a claim as a `Draft` instead of submitting it. Only the requester can see their drafts; no reviewer is involved and no history is written. The requester can edit a draft with the same form and rules, submit it later (which is its first submission), or delete it. Deleting removes the draft and its line items completely.

## User Stories

1. As a requester, I want to save a claim as a draft, so that I can finish it later (FR-08).
2. As a requester, I want the expense date window checked only when I submit, not when I save a draft, so that the 90-day window counts from the real first submission (BR-07).
3. As a requester, I want my drafts in My Claims with a grey "Draft" badge, so that I can find them.
4. As a requester, I want drafts visible only to me, so that nobody sees unfinished work (Draft).
5. As a reviewer, I want drafts never to appear in my queue or claim views, so that I only see claims sent to me.
6. As a requester, I want to open a draft and edit its title, purpose, and line items, so that I can complete it (FR-08).
7. As a requester, I want the edit form to be the same as the New Claim form, so that I do not learn two forms.
8. As a requester, I want the total recalculated on the server every time I save, so that it is always right (FR-03, BR-08).
9. As a requester, I want the same line-item and total rules on every save, so that a draft cannot store invalid amounts (BR-05, BR-06, BR-12).
10. As a requester, I want field-level messages on the edit form, so that I can fix mistakes (NFR-08).
11. As a requester, I want to submit a draft when it is ready, so that it goes to my assigned reviewer (FR-08).
12. As a requester, I want the expense date window checked when I submit the draft, so that the 90-day rule counts from the real first submission (BR-07).
13. As a requester, I want my assigned reviewer recorded when the draft is first submitted, so that the reviewer is fixed from then on (BR-17).
14. As an auditor, I want the first history entry written when a draft is submitted, from `Draft` to `Submitted`, so that history starts at the first submission (FR-07).
15. As a requester, I want to delete a draft I no longer need, so that My Claims stays tidy (FR-08).
16. As a requester, I want a confirmation before deleting a draft, so that I do not delete by accident.
17. As an auditor, I want only drafts to be deletable, so that a claim with history is never removed (BR-10).
18. As a security reviewer, I want only the owner to edit or delete a draft, so that nobody changes another member's claim (BR-10).
19. As a security reviewer, I want editing blocked while a claim is `Submitted`, so that a claim cannot change while a reviewer looks at it (BR-08, D-10).
20. As a Finance Officer, I want to be unable to create drafts, so that no claim exists without a reviewer (BR-15).
21. As a requester, I want a 409 message when my draft was changed in another tab, so that I do not overwrite newer work.
22. As a developer, I want edits and deletes to require `If-Match`, so that stale writes fail with 409.
23. As a developer, I want the edit action shown only from `allowedActions`, so that the web never re-derives the rules.

## Implementation Decisions

- **Lifecycle.** Add `SAVE_DRAFT → DRAFT` (create), `DRAFT + SUBMIT → SUBMITTED`, and `DRAFT + DELETE → (removed)`. Drafts have no history; history starts at `SUBMIT`.
- **Policy.** Add `edit` (own, `DRAFT` or `RETURNED`), `delete` (own, `DRAFT`), `submit` (own, `DRAFT`, owner has an assigned reviewer). Drafts are visible only to their owner.
- **Claim module.** `Create(actor, input, submit=false)` saves a draft: BR-15 still applies; line-item rules (BR-05, BR-06, BR-12) and server total (BR-08) run; `assigned_reviewer_id`, `first_submitted_at`, and `submitted_at` stay NULL. `Edit(actor, id, version, input)` replaces title, purpose, and line items, recalculates the total, runs the same rules, and uses the version check. `Delete(actor, id, version)` deletes the claim and its line items. `Act(SUBMIT)` on a draft sets the assigned reviewer, `first_submitted_at`, `submitted_at`, runs BR-07, and writes the first history row (`DRAFT → SUBMITTED`).
- **Edit for `RETURNED`.** The edit path is built to accept `RETURNED` too, but that state appears only in F-06.
- **API.** `POST /claims?draft=true` (`claim:create`), `PUT /claims/{id}` with `If-Match`, `DELETE /claims/{id}` with `If-Match` (204), `POST /claims/{id}/submit` with `If-Match`. Errors as in SDS §4.2: 403 not owner, 409 `INVALID_TRANSITION` with BR-10 when not editable or deletable, 409 `CONFLICT`, 422, 428.
- **Web.** "Save as draft" next to "Submit" on the New Claim form. An Edit Claim route that reuses the claim form with initial values. Submit and Delete buttons on the detail page from `allowedActions`; Delete asks for confirmation. My Claims shows drafts.

## Testing Decisions

- Test through `claim.Service` against real PostgreSQL and through the HTTP router.
- **Module:** save draft writes no history and no reviewer; another requester and the reviewer cannot see the draft; Finance cannot save a draft (BR-15); edit re-runs BR-05/06/12 and recalculates the total; client total ignored; edit of a `SUBMITTED` claim → `ErrInvalidTransition{BR-10}`; delete of a `SUBMITTED` claim → `ErrInvalidTransition{BR-10}`; non-owner edit or delete → `ErrForbidden`; submit a draft records reviewer and `first_submitted_at` and writes one history row; BR-07 checked at draft submission with a fixed clock; stale version → `ErrConflict`.
- **HTTP:** status codes for each path, 428 without `If-Match`, route inventory picks up the new operations automatically.
- **Web:** save as draft, edit form with initial values, field messages, delete confirmation, buttons from `allowedActions`.
- Prior art: F-02 rules tests and F-04 `Act` tests.

## Out of Scope

- Return and resubmit of `Returned` claims (F-06).
- Autosave while typing.

## Further Notes

- F-06 depends on this feature because it reuses the editor and the edit path for `RETURNED` claims.

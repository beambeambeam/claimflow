# ClaimFlow — Expense Reimbursement

A department reimburses its members for work expenses they have already paid. Members submit claims, and an assigned reviewer decides on each one.

## Language

### Claims

**Claim**: A request to be paid back for expenses the requester has already paid. It holds one or more line items.

_Avoid_: Request, application, form

**Draft**: A claim the requester has saved but not yet submitted. Only its requester can see it, and no reviewer is involved.

_Avoid_: Unsent claim, saved claim

**Line Item**: One paid expense within a claim, backed by a receipt.

_Avoid_: Entry, row, expense line

**Requester**: The member who submits a claim and owns it. Any member with an assigned reviewer can be a requester, including a reviewer. A Finance Officer cannot be a requester.

_Avoid_: Claimant, applicant, user (when the owner is meant)

### Review

**Reviewer**: A member who can decide on claims assigned to them. A reviewer can also submit their own claims as a requester.

_Avoid_: Approver, manager

**Assigned Reviewer**: The one reviewer responsible for deciding a given requester's claims. It is never the requester themselves.

_Avoid_: Approver, supervisor

**Reporting Line**: The chain of assigned reviewers: staff → Section Head → Department Head → Finance Officer. Section Head and Department Head are positions held by reviewers, not separate roles.

_Avoid_: Hierarchy, org chart, chain of command

**Decision**: A reviewer's outcome on a submitted claim: approve, reject, or return.

**Return**: A decision that sends a claim back to its requester to be corrected and resubmitted as the same claim.

_Avoid_: Send back, revise

**Reject**: A final decision that ends a claim. To try again, the requester must submit a new, unrelated claim.

_Avoid_: Deny, decline

**Self-approval**: A reviewer making a decision on a claim they submitted. It is never allowed.

**Withdraw**: The requester taking back their own claim while it is submitted or returned. Withdrawing ends the claim. A claim waiting for a Finance Officer's final decision cannot be withdrawn.

_Avoid_: Cancel, retract, delete

### Approval stages

**High-value Claim**: A claim whose total is more than 10,000.00 THB. After the assigned reviewer approves it, it also needs a final decision from a Finance Officer. When the assigned reviewer is already a Finance Officer, that approval is final.

_Avoid_: Large claim, big claim

**Finance Officer**: The member who gives the final approve-or-reject decision on high-value claims, and who is the assigned reviewer for the Department Head. A Finance Officer cannot return a claim when making a final decision.

_Avoid_: Accountant, finance approver

**Final Decision**: A Finance Officer's approve or reject on a high-value claim that the assigned reviewer has already approved.

_Avoid_: Second approval, finance approval

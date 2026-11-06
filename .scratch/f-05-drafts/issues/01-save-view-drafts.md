# 01: Save and view drafts

**What to build:** On the New Claim form a requester can choose "Save as draft" instead of "Submit". The draft appears in My Claims with a grey "Draft" badge and opens on its detail page. Only its requester can see it; no reviewer is recorded and no history is written. A Finance Officer still cannot create drafts.

**Blocked by:** F-02/02 Claim rules with field-level errors; F-03/01 My Claims list; F-03/02 Claim Detail and the claim policy. Note: the #6/D2 contract for F-05 merges before the implementation PR.

**Status:** ready-for-agent

- [ ] `POST /claims?draft=true` saves a `DRAFT` with line-item rules (BR-05, BR-06, BR-12) and server total (BR-08); BR-07 is not checked yet.
- [ ] `assigned_reviewer_id`, `first_submitted_at`, and `submitted_at` stay NULL; no history row.
- [ ] Finance draft → 403 (BR-15).
- [ ] Policy: drafts are visible only to their owner; the reviewer and other requesters get 403.
- [ ] Web: "Save as draft" button; Draft badge in My Claims and detail.

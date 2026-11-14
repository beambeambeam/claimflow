-- name: NextClaimSequence :one
-- NextClaimSequence takes the next number of claim_no_seq. The sequence never resets and never gives a number twice,
-- so a rolled-back claim leaves a gap in the numbers.
SELECT nextval('claim_no_seq')::bigint AS seq;

-- name: GetClaimMember :one
-- GetClaimMember returns a member's reviewer, which BR-15 requires and BR-17 copies onto a claim at first submission.
SELECT id, reviewer_id
FROM "user"
WHERE id = sqlc.arg(id);

-- name: InsertClaim :exec
INSERT INTO claims (id, claim_no, requester_id, assigned_reviewer_id, title, purpose, status, total_satang,
                    first_submitted_at, submitted_at, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(claim_no), sqlc.arg(requester_id), sqlc.narg(assigned_reviewer_id), sqlc.arg(title),
        sqlc.arg(purpose), sqlc.arg(status), sqlc.arg(total_satang), sqlc.narg(first_submitted_at),
        sqlc.narg(submitted_at), sqlc.arg(now), sqlc.arg(now));

-- name: InsertLineItem :exec
INSERT INTO line_items (id, claim_id, line_no, category, description, expense_date, amount_satang, receipt_ref)
VALUES (sqlc.arg(id), sqlc.arg(claim_id), sqlc.arg(line_no), sqlc.arg(category), sqlc.arg(description),
        sqlc.arg(expense_date), sqlc.arg(amount_satang), sqlc.arg(receipt_ref));

-- name: InsertClaimHistory :exec
INSERT INTO claim_history (claim_id, actor_id, action, from_status, to_status, comment, created_at)
VALUES (sqlc.arg(claim_id), sqlc.arg(actor_id), sqlc.arg(action), sqlc.narg(from_status), sqlc.arg(to_status),
        sqlc.narg(comment), sqlc.arg(now));

-- name: GetClaim :one
-- GetClaim returns a claim with its requester and, when it has one, its assigned reviewer.
SELECT c.id, c.claim_no, c.status, c.title, c.purpose, c.total_satang, c.return_count, c.version,
       c.first_submitted_at, c.submitted_at, c.decided_at, c.created_at, c.updated_at,
       r.id AS requester_id, r.name AS requester_name, r.position AS requester_position,
       a.id AS reviewer_id, a.name AS reviewer_name, a.position AS reviewer_position
FROM claims c
JOIN "user" r ON r.id = c.requester_id
LEFT JOIN "user" a ON a.id = c.assigned_reviewer_id
WHERE c.id = sqlc.arg(id);

-- name: ListClaimLineItems :many
SELECT line_no, category, description, expense_date, amount_satang, receipt_ref
FROM line_items
WHERE claim_id = sqlc.arg(claim_id)
ORDER BY line_no;

-- name: ListClaimHistory :many
SELECT h.id, h.action, h.from_status, h.to_status, h.comment, h.created_at,
       u.id AS actor_id, u.name AS actor_name, u.position AS actor_position
FROM claim_history h
JOIN "user" u ON u.id = h.actor_id
WHERE h.claim_id = sqlc.arg(claim_id)
ORDER BY h.id;

-- name: ListRequesterClaims :many
-- ListRequesterClaims returns one page of a requester's claims in every status, newest first (F-03 §1.4).
SELECT c.id, c.claim_no, c.status, c.title, c.total_satang, c.submitted_at, c.created_at,
       r.id AS requester_id, r.name AS requester_name, r.position AS requester_position
FROM claims c
JOIN "user" r ON r.id = c.requester_id
WHERE c.requester_id = sqlc.arg(requester_id)
ORDER BY c.created_at DESC, c.id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountRequesterClaims :one
SELECT count(*)::int AS total
FROM claims
WHERE requester_id = sqlc.arg(requester_id);

-- name: ListReviewClaims :many
-- ListReviewClaims returns one page of the SUBMITTED claims assigned to a reviewer, oldest submission first (F-04 §3).
SELECT c.id, c.claim_no, c.status, c.title, c.total_satang, c.submitted_at, c.created_at,
       r.id AS requester_id, r.name AS requester_name, r.position AS requester_position
FROM claims c
JOIN "user" r ON r.id = c.requester_id
WHERE c.assigned_reviewer_id = sqlc.arg(reviewer_id) AND c.status = 'SUBMITTED'
ORDER BY c.submitted_at, c.id
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountReviewClaims :one
SELECT count(*)::int AS total
FROM claims
WHERE assigned_reviewer_id = sqlc.arg(reviewer_id) AND status = 'SUBMITTED';

-- name: DecideClaim :execrows
-- DecideClaim moves a claim to its decided status and adds one to the version, but only while the version is still the
-- one the caller read (BR-13). It returns the rows changed: 0 means the claim changed since.
UPDATE claims
SET status = sqlc.arg(status), version = version + 1, decided_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version);

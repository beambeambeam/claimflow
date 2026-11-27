-- name: CountClaimsByStatus :many
-- CountClaimsByStatus counts the claims in a summary scope by current status (F-10 §1.3, §1.4). A draft is never
-- counted. all_claims is the Finance Officer's scope, every claim that is not a draft. Otherwise the scope is the
-- claims assigned to actor_id.
SELECT status, count(*)::int AS total
FROM claims
WHERE status <> 'DRAFT'
  AND (sqlc.arg(all_claims)::bool OR assigned_reviewer_id = sqlc.arg(actor_id))
GROUP BY status;

-- name: ApprovedTotalsByMonthCategory :many
-- ApprovedTotalsByMonthCategory sums the line items of the APPROVED claims in a summary scope by the Asia/Bangkok month
-- of the decision and by category, ordered by month and then by category in the order of the expense_category enum
-- (F-10 §1.4). from_at and to_at bound decided_at as a half-open range; a NULL bound is open.
SELECT to_char(c.decided_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM')::text AS month,
       li.category,
       sum(li.amount_satang)::bigint AS total
FROM claims c
JOIN line_items li ON li.claim_id = c.id
WHERE c.status = 'APPROVED'
  AND (sqlc.arg(all_claims)::bool OR c.assigned_reviewer_id = sqlc.arg(actor_id))
  AND (sqlc.narg(from_at)::timestamptz IS NULL OR c.decided_at >= sqlc.narg(from_at))
  AND (sqlc.narg(to_at)::timestamptz IS NULL OR c.decided_at < sqlc.narg(to_at))
GROUP BY 1, li.category
ORDER BY 1, li.category;

-- name: AverageDecisionHours :one
-- AverageDecisionHours returns how many APPROVED and REJECTED claims in a summary scope were decided in the range, and
-- the mean hours from first submission to decision, rounded to one decimal place with round(numeric), which rounds half
-- away from zero (F-10 §1.4). The mean is 0 when no claim was decided; the caller reads that as null by the count.
-- from_at and to_at bound decided_at as a half-open range; a NULL bound is open.
SELECT count(*)::int AS decided,
       COALESCE(round(avg(extract(epoch FROM c.decided_at - c.first_submitted_at)) / 3600, 1), 0)::float8 AS hours
FROM claims c
WHERE c.status IN ('APPROVED', 'REJECTED')
  AND (sqlc.arg(all_claims)::bool OR c.assigned_reviewer_id = sqlc.arg(actor_id))
  AND (sqlc.narg(from_at)::timestamptz IS NULL OR c.decided_at >= sqlc.narg(from_at))
  AND (sqlc.narg(to_at)::timestamptz IS NULL OR c.decided_at < sqlc.narg(to_at));

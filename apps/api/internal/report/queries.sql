-- name: CountClaimsByStatus :many
-- CountClaimsByStatus counts the claims in a summary scope by current status (F-10 §1.3, §1.4). A draft is never
-- counted. all_claims is the Finance Officer's scope, every claim that is not a draft. Otherwise the scope is the
-- claims assigned to actor_id.
SELECT status, count(*)::int AS total
FROM claims
WHERE status <> 'DRAFT'
  AND (sqlc.arg(all_claims)::bool OR assigned_reviewer_id = sqlc.arg(actor_id))
GROUP BY status;

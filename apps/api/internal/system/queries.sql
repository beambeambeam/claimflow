-- name: CheckReady :one
-- CheckReady runs a trivial query, so /ready fails when PostgreSQL cannot answer.
SELECT 1::integer AS ok;

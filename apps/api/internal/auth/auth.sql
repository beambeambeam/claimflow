-- name: GetCredentialForLogin :one
-- GetCredentialForLogin returns the user and password hash of the credential account for a lowercase username.
-- The migration requires a password on every credential account, so COALESCE only guards the scan type.
SELECT u.id AS user_id, u.name, u.username, u.role, COALESCE(a.password, '')::text AS password_hash
FROM "user" u
JOIN account a ON a.user_id = u.id AND a.provider_id = 'credential'
WHERE u.username = sqlc.arg(username);

-- name: CreateSession :exec
INSERT INTO session (id, user_id, token, expires_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(token), sqlc.arg(expires_at));

-- name: GetSessionUser :one
-- GetSessionUser returns the user of an unexpired session whose user is not banned.
-- A ban with no end, or with an end in the future, is active.
SELECT u.id, u.name, u.username, u.role
FROM session s
JOIN "user" u ON u.id = s.user_id
WHERE s.token = sqlc.arg(token)
  AND s.expires_at > now()
  AND NOT (u.banned AND (u.ban_expires IS NULL OR u.ban_expires > now()));

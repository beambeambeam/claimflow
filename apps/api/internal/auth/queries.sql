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

-- name: UpsertSeedUser :one
-- UpsertSeedUser inserts a demo user, or updates the user with the same username. It returns the user id,
-- which the row keeps across runs, so a reviewer_id taken from it stays valid.
INSERT INTO "user" (id, name, email, username, display_username, role, position, reviewer_id)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(email), sqlc.arg(username), sqlc.arg(username), sqlc.arg(role), sqlc.arg(position), sqlc.narg(reviewer_id))
ON CONFLICT (username) DO UPDATE SET
  name = EXCLUDED.name,
  email = EXCLUDED.email,
  display_username = EXCLUDED.display_username,
  role = EXCLUDED.role,
  position = EXCLUDED.position,
  reviewer_id = EXCLUDED.reviewer_id,
  updated_at = now()
RETURNING id;

-- name: UpsertSeedAccount :exec
-- UpsertSeedAccount stores the bcrypt hash of a demo user's credential account. The account id is the user id,
-- as Better Auth keys credential accounts.
INSERT INTO account (id, user_id, provider_id, account_id, password)
VALUES (sqlc.arg(id), sqlc.arg(user_id), 'credential', sqlc.arg(account_id), sqlc.arg(password))
ON CONFLICT (provider_id, account_id) DO UPDATE SET
  user_id = EXCLUDED.user_id,
  password = EXCLUDED.password,
  updated_at = now();

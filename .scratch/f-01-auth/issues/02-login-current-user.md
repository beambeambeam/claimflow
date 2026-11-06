# 02: Log in and see who I am

**What to build:** A seeded member opens the Login screen, enters username and password, and gets a server-side session in an HttpOnly cookie. Every protected page under `_auth` shows the header with display name and role, and an anonymous visitor is redirected to `/login?redirect=<path>`. Requesters land on `/claims`; reviewers and Finance Officers land on `/reviews` (placeholders until F-03 and F-04).

**Blocked by:** F-01/01 Walking skeleton

**Status:** ready-for-agent

- [ ] Migrations add `user`, `account`, and `session` tables as in SDS §2.2; UUIDv7 keys generated in Go.
- [ ] An idempotent seed loads the 7 reporting-line users with the password from `SEED_USER_PASSWORD`; no password is committed.
- [ ] `POST /auth/login` lowercases the username, checks the bcrypt hash (cost ≥ 10), and sets `__Host-sid` (or `sid` when `COOKIE_SECURE=false`) with `HttpOnly`, `SameSite=Lax`, `Path=/`, 8 h `Max-Age`.
- [ ] Only the SHA-256 of the token is stored; no password or token appears in logs or responses.
- [ ] Wrong password and unknown user both return 401 "Invalid username or password."; unknown users are compared against a dummy hash.
- [ ] `GET /auth/me` returns `{ id, username, name, role, permissions }` from the session only.
- [ ] Cross-origin state-changing requests return 403.
- [ ] Web: Login form with loading state and error message; `_auth` guard redirects anonymous users and returns them to the requested page after login.

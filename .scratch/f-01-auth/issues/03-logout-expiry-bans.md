# 03: Log out, session expiry, and bans

**What to build:** A member clicks Log out in the header and their session ends at once; the old cookie no longer works. A session also ends 8 hours after login. A banned member cannot log in and loses every session. Whenever the web gets a 401, it clears cached data and sends the member to the login page.

**Blocked by:** F-01/02 Log in and see who I am

**Status:** ready-for-agent

- [ ] `POST /auth/logout` deletes the session row and clears the cookie; reusing the old token → 401 (NFR-03).
- [ ] A session older than 8 hours → 401, tested with an injected fixed clock; the lifetime is a constant.
- [ ] Expired session rows are deleted at login.
- [ ] A banned user (`banned = true`, `ban_expires` NULL or in the future) gets the same generic 401 at login; a ban with `ban_expires` in the past counts as no ban.
- [ ] Setting a ban deletes the user's sessions; a session created before the ban → 401.
- [ ] Web: Log out button in the header; any 401 clears the query cache and redirects to `/login?redirect=<path>`.

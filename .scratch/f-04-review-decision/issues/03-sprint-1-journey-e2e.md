# 03: Sprint 1 journey: demo seed, Playwright, and restart check

**What to build:** The seed adds sample claims in `Submitted`, `Approved`, and `Rejected` with matching history, so every Sprint 1 screen has data. A Playwright test walks the main journey: requester logs in and submits a claim → assigned reviewer approves → requester sees `Approved` with history. A restart check proves claims and history survive an API restart.

**Blocked by:** F-04/02 Reject, invalid transitions, and concurrent decisions; F-03/01 My Claims list; F-02/02 Claim rules with field-level errors

**Status:** ready-for-agent

- [ ] Seed claims in every Sprint 1 status, idempotent by claim number, each with history.
- [ ] Playwright is set up in the web app and runs against the local API and web servers with a seeded database.
- [ ] E2E: submit → approve → requester sees `Approved` with the decision in history.
- [ ] E2E or recorded manual check: a business-rule rejection shown under its field, and a forbidden action blocked.
- [ ] Restart check (NFR-05): restart the API and read the same claims and history; result recorded for #4.
- [ ] README documents database setup, migrate, seed, and how to get demo accounts without committing passwords.

# 02: Claim rules with field-level errors

**What to build:** When a requester submits a claim that breaks a rule, the server rejects it with 422 and one error per field, and the form shows each message under the field that broke the rule (for example "Expense date cannot be in the future."). The form also checks simple limits before sending.

**Blocked by:** F-02/01 Submit a valid claim end to end

**Status:** ready-for-agent

- [ ] Limits are named constants: 1–20 line items, item 0.01–20,000.00, at most 2 decimals (BR-05); total ≤ 50,000.00 (BR-06); Meals ≤ 1,500.00 (BR-12).
- [ ] Expense date ≤ today and ≥ first-submission date − 90 days, in Asia/Bangkok, using an injected clock (BR-07).
- [ ] Table-driven boundary tests through `claim.Service`: 0 and 21 items; `0.00`, `0.01`, `20000.00`, `20000.01`, `1.005`; total 50,000.00 vs 50,000.01; Meals 1,500.00 vs 1,500.01; tomorrow, today, 90 and 91 days ago; just after Bangkok midnight; invalid category.
- [ ] A client-sent total is ignored (BR-08).
- [ ] 422 problem body lists `field` (for example `items[1].expenseDate`), `rule`, and `message`.
- [ ] Web: generated Zod checks before sending; server errors mapped to fields; input kept after an error (NFR-08).

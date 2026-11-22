# CPE334 Lab 7

> Source: `Lab_7_Sprint_1_Maintenance.pdf`. Full text converted to Markdown. See [conversion and privacy notes](README.md).

Software maintenance metrics and release readiness

Due 29 November 2026 at 23:59 Thailand time | 100 lab points | 5% of course grade | Semester 1/2026

## 1 Purpose and assessment

Lab 7 asks: Can you change it without breaking it? Obtain approval for one bounded change to a behavior or business rule already implemented in Sprint 1. Unlike Lab 6 corrective maintenance, which fixes behavior that violates an existing specification, Lab 7 changes previously correct behavior because a requirement has changed. Analyze its impact, revise the engineering contract before implementation, demonstrate test-first red/green evidence, and release the change while preserving unaffected behavior. Use objective metrics and a short readiness review to guide the final project release.

Use the same two students, application and repository as the group project. Both students must participate and review each other’s work. This lab is worth 100 points and 5% of the course grade. Labs 5–7 total 300 points and 15%; project deliverables retain their separate scores.

| Part | Points | Required outcome |
| --- | --- | --- |
| A Change impact and updated contract | 25 | Approved old-to-new behavior and reviewed impact/contract revisions |
| B Implementation and regression safe delivery | 50 | Changed behavior passes and existing regression remains healthy through deployment |
| C Objective engineering metrics | 15 | Five verifiable metrics and a short evidence-based action |
| D Release readiness and retrospective | 10 | Honest release checklist and three brief retrospective answers |
| Total | 100 |  |

## 2 Starting point and scope

Use the accepted Sprint 1 release from Lab 6 as the reference for the behavior being maintained, and continue using the working GCP/CI/CD process from Labs 5–6. The selected rule or behavior must already have been implemented in Sprint 1. You may apply the approved change in your evolving project repository, including code from later sprints; a separate Sprint 1 maintenance branch is not required. Identify the accepted Sprint 1 commit/version, the actual pre-change commit/deployed version, and any intervening development. Obtain instructor/TA approval before implementation. Simply completing an unchanged Sprint 2/3 backlog feature does not qualify.

For example, change an appointment cancellation rule already implemented in Sprint 1 from cancellation before the start time to a specified advance cutoff, with exact boundary times and permitted roles. For Personal Finance, change an existing Sprint 1 payment validation or rounding rule, specifying balance effects and owner-only access. These are examples: choose a genuine change to behavior already delivered in your own Sprint 1, not a large new subsystem.

Lab 7 and Deliverable 2 are both due on 29 November. The same approved change may be included in the project release candidate, but Lab 7 assesses the maintenance process and evidence. Deliverable 2 assesses overall sprint discipline and increment integration. Give each submission its own self-contained assessed material.

## 3 Required engineering work

### 3.1 Part A Change impact and updated contract

25 points. Explain what changes, what it affects and how the contract is revised before code implementation.

#### Approve a precise change request

Create an Issue/change request identifying the original Sprint 1 requirement and implemented rule, the proposed new rule, reason, scope and acceptance criteria. State relevant boundaries, invalid cases and access expectations. Record instructor/TA approval, approver and date. Identify the accepted Sprint 1 baseline where the old behavior existed and the actual pre-change implementation to be modified.

#### Trace the impact

Use a compact table connecting requirement ID, feature, affected SRS/Feature SDS/System-Level SDS sections, acceptance criteria, STS test IDs and code modules. Consider callers/dependencies, UI/API validation, roles/ownership, persistence, historical data and migrations where applicable. Give a reason for not-applicable items. Define what existing behavior must remain unchanged and how regression tests will protect it.

#### Revise and review the engineering contract

Update the relevant specification sections and Software Test Specification (STS), including inputs/preconditions, expected results and acceptance-criterion coverage. Version the changes and have the teammate review the revised contract before implementing the code change. Preserve dates/commits showing this sequence. Embed the old/new assessed excerpts and review in the PDF. A retrospective document edited only after code is finished does not show planning before implementation.

### 3.2 Part B Implementation and regression safe delivery

50 points. Use meaningful tests to drive the change and retain the existing behavior through a reviewed release.

#### Show the new behavior failing on old code

Add or revise an automated test that asserts the approved new behavior, including a relevant boundary. Run it against the actual pre-change implementation of the existing Sprint 1 behavior and preserve the failure output and tested commit. The failure must be due to the missing new behavior; a compile error or deliberately broken fixture is not valid red evidence. Tests and contract changes should precede the implementation commit.

#### Implement and show green

Make the bounded implementation change. Show the same changed-behavior test passing, with relevant setup/assertions and the actual code diff. Address rule, authorization and data effects identified by the impact analysis. Include migrations only if the change requires them. Keep calculation precision and transaction consistency correct where relevant.

#### Run relevant existing regression tests

Run existing tests that cover dependent behavior, domain rules, persistence and access affected by the change. State the selected suite and why it is relevant; identify commands, versions and passing/failed/skipped counts. Do not weaken, delete or disable old tests to hide unrelated failures. When a changed requirement legitimately changes an old expected result, explain and version that revision while retaining tests for behavior that should remain unchanged.

#### Review integrate and deploy

Connect the approved Issue to the contract/test/code commits, substantive teammate PR review, passing CI and sprint staging integration, then a reviewed main release and GCP deployment. Use the normal project Git workflow; a separate Sprint 1-only branch or release is not required. Identify the maintenance change separately from any unrelated Sprint 2/3 work. Record the accepted release and deployed revision. Perform selected post-deployment checks for the changed behavior and an important preserved Sprint 1 domain/access path. Use the Lab 5 pipeline; show actual outcomes rather than only a green status icon.

For Personal Finance, preserved behavior includes user isolation and Administrator denial of financial data. A rule change must not bypass backend ownership checks or produce inconsistent balances. Use synthetic accounts and transactions in the evidence.

### 3.3 Part C Objective engineering metrics

15 points. Report five dated engineering metrics and use them to select one concrete action. Counts earn credit for accuracy and usefulness, not size.

#### Define five metrics and their sources

For each metric, give its definition, repository/product scope, observation window or snapshot time, value and source/evidence ID. Choose measures your actual records can support. The table below is a recommended set; equivalent meaningful measures are acceptable. A zero is valid when the source supports it. Do not invent activity to increase counts.

| Metric example | Definition and source example |
| --- | --- |
| Completed Issues | Issues completed in a stated date window; list IDs and completion dates, distinguishing duplicates/cancelled work. |
| Merged PRs | PRs merged in the stated window; list IDs, merge dates and target branches. |
| Automated test results | Passed/failed/skipped counts for one named suite on one identified commit/run; embed the summary and command. |
| UAT defect status | Defects opened and resolved in a stated window or status at a stated time; give Issue IDs/statuses and define resolved. |
| CI run outcomes | Successful/failed/cancelled runs in the stated window; identify workflow/branch scope and list run IDs/outcomes. |

Embed the underlying compact lists, source tables or readable result summaries so a TA can reconcile at least two values without opening external links. Derived percentages must state numerator and denominator. Do not mix unrelated repositories or an old test run into the current release snapshot.

#### Interpret the evidence in at most 100 words

Identify one concrete engineering problem or risk shown by the metrics and choose a specific action with an owner or next milestone. If the values are healthy, describe an evidence-based risk and how you will preserve that result; do not invent a problem. Generic advice such as improve quality is insufficient.

### 3.4 Part D Release readiness and retrospective

10 points. State whether the maintained Sprint 1 functionality in the current deployed release is ready, what remains, and what the team will improve before Final Delivery. This is not a declaration that the entire three-sprint product is complete.

#### Complete a release readiness checklist

Record the submitted/deployed version, defects by severity/status, automated and selected user-test results including failed/skipped checks, technical debt, configuration/secrets, documentation/run instructions and deployment/smoke status. State READY or NOT READY for the maintained Sprint 1 scope, including the approved change and unaffected previously accepted behavior. Give reasons and an owner/action for outstanding items. Later-sprint features may coexist in the deployed project but must not be mistaken for completed Sprint 1 maintenance evidence. A checklist may honestly report incomplete work; it must not conceal failed checks.

Technical debt is engineering work deferred now that creates future cost or risk, such as duplicated business-rule code or missing regression coverage. Keep a short table of item, impact, priority and disposition/action. Distinguish it from a known incorrect behavior, which belongs in the defect list. State none known with the review basis if no debt is identified.

#### Answer three brief retrospective questions

Answer what helped, what hurt, and what will change for the final release. Use one or two sentences for each. Tie the improvement to observed work, with a concrete next action. This is a short engineering reflection, not an essay.

## 4 What to submit

Submit one self-contained PDF per team through LEB2 by 29 November 2026 at 23:59. Identify both students and IDs, project theme/context, accepted Sprint 1 baseline tag/commit, pre-change commit, submitted main release tag/commit, repository URL and deployed URL/version. The PDF is the normal grading surface. Include a contents table with Parts A–D, page ranges and evidence IDs.

### 4.1 Include the assessed material directly in the PDF

| Part | Include directly in the PDF |
| --- | --- |
| A | Approved Sprint 1 behavior change with old/new requirements and accepted Sprint 1 baseline; impact table; revised relevant SRS/SDS/AC/STS sections and teammate review before implementation. |
| B | Test setup/assertions and actual red/green output on pre-change behavior; code diff; existing Sprint 1 regression results; Issue/PR review, CI, integration, accepted main release and deployed changed/preserved-behavior checks. |
| C | Five defined, dated metrics with observation windows, values and embedded source evidence; one action-oriented interpretation of at most 100 words. |
| D | Readiness checklist for the maintained Sprint 1 scope, defect/debt status and deployed version; three brief retrospective answers; contributions, AI-use declaration and known limitations. |

Appendices belong in the same PDF. Embed the relevant complete assessed cases, specification sections, report records, test excerpts and selected execution output; use internal page references. Include readable Issue/PR review text and results rather than a collection of external links. Full repository history remains available for targeted corroboration. Do not paste the whole application source or unchanged SRS.

### 4.2 Make evidence quick to inspect

- Give screenshots/log excerpts an evidence ID, action, expected/actual result, time and relevant tested commit or deployed revision. Keep text readable and preserve enough context to support the claim.

- For automated tests, include the actual command, file/test IDs, relevant setup/assertion, tested version and passing/failing/skipped counts. Label manual tests as manual. A green icon alone is insufficient.

- Provide a short version table connecting the accepted Sprint 1 baseline, actual pre-change implementation, test changes, accepted main release and deployed revision. Explain legitimate merge-commit differences and identify any unrelated later-sprint changes.

- Include a short contribution table, AI-use declaration and known limitations. Identify pair work; equal commit counts, full prompt transcripts and long reflections are not required.

There is no fixed PDF page limit. Include material once and refer to its pages. A separate video is not required. Supply safe test-account credentials through the private course handoff, and keep the assessed deployment and repository available through grading. Use synthetic data; do not include real passwords, tokens or production secrets in the PDF or Git.

## 5 Submission checklist

- The approved request changes behavior already implemented in Sprint 1, states the old/new rule precisely, and identifies the Sprint 1 and pre-change versions.

- The impact analysis, revised contract and review precede implementation in recorded history.

- A meaningful test fails on old code, then passes with the fix; relevant existing regression remains covered.

- Review, CI, release version and deployment checks form one traceable change sequence.

- Five metrics can be reconciled with sources embedded in the PDF; the interpretation is no more than 100 words.

- Readiness is stated for the maintained Sprint 1 scope, and all three retrospective answers and assessed evidence are inside one PDF.

Existing course deadline and academic-integrity policies apply. Disclose failed/skipped tests and unfinished work. Do not claim that a large test count, zero reported bugs or many PRs alone proves release readiness.

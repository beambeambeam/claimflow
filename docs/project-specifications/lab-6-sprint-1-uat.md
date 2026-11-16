# CPE334 Lab 6

> Source: `Lab_6_Sprint_1_UAT.pdf`. Full text converted to Markdown. See [conversion and privacy notes](README.md).

Independent user acceptance testing and defect management

Due 22 November 2026 | 100 lab points | 5% of course grade | Semester 1/2026

## 1 Purpose and assessment

Lab 6 asks: Can somebody else verify and accept it? Give another team an executable user acceptance test (UAT) plan for your delivered Sprint 1 product. Independently test that team’s Sprint 1 release, review all findings on your own product, correct release-blocking defects, and obtain a documented independent UAT acceptance decision. UAT verifies delivered behavior against the agreed Sprint 1 requirements and acceptance criteria; it does not accept unfinished Sprint 2 or 3 features.

Use the same two students, application and repository as the group project. Both students must participate and review each other’s work.

| Part | Points | Required outcome |
| --- | --- | --- |
| A UAT preparation and handoff | 20 | Another team can execute cases covering the delivered Sprint 1 scope |
| B Independent execution | 25 | Your team independently tests the other team’s Sprint 1 release and records results |
| C Defect reporting and triage | 20 | All findings classified and assigned a documented disposition |
| D Resolution and independent retest | 35 | Release-blocking defects corrected/retested; Sprint 1 acceptance recorded |
| Total | 100 |  |

## 2 Starting point and partner arrangement

Use the working GCP release and CI/CD process established in Lab 5. The assessed UAT scope is the delivered Sprint 1 functionality, including its agreed requirements and acceptance criteria. Identify its baseline release and tested version. A newer deployment is acceptable only when the tested Sprint 1 scope and contract are clearly identified; Sprint 2 or 3 features are not part of this acceptance.

Pair with another project team (two students per team). The two teams will exchange roles:

- Product Team: The team whose application is being tested.

- UAT Team: The other team, which independently tests the Product Team's application.

Each team must serve in both roles: as the Product Team for its own application and as the UAT Team for the other team's application. Record both teams and the names of all participating students. Each team submits its own PDF covering both responsibilities.

| Your role | Your responsibility |
| --- | --- |
| Product Team | Prepare Sprint 1 UAT cases and safe test accounts for your application. Provide the deployed release and testing instructions to the UAT Team. Review and classify every finding, correct all release-blocking defects, run regression tests, redeploy, and obtain an independent UAT acceptance decision. Document any accepted minor limitations. |
| UAT Team | Execute the Product Team’s Sprint 1 UAT cases on its identified deployed release. Record actual results and reproducible findings, independently re-test corrected releases, and record an evidence-based acceptance decision for the tested Sprint 1 scope. |

For example, suppose the UAT Team discovers that TokTickIT allows a Requester to close another user's ticket.

1. Defect Report: The UAT Team reports the unauthorized operation.

2. Triage: The Product Team confirms that this violates the specification and accepts it as a defect.

3. Regression Test to Reproduce Error (RED): Write an automated test asserting that a Requester cannot close another user's ticket. Run it against the existing code and verify that it fails.

4. Fix (GREEN): Correct the authorization logic and run the test again. It should now pass.

5. Regression Testing: Run the broader test suite to verify that the fix has not broken existing functionality.

6. Review and Deployment: Submit a PR, obtain approval, pass CI, and deploy the corrected version.

7. Independent UAT Re-test: The UAT Team confirms that the corrected behavior passes in the deployed version and records its acceptance decision for Sprint 1.

### Coordination and version control

Agree on initial UAT, defect-correction, and re-test sessions early enough to finish before the deadline. Record the exact Sprint 1 application version being tested. The Product Team must inform the UAT Team before deploying any changes that could affect testing. Acceptance applies only to the specifically recorded release and Sprint 1 scope.

If the other team is unavailable or access to its application fails, notify the instructor/TA promptly and retain evidence of the notification. The instructor/TA will help arrange an alternative team or tester. Students must not substitute testing by their own team for the required independent UAT.

## 3 Required engineering work

### 3.1 Part A UAT preparation and handoff

20 points. Prepare executable cases covering the agreed Sprint 1 acceptance scope and make the handoff usable by another team.

#### Prepare at least four executable cases

Give each case a stable ID and Sprint 1 requirement/acceptance-criterion reference, purpose, role, preconditions, synthetic inputs, numbered user actions and specific expected results. Include at least four cases: one normal domain journey, one business-rule or boundary case, one forbidden role/ownership case, and one state/history/persistence case. Add cases where necessary to cover the important implemented Sprint 1 acceptance criteria; four is a minimum, not a limit. A login-only checklist is insufficient.

#### Provide a safe handoff

State the Sprint 1 release tag/commit and deployed revision, URL, accepted-for-test scope, required roles/accounts, data setup and reset/cleanup instructions. Make cases independent or declare their sequence. Give credentials privately; identify account labels and roles in the PDF without secret values. The UAT Team should not need verbal explanations to determine expected behavior.

For Personal Finance, use synthetic balances and transactions. The access case must cover another user’s records and Administrator denial of financial data. The normal and state cases should demonstrate account/transaction behavior in the implemented scope, with expected balances or money movement calculated explicitly.

### 3.2 Part B Independent execution

25 points. Execute the other Product Team’s prepared Sprint 1 cases on its recorded deployed release. These results belong to your team as independent UAT testers.

#### Record actual execution

Run all agreed Sprint 1 UAT cases supplied by the Product Team, including at least the four required categories. Record case ID, tester name/team, date/time, tested version, inputs, actual result and PASS/FAIL. If blocked, record BLOCKED and the reason; it is not a pass. Both students must participate. Do not copy expected results into the actual-result field.

#### Capture selected evidence

Include readable observations or screenshots supporting each outcome, with a normal journey and a negative/access case clearly demonstrated. For rejected operations, show the rejection and check that protected state was not changed or disclosed. Manual user-facing UAT is acceptable. Developer unit/API tests can support later corrections, but do not replace independent user testing.

### 3.3 Part C Defect reporting and triage

20 points. Document every actual finding and reach a reasoned, traceable disposition on every finding received about your own Sprint 1 product.

#### Report reproducible defects

For each actual failure, record an Issue ID, product owner/team, reporter, tested version, linked case/requirement, preconditions/data, reproduction steps, expected result, actual result, severity and supporting evidence. Explain severity by user impact. Distinguish a defect against specified behavior from a new feature request. Notify the partner through the agreed handoff and record the outcome.

#### Triage received findings

Keep a table of all received findings, including none if appropriate. For each, classify it as an accepted defect, duplicate, not-defect or enhancement, with a reason tied to the Sprint 1 contract and evidence. Link duplicates to the original Issue. For accepted defects, record severity, owner, fix status and disposition. Critical and major defects block Sprint 1 acceptance and must be corrected and independently re-tested. Minor defects must be fixed or explicitly accepted by the UAT Team as documented known limitations. Enhancements outside the agreed Sprint 1 scope need not be implemented for acceptance. Reproduce and clarify reports before dismissal; never relabel a failing requirement as an enhancement.

### 3.4 Part D Resolution and independent retest

35 points. Complete resolution of all received Sprint 1 UAT findings, correct all release-blocking accepted defects through the engineering and delivery process, and obtain a documented independent UAT acceptance decision for the tested release. All critical and major defects must be fixed and independently re-tested; minor defects must be fixed or explicitly accepted as known limitations. Report Not Accepted if these conditions are not met.

#### Demonstrate regression red and green

For each accepted defect corrected in this lab, first add a meaningful automated test that reproduces the reported behavior and fails against the defective implementation (RED). Correct the code and show the same test passing (GREEN), together with relevant existing regression tests. Preserve test names/assertions, commands, tested commits and actual output. A syntax error, broken fixture or test that always passes is insufficient. If the original release has no legitimate defect, follow Section 3.5 for a controlled exercise; do not invent one.

#### Review and redeploy the correction

For each corrected release-blocking defect, connect the Issue to a fix branch from sprint staging, substantive teammate PR review, passing CI and integration, then a reviewed release to main and deployment. Identify the corrected deployed revision and run post-deployment smoke checks. Where several fixes are needed, they may be combined into a clearly reviewed, tested release; maintain traceability from each Issue to its correction.

#### Obtain independent retest

Give the UAT Team the corrected release version and the original failed cases/reproduction steps. The UAT Team independently re-tests each corrected release-blocking defect, recording tester identity, version, actual result and evidence. Re-run affected UAT cases, and record one final Sprint 1 decision: Accepted, Accepted with explicitly documented minor limitations, or Not Accepted, with reasons. Only the UAT Team may confirm independent acceptance. Never claim acceptance while critical/major defects remain unresolved or required tests are BLOCKED.

### 3.5 If there is no legitimate defect

A sound Sprint 1 product is not penalized for having no natural defect. Complete the independent UAT cases and obtain the UAT Team’s documented acceptance decision. To demonstrate the defect-correction learning outcome, ask the instructor/TA for a controlled defect exercise on an isolated version/branch of your own application. Apply the same report, triage, reproducing-test (RED), fix (GREEN), review, release and independent-retest process for that exercise. Keep the exercise separate from the healthy accepted Sprint 1 release. Do not fabricate findings, substitute a planned enhancement, or deliberately break the live assessed release.

If a controlled exercise or replacement tester has not been provided despite timely notification, record completed work and the unresolved dependency in the PDF. The instructor/TA will resolve that assessment dependency; do not claim unperformed work. A recorded Not Accepted decision is preferable to an unsupported sign-off when release-blocking defects remain.

## 4 What to submit

Submit one self-contained PDF per team through LEB2 by 22 November 2026 at 23:59. Identify both students and IDs, project theme/context, Sprint 1 scope, repository URL, tested release tag/commit, and deployed URL/version. Include the final UAT decision and names of independent UAT testers. The PDF is the normal grading surface. Include a contents table with Parts A–D, page ranges and evidence IDs.

### 4.1 Include the assessed material directly in the PDF

| Part | Include directly in the PDF |
| --- | --- |
| A | Your complete Sprint 1 UAT cases, requirement/AC references and handoff instructions; tested scope, safe accounts, setup/cleanup and version. |
| B | Other team/tester identities and tested Sprint 1 release; all agreed cases executed, actual results, dates/testers and selected readable evidence. |
| C | Outgoing reports, if any; all received findings with classification, severity, disposition and resolution status; documented minor limitations if any. |
| D | Traceable correction/retest evidence for release-blocking defects (or an approved controlled exercise if none arose), red/green test results, review/CI/redeployment and a signed/recorded Sprint 1 UAT decision with tester identity, version, date and any conditions. Your own accepted-defect chain: regression test excerpt and red/green output, relevant code diff, review/CI/release evidence, deployed fix version and partner retest/closure. Include contributions, AI use and limitations. |

> Conversion note: the source table labels its rows Part, A, B, C, D, which shifts each description down one row relative to Parts A–D in Section 3. This table aligns each description with the matching Part in Section 3 and merges the two Part D descriptions.

Appendices belong in the same PDF. Embed the relevant complete UAT cases, applicable Sprint 1 contract sections, finding/disposition records, test excerpts, selected execution output, and independent acceptance record; use internal page references. Include readable Issue/PR review text and results rather than only external links. Full repository history remains available for targeted corroboration. Do not paste the whole application source or unchanged SRS.

### 4.2 Make evidence quick to inspect

- Give screenshots/log excerpts an evidence ID, action, expected/actual result, time and relevant tested commit or deployed revision. Keep text readable and preserve enough context to support the claim.

- For automated tests, include the actual command, file/test IDs, relevant setup/assertion, tested version and passing/failing/skipped counts. Label manual tests as manual. A green icon alone is insufficient.

- Provide a short version table connecting the tested Sprint 1 baseline, defective commit(s) if any, test changes, corrected main release and deployed revision. Explain legitimate merge-commit differences. The final UAT acceptance decision must identify the exact release tested.

- Include a short contribution table, AI-use declaration and known limitations. Identify pair work; equal commit counts, full prompt transcripts and long reflections are not required.

There is no fixed PDF page limit. Include material once and refer to its pages. A separate video is not required. Supply safe test-account credentials through the private course handoff, and keep the assessed deployment and repository available through grading. Use synthetic data; do not include real passwords, tokens or production secrets in the PDF or Git.

## 5 Submission checklist

- Your PDF distinguishes your Sprint 1 Product Team responsibilities from independent UAT testing of the other team’s Sprint 1 product.

- UAT covers the agreed implemented Sprint 1 scope, with at least four cases covering the required categories, tested versions, actual results and independent tester identities.

- Every finding is recorded, classified and dispositioned; critical/major defects are corrected and independently re-tested, while minor limitations are fixed or explicitly accepted.

- Corrected defects have meaningful automated RED/GREEN evidence and reviewed CI/CD deployment; if none were found, the approved controlled exercise demonstrates that workflow separately.

- The UAT Team records its independent final Sprint 1 acceptance decision (Accepted, Accepted with minor limitations, or Not Accepted), with version, tester names, date and evidence.

- All assessed material is included in one PDF, including acceptance conditions and any recorded tester/exercise dependency.

Existing course deadline and academic-integrity policies apply. Report actual outcomes, including failed, blocked and Not Accepted results. Do not invent sign-offs or conceal outstanding defects. Defect count itself does not earn points; complete testing, responsible disposition and verified acceptance are the objectives.

// The demos of Parts A and B, run by bun run capture-demo 02.
//
//   continuity  the original Sprint 1 journey: req.anan submits a claim with
//               two items, rev.malee approves it, req.anan reads it back as
//               Approved with its history (F02-AC-01, F04-AC-04, F03-AC-05),
//               and a Meals item over 1,500.00 THB is refused (BR-12).
//   demo 1      F-05 Drafts: save a draft (F05-AC-01), read it back from the
//               list and from the API (F05-AC-04), submit it with a future
//               expense date and see the rule refuse it (F05-AC-09, BR-07),
//               correct and submit (F05-AC-08), and the 403 another requester
//               gets from the API for the draft (F05-AC-04, NFR-02).
//   demo 2      F-06 Return and resubmit: a reason below the BR-04 minimum is
//               refused, the return (F06-AC-01), the owner sees the reason and
//               resubmits (F06-AC-07, F06-AC-12), and after three returns the
//               reviewer has no Return action left (F06-AC-05, BR-14).
//   demo 3      F-07 Withdraw: the owner withdraws with a comment (F07-AC-01,
//               F07-AC-09), the claim leaves the reviewer's queue (F07-AC-08).
//
// The API half of the access evidence is the automated tests the report cites
// (F05-TEST-05, F06-TEST-04, F07-TEST-04); the one 403 in demo 1 is read in the
// browser from the API itself.
import type { Page } from "@playwright/test";

import type { Demo } from "../../scripts/lib/app.ts";

const fillItem = async (
  page: Page,
  i: number,
  [category, description, date, amount]: [string, string, string, string]
) => {
  await page.getByRole("combobox", { name: "Category" }).nth(i).click();
  await page.getByRole("option", { name: category }).click();
  await page.getByLabel("Description").nth(i).fill(description);
  await page.getByLabel("Expense date").nth(i).fill(date);
  await page.getByLabel("Amount").nth(i).fill(amount);
  await page
    .getByLabel("Receipt reference")
    .nth(i)
    .fill(`RC-${i + 1}`);
};

const CLAIM = /\/claims\/[0-9a-f-]{36}$/u;
const NO = /CLM-\d{4}-\d{6}/u;

const demo: Demo = async ({ page, base, login, logout, shot, day }) => {
  const newClaim = async (title: string, purpose: string) => {
    await page.goto(`${base}/claims/new`);
    await page.getByLabel("Title").fill(title);
    await page.getByLabel("Purpose").fill(purpose);
  };

  const claimNoOf = async () => {
    const header = page.getByRole("region", { name: "Claim", exact: true });

    const text =
      (await header.getByRole("heading", { level: 1 }).textContent()) ?? "";

    const claimNo = NO.exec(text)?.[0];

    if (claimNo === undefined) {
      throw new Error("no claim number in the claim heading");
    }

    return claimNo;
  };

  // Submits a one-item claim as the logged-in requester and returns its number.
  const submit = async (title: string) => {
    await newClaim(title, "Train fare for the workshop");
    await fillItem(page, 0, ["Travel", "Train return", day(-1), "250.00"]);
    await page.getByRole("button", { name: "Submit" }).click();

    const heading = page.getByRole("heading", {
      name: /^Claim CLM-\d{4}-\d{6} submitted$/u,
    });

    await heading.waitFor();

    return NO.exec((await heading.textContent()) ?? "")?.[0] ?? "";
  };

  const openFromList = async (claimNo: string) => {
    await page.getByRole("link", { name: claimNo }).first().click();
    await page.waitForURL(CLAIM);
  };

  const returnClaim = async (reason: string) => {
    await page.getByRole("button", { name: "Return" }).click();
    await page.getByLabel("Reason for return").fill(reason);
    await page.getByRole("button", { name: "Return" }).last().click();
    await page.getByText("Claim returned.").first().waitFor();
  };

  // ---- Continuity: the Sprint 1 journey on this release ----
  await login("req.anan");
  await newClaim(`Continuity ${Date.now()}`, "Train fare and a client lunch");
  await fillItem(page, 0, ["Travel", "Train return", day(-1), "250.00"]);
  await page.getByRole("button", { name: "Add line item" }).click();
  await fillItem(page, 1, ["Meals", "Client lunch", day(-1), "400.00"]);
  await shot("a1-1-new-claim-two-items.png");
  await page.getByRole("button", { name: "Submit" }).click();

  const confirmation = page.getByRole("heading", {
    name: /^Claim CLM-\d{4}-\d{6} submitted$/u,
  });

  await confirmation.waitFor();

  const journeyNo =
    NO.exec((await confirmation.textContent()) ?? "")?.[0] ?? "";

  await shot("a1-2-claim-submitted.png");

  await logout();
  await login("rev.malee");
  await page.goto(`${base}/reviews`);
  await page.getByRole("link", { name: journeyNo }).first().waitFor();
  await openFromList(journeyNo);
  await page.getByRole("button", { name: "Approve" }).click();
  await page.getByRole("button", { name: "Approve" }).last().click();
  await page.getByText("Claim approved.").first().waitFor();
  await shot("a1-3-reviewer-approves.png");

  await logout();
  await login("req.anan");
  await page.goto(`${base}/claims`);
  await shot("a1-4-my-claims-approved.png");
  await openFromList(journeyNo);
  await page.getByRole("region", { name: "History" }).waitFor();
  await shot("a1-5-claim-detail-approved-with-history.png");

  await newClaim(`Continuity rule ${Date.now()}`, "Meals over the limit");
  await fillItem(page, 0, ["Meals", "Team dinner", day(-1), "1600.00"]);
  await page.getByRole("button", { name: "Submit" }).click();
  await page
    .getByText(/1,500\.00/u)
    .first()
    .waitFor();
  await shot("a1-6-meals-limit-refused.png");

  // ---- Demo 1: F-05 drafts ----
  await logout();
  await login("req.anan");
  await newClaim(`Draft demo ${Date.now()}`, "Workshop travel");
  await fillItem(page, 0, ["Travel", "Train return", day(-1), "250.00"]);
  await page.getByRole("button", { name: "Save as draft" }).click();
  await page.waitForURL(CLAIM);
  await page.getByRole("region", { name: "Claim", exact: true }).waitFor();

  const draftNo = await claimNoOf();
  const draftUrl = page.url();
  const draftId = draftUrl.slice(draftUrl.lastIndexOf("/") + 1);

  await shot("d1-1-draft-saved.png");

  await page.goto(`${base}/claims`);
  await page.getByRole("link", { name: draftNo }).first().waitFor();
  await shot("d1-2-my-claims-lists-the-draft.png");

  // Retrieval of the saved data: the API's own answer, read from PostgreSQL.
  await page.goto(`${base}/api/v1/claims/${draftId}`);
  await shot("d1-3-api-returns-the-saved-draft.png");

  // BR-07 is checked at submission, not at saving.
  await page.goto(`${draftUrl}/edit`);
  await page.getByLabel("Expense date").first().fill(day(1));
  await page.getByRole("button", { name: "Save changes" }).click();
  await page.getByRole("button", { name: "Submit claim" }).waitFor();
  await page.getByRole("button", { name: "Submit claim" }).click();
  await page
    .getByText(/Line 1/u)
    .first()
    .waitFor();
  await shot("d1-4-submit-refused-future-date.png");

  await page.goto(`${draftUrl}/edit`);
  await page.getByLabel("Expense date").first().fill(day(-1));
  await page.getByRole("button", { name: "Save changes" }).click();
  await page.getByRole("button", { name: "Submit claim" }).waitFor();
  await page.getByRole("button", { name: "Submit claim" }).click();
  await page.getByText("Claim submitted.").first().waitFor();
  await shot("d1-5-draft-submitted.png");

  // A second draft, read by another requester through the API: 403.
  await newClaim(`Private draft ${Date.now()}`, "Not for others");
  await fillItem(page, 0, ["Travel", "Taxi", day(-1), "120.00"]);
  await page.getByRole("button", { name: "Save as draft" }).click();
  await page.waitForURL(CLAIM);

  const privateUrl = page.url();
  const privateId = privateUrl.slice(privateUrl.lastIndexOf("/") + 1);

  await logout();
  await login("req.ploy");
  await page.goto(`${base}/api/v1/claims/${privateId}`);
  await shot("d1-6-other-requester-gets-403.png");

  // ---- Demo 2: F-06 return and resubmit ----
  await logout();
  await login("req.anan");

  const returnedNo = await submit(`Return demo ${Date.now()}`);

  await logout();
  await login("rev.malee");
  await page.goto(`${base}/reviews`);
  await openFromList(returnedNo);
  await page.getByRole("button", { name: "Return" }).click();
  await page.getByLabel("Reason for return").fill("Too short");
  await shot("d2-1-short-reason-refused.png");
  await page.getByRole("button", { name: "Cancel" }).click();
  await returnClaim("Please attach the taxi receipt");
  await shot("d2-2-claim-returned.png");

  await logout();
  await login("req.anan");
  await page.goto(`${base}/claims`);
  await openFromList(returnedNo);
  await page.getByText("Returned 1 / 3").waitFor();
  await shot("d2-3-owner-sees-reason-and-count.png");
  await page.getByRole("button", { name: "Resubmit" }).click();
  await page.getByText("Claim resubmitted.").first().waitFor();
  await shot("d2-4-claim-resubmitted.png");

  // Two more return and resubmit rounds reach the limit of three (BR-14).
  for (const round of [2, 3]) {
    await logout();
    await login("rev.malee");
    await page.goto(`${base}/reviews`);
    await openFromList(returnedNo);
    await returnClaim(`Round ${round}: please check the receipt`);
    await logout();
    await login("req.anan");
    await page.goto(`${base}/claims`);
    await openFromList(returnedNo);
    await page.getByRole("button", { name: "Resubmit" }).click();
    await page.getByText("Claim resubmitted.").first().waitFor();
  }

  await logout();
  await login("rev.malee");
  await page.goto(`${base}/reviews`);
  await openFromList(returnedNo);
  await page.getByText("Returned 3 / 3").waitFor();
  await shot("d2-5-no-fourth-return.png");

  // ---- Demo 3: F-07 withdraw ----
  await logout();
  await login("req.anan");

  const withdrawnNo = await submit(`Withdraw demo ${Date.now()}`);

  await page.goto(`${base}/claims`);
  await openFromList(withdrawnNo);
  await page.getByRole("button", { name: "Withdraw", exact: true }).click();
  await page
    .getByRole("dialog")
    .getByLabel("Comment", { exact: true })
    .fill("No longer needed");
  await shot("d3-1-withdraw-dialog.png");
  await page.getByRole("button", { name: "Withdraw claim" }).click();
  await page.getByText("Claim withdrawn.").first().waitFor();
  await shot("d3-2-claim-withdrawn-with-history.png");

  await logout();
  await login("rev.malee");
  await page.goto(`${base}/reviews`);
  await page.getByRole("table").waitFor();
  await shot("d3-3-review-queue-without-the-claim.png");
};

export default demo;

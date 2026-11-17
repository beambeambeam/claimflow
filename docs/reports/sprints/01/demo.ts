// Demos 1, 2 and 3 of Part F.2, run by bun run capture-demo 01.
//
//   demo 1  submit, approve, retrieve: req.anan submits a claim with two line
//           items (F02-AC-01); rev.malee approves it from the Review Queue
//           (F04-AC-04); req.anan opens My Claims and the claim detail, which
//           shows Approved with the SUBMIT and APPROVE history (F03-AC-01,
//           F04-AC-20). The journey is the same as E2E test F04-TEST-23.
//   demo 2  rule rejection under the field: a Meals line over 1,500.00 THB
//           (BR-12, F02-AC-06) and a future expense date (BR-07, F02-AC-07),
//           then the corrected claim accepted with its claim number.
//   demo 3  login and logout (F-01), and a forbidden operation: a Finance
//           Officer opens New Claim and is refused (F02-AC-09, BR-15). The API
//           half of the evidence is the automated tests F02-TEST-14 and
//           F01-TEST-12/13, which the report cites; hiding a form proves
//           nothing by itself.
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

const demo: Demo = async ({ page, base, login, logout, shot, day }) => {
  // ---- Demo 1: submit, approve, retrieve ----
  await login("req.anan");
  await page.goto(`${base}/claims/new`);
  await page.getByLabel("Title").fill(`Demo approve ${Date.now()}`);
  await page.getByLabel("Purpose").fill("Train fare and a client lunch");
  await fillItem(page, 0, ["Travel", "Train return", day(-1), "250.00"]);
  await page.getByRole("button", { name: "Add line item" }).click();
  await fillItem(page, 1, ["Meals", "Client lunch", day(-1), "400.00"]);
  await shot("demo1-1-new-claim-two-items.png");
  await page.getByRole("button", { name: "Submit" }).click();

  const confirmation = page.getByRole("heading", {
    name: /^Claim CLM-\d{4}-\d{6} submitted$/u,
  });

  await confirmation.waitFor();

  const claimNo = /CLM-\d{4}-\d{6}/u.exec(
    (await confirmation.textContent()) ?? ""
  )?.[0];

  if (claimNo === undefined) {
    throw new Error("no claim number in the confirmation heading");
  }

  await shot("demo1-2-claim-submitted.png");

  await logout();
  await login("rev.malee");
  await page.goto(`${base}/reviews`);
  const link = page.getByRole("link", { name: claimNo }).first();
  await link.waitFor();
  await shot("demo1-3-review-queue-lists-the-claim.png");
  await link.click();
  await page.waitForURL(CLAIM);
  await page.getByRole("button", { name: "Approve" }).click();
  await page.getByRole("button", { name: "Approve" }).last().waitFor();
  await shot("demo1-4-approve-dialog.png");
  await page.getByRole("button", { name: "Approve" }).last().click();
  await page.getByText("Claim approved.").first().waitFor();
  await shot("demo1-5-reviewer-sees-claim-approved.png");

  await logout();
  await login("req.anan");
  await link.waitFor();
  await shot("demo1-6-my-claims-after-approval.png");
  await link.click();
  await page.waitForURL(CLAIM);
  await page
    .getByRole("region", { name: "History" })
    .getByText("Submitted → Approved")
    .waitFor();
  await shot("demo1-7-claim-detail-approved-with-history.png");
  await logout();

  // ---- Demo 2: rule rejection, then acceptance ----
  await login("req.anan");
  await page.goto(`${base}/claims/new`);
  await page.getByLabel("Title").fill("Bangkok workshop travel");
  await page
    .getByLabel("Purpose")
    .fill("Train fares and meals for the workshop");
  await fillItem(page, 0, ["Travel", "Train return", day(-1), "250.00"]);
  await page.getByRole("button", { name: "Add line item" }).click();
  await fillItem(page, 1, ["Meals", "Team dinner", day(1), "1600.00"]);
  await shot("demo2-1-form-before-submit.png");
  await page.getByRole("button", { name: "Submit" }).click();
  await page
    .getByText(/cannot be in the future/iu)
    .first()
    .waitFor();
  await shot("demo2-2-rule-errors-under-fields.png");

  await page.getByLabel("Expense date").nth(1).fill(day(-1));
  await page.getByLabel("Amount").nth(1).fill("1000.00");
  await page.getByRole("button", { name: "Submit" }).click();
  await page.getByRole("heading", { name: /submitted$/u }).waitFor();
  await shot("demo2-3-corrected-claim-submitted.png");

  // ---- Demo 3: login, logout, forbidden operation ----
  await page.getByRole("button", { name: "Log out" }).click();
  await page.waitForURL((u) => u.pathname.startsWith("/login"));
  await shot("demo3-1-after-logout-login-screen.png");

  // Anonymous: redirected to login with ?redirect.
  await page.goto(`${base}/claims`);
  await page.waitForURL((u) => u.pathname.startsWith("/login"));
  await shot("demo3-2-anonymous-claims-redirects-to-login.png");

  await login("fin.wanida");
  await shot("demo3-3-finance-officer-landing.png");
  await page.goto(`${base}/claims/new`);
  await page.getByText("You can't create claims.").waitFor();
  await shot("demo3-4-finance-officer-cannot-create-claims.png");
};

export default demo;

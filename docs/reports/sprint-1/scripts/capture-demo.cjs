/* eslint-disable typescript/no-unsafe-call, typescript/no-unsafe-member-access, typescript/no-unsafe-assignment, typescript/no-unsafe-return, typescript/strict-boolean-expressions, typescript/use-unknown-in-catch-callback-variable, unicorn/no-await-expression-member, unicorn/filename-case, eslint/require-unicode-regexp, eslint/func-style, eslint/no-inline-comments, promise/prefer-await-to-callbacks, typescript/prefer-nullish-coalescing, typescript/no-unsafe-argument */
// A standalone capture script, run by hand with node: untyped, not part of the app.
// Capture demos 1, 2 and 3 of Part F.2 from a running local stack.
//
//   demo 1  submit, approve, retrieve: req.anan submits a claim with two line
//           items (F02-AC-01); rev.malee approves it from the Review Queue
//           (F04-AC-04); req.anan opens My Claims and the claim detail, which
//           shows Approved with the SUBMIT and APPROVE history (F03-AC-01,
//           F04-AC-20). The journey is the same as E2E test F04-TEST-23.
//   demo 2  rule rejection under the field: a Meals line over 1,500.00 THB (BR-12,
//           F02-AC-06) and a future expense date (BR-07, F02-AC-07), then the
//           corrected claim accepted with its claim number.
//   demo 3  login and logout (F-01), and a forbidden operation: a Finance Officer
//           opens New Claim and is refused (F02-AC-09, BR-15). The API half of the
//           evidence is the automated tests F02-TEST-14 and F01-TEST-12/13, which
//           the report cites; hiding a form proves nothing by itself.
//
// Prerequisites (see environment notes in the report README): Postgres with the
// migrations applied and `seed` run, the API on :8080, the web dev server on
// :3000. This script does not start any of them.
//
//   SEED_USER_PASSWORD=<local value> node scripts/capture-demo.cjs
//   BASE_URL=http://localhost:3000 can override the address.
//
// Playwright's bundled browser is not installed on this machine, so it launches
// the installed Chrome. Output: figures/demo/*.png. The password is typed into
// the form but never written to a file or a screenshot (the field is masked).
const path = require("node:path");
const fs = require("node:fs");

const web = path.resolve(__dirname, "../../../../apps/web");

const { chromium } = require(
  require.resolve("@playwright/test", { paths: [web] })
);

const base = process.env.BASE_URL || "http://localhost:3000";
const out = path.resolve(__dirname, "../figures/demo");
const pw = process.env.SEED_USER_PASSWORD;

if (!pw) {
  console.error("Set SEED_USER_PASSWORD to the value used for `seed`.");
  process.exit(1);
}

fs.mkdirSync(out, { recursive: true });

// A date n days from today in Bangkok time, as YYYY-MM-DD.
const day = (n) => {
  const d = new Date(Date.now() + 7 * 3600e3);
  d.setUTCDate(d.getUTCDate() + n);

  return d.toISOString().slice(0, 10);
};

async function login(page, username) {
  await page.goto(`${base}/login`);
  await page.getByLabel("Username").fill(username);
  await page.getByLabel("Password").fill(pw);
  await page.getByRole("button", { name: "Log in" }).click();
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
}

async function fillItem(page, i, category, description, date, amount) {
  await page.getByRole("combobox", { name: "Category" }).nth(i).click();
  await page.getByRole("option", { name: category }).click();
  await page.getByLabel("Description").nth(i).fill(description);
  await page.getByLabel("Expense date").nth(i).fill(date);
  await page.getByLabel("Amount").nth(i).fill(amount);
  await page
    .getByLabel("Receipt reference")
    .nth(i)
    .fill(`RC-${i + 1}`);
}

// Ends the session, so the same page can log in as someone else.
const logout = (page) => page.context().clearCookies();

const shot = (page, name) =>
  page.screenshot({ fullPage: true, path: path.join(out, name) });

(async () => {
  const browser = await chromium.launch({ channel: "chrome" });

  const page = await (
    await browser.newContext({ viewport: { height: 1000, width: 1280 } })
  ).newPage();

  // ---- Demo 1: submit, approve, retrieve --------------------------------------
  await login(page, "req.anan");
  await page.goto(`${base}/claims/new`);
  const title = `Demo approve ${Date.now()}`;
  await page.getByLabel("Title").fill(title);
  await page.getByLabel("Purpose").fill("Train fare and a client lunch");
  await fillItem(page, 0, "Travel", "Train return", day(-1), "250.00");
  await page.getByRole("button", { name: "Add line item" }).click();
  await fillItem(page, 1, "Meals", "Client lunch", day(-1), "400.00");
  await shot(page, "demo1-1-new-claim-two-items.png");
  await page.getByRole("button", { name: "Submit" }).click();

  const confirmation = page.getByRole("heading", {
    name: /^Claim CLM-\d{4}-\d{6} submitted$/,
  });

  await confirmation.waitFor();
  const [claimNo] = /CLM-\d{4}-\d{6}/.exec(await confirmation.textContent());
  await shot(page, "demo1-2-claim-submitted.png");

  await logout(page);
  await login(page, "rev.malee");
  await page.goto(`${base}/reviews`);
  await page.getByRole("link", { name: claimNo }).first().waitFor();
  await shot(page, "demo1-3-review-queue-lists-the-claim.png");
  await page.getByRole("link", { name: claimNo }).first().click();
  await page.waitForURL(/\/claims\/[0-9a-f-]{36}$/);
  await page.getByRole("button", { name: "Approve" }).click();
  await page.getByRole("button", { name: "Approve" }).last().waitFor();
  await shot(page, "demo1-4-approve-dialog.png");
  await page.getByRole("button", { name: "Approve" }).last().click();
  await page.getByText("Claim approved.").first().waitFor();
  await shot(page, "demo1-5-reviewer-sees-claim-approved.png");

  await logout(page);
  await login(page, "req.anan");
  await page.getByRole("link", { name: claimNo }).first().waitFor();
  await shot(page, "demo1-6-my-claims-after-approval.png");
  await page.getByRole("link", { name: claimNo }).first().click();
  await page.waitForURL(/\/claims\/[0-9a-f-]{36}$/);
  const history = page.getByRole("region", { name: "History" });
  await history.getByText("Submitted → Approved").waitFor();
  await shot(page, "demo1-7-claim-detail-approved-with-history.png");
  await logout(page);

  // ---- Demo 2: rule rejection, then acceptance --------------------------------
  await login(page, "req.anan");
  await page.goto(`${base}/claims/new`);
  await page.getByLabel("Title").fill("Bangkok workshop travel");
  await page
    .getByLabel("Purpose")
    .fill("Train fares and meals for the workshop");
  await fillItem(page, 0, "Travel", "Train return", day(-1), "250.00");
  await page.getByRole("button", { name: "Add line item" }).click();
  await fillItem(page, 1, "Meals", "Team dinner", day(1), "1600.00");
  await shot(page, "demo2-1-form-before-submit.png");
  await page.getByRole("button", { name: "Submit" }).click();
  await page
    .getByText(/cannot be in the future/i)
    .first()
    .waitFor();
  await shot(page, "demo2-2-rule-errors-under-fields.png");

  await page.getByLabel("Expense date").nth(1).fill(day(-1));
  await page.getByLabel("Amount").nth(1).fill("1000.00");
  await page.getByRole("button", { name: "Submit" }).click();
  await page.getByRole("heading", { name: /submitted$/ }).waitFor();
  await shot(page, "demo2-3-corrected-claim-submitted.png");

  // ---- Demo 3: login, logout, forbidden operation ------------------------------
  await page.getByRole("button", { name: "Log out" }).click();
  await page.waitForURL((u) => u.pathname.startsWith("/login"));
  await shot(page, "demo3-1-after-logout-login-screen.png");

  await page.goto(`${base}/claims`); // anonymous: redirected to login with ?redirect
  await page.waitForURL((u) => u.pathname.startsWith("/login"));
  await shot(page, "demo3-2-anonymous-claims-redirects-to-login.png");

  await login(page, "fin.wanida");
  await shot(page, "demo3-3-finance-officer-landing.png");
  await page.goto(`${base}/claims/new`);
  await page.getByText("You can't create claims.").waitFor();
  await shot(page, "demo3-4-finance-officer-cannot-create-claims.png");

  await browser.close();
  console.log(`Screenshots written to ${out}`);
})().catch((error) => {
  console.error(error);
  process.exit(1);
});

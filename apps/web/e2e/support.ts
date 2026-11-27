import { expect } from "@playwright/test";
import type { Page } from "@playwright/test";

// The same value `api seed` hashes. Set it in the environment for the test run. It is never committed.
export const password = process.env.SEED_USER_PASSWORD ?? "";

if (password === "") {
  throw new Error(
    "SEED_USER_PASSWORD must be set to the value used by `api seed`"
  );
}

// An expense date one day back, in the Asia/Bangkok calendar, so BR-07 passes on any day the test runs.
const yesterday = () => {
  const bangkokNow = new Date(Date.now() + 7 * 60 * 60 * 1000);
  bangkokNow.setUTCDate(bangkokNow.getUTCDate() - 1);

  return bangkokNow.toISOString().slice(0, 10);
};

// Logs in and waits for the landing page of the role: My Claims for a requester, Review Queue for a reviewer.
export const logIn = async (
  page: Page,
  username: string,
  landing = /\/claims$/u
) => {
  await page.goto("/login");
  await page.getByLabel("Username").fill(username);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Log in" }).click();
  await expect(page).toHaveURL(landing);
};

// Ends the session, so the same page can log in as someone else.
export const logOut = async (page: Page) => {
  await page.context().clearCookies();
};

// Fills the `index`-th line item row with a Travel item.
export const fillItem = async (
  page: Page,
  index: number,
  { amount, description }: { amount: string; description: string }
) => {
  await page.getByRole("combobox", { name: "Category" }).nth(index).click();
  await page.getByRole("option", { name: "Travel" }).click();
  await page.getByLabel("Description").nth(index).fill(description);
  await page.getByLabel("Expense date").nth(index).fill(yesterday());
  await page.getByLabel("Amount").nth(index).fill(amount);
  await page
    .getByLabel("Receipt reference")
    .nth(index)
    .fill(`SRT-${index + 1}`);
};

// Opens New Claim from the main navigation. The page has its own New claim button, so the navigation scopes the link.
export const openNewClaim = async (page: Page) => {
  await page
    .getByRole("navigation", { name: "Main navigation" })
    .getByRole("link", { name: "New Claim" })
    .click();
  await expect(page).toHaveURL(/\/claims\/new$/u);
};

// Submits a claim with one Travel item of `amount` as the logged-in requester and returns its claim number.
export const submitClaim = async (
  page: Page,
  title: string,
  amount = "250.00"
) => {
  await openNewClaim(page);
  await page.getByLabel("Title").fill(title);
  await page.getByLabel("Purpose").fill("Train fare for the workshop");
  await fillItem(page, 0, { amount, description: "Train return" });
  await page.getByRole("button", { name: "Submit" }).click();

  const confirmation = page.getByRole("heading", {
    name: /^Claim CLM-\d{4}-\d{6} submitted$/u,
  });

  await expect(confirmation).toBeVisible();

  const claimNo = /CLM-\d{4}-\d{6}/u.exec(
    (await confirmation.textContent()) ?? ""
  )?.[0];

  if (claimNo === undefined) {
    throw new Error("The confirmation heading has no claim number");
  }

  return claimNo;
};

// The Review Queue pages the spec looks through before it gives up: 20 claims each, more than the load data holds.
const maxQueuePages = 10;

// Opens the claim from the Review Queue of the logged-in reviewer. The queue lists the oldest submission first, so with
// the F-10/02 load data a claim the spec just submitted is on a later page: each page is checked in turn.
export const openFromReviewQueue = async (
  page: Page,
  claimNo: string,
  queuePage = 1
): Promise<void> => {
  if (queuePage > 1) {
    await page.goto(`/reviews?page=${queuePage}`);
  }

  // Wait for the rows, not the skeleton, before looking for the claim.
  await expect(
    page.getByRole("table").getByRole("link", { name: /^CLM-/u }).first()
  ).toBeVisible();

  const link = page.getByRole("link", { name: claimNo }).first();

  if ((await link.count()) === 0) {
    if (queuePage >= maxQueuePages) {
      throw new Error(
        `${claimNo} is not in the first ${maxQueuePages} pages of the Review Queue`
      );
    }

    await openFromReviewQueue(page, claimNo, queuePage + 1);

    return;
  }

  await link.click();
  await expect(page).toHaveURL(/\/claims\/[0-9a-f-]{36}$/u);
};

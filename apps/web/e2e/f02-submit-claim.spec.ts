import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";

// The same value `api seed` hashes. Set it in the environment for the test run. It is never committed.
const password = process.env.SEED_USER_PASSWORD ?? "";

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

// Fills the `index`-th line item row with a Travel item.
const fillItem = async (
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

test("F02-TEST-21: req.anan submits a claim with 2 line items and sees the claim number and Submitted", async ({
  page,
}) => {
  await page.goto("/login");
  await page.getByLabel("Username").fill("req.anan");
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Log in" }).click();
  await expect(page).toHaveURL(/\/claims$/u);

  await page.getByRole("link", { name: "New Claim" }).click();
  await expect(page).toHaveURL(/\/claims\/new$/u);

  await page.getByLabel("Title").fill("Bangkok workshop travel");
  await page
    .getByLabel("Purpose")
    .fill("Train fares and hotel for the workshop");
  await fillItem(page, 0, { amount: "250.00", description: "Train return" });
  await page.getByRole("button", { name: "Add line item" }).click();
  await fillItem(page, 1, {
    amount: "1000.00",
    description: "Hotel, one night",
  });
  await expect(page.getByText("Total ฿1,250.00")).toBeVisible();

  await page.getByRole("button", { name: "Submit" }).click();

  await expect(
    page.getByRole("heading", { name: /^Claim CLM-\d{4}-\d{6} submitted$/u })
  ).toBeVisible();
  await expect(page.getByText("Submitted", { exact: true })).toBeVisible();
  await expect(page.getByText("Total ฿1,250.00")).toBeVisible();
});

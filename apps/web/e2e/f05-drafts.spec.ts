import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";

import {
  fillItem,
  logIn,
  logOut,
  openFromReviewQueue,
  openNewClaim,
} from "./support";

const saveDraft = async (page: Page, title: string) => {
  await openNewClaim(page);
  await page.getByLabel("Title").fill(title);
  await page.getByLabel("Purpose").fill("Travel for the workshop");
  await fillItem(page, 0, { amount: "250.00", description: "Train return" });
  await page.getByRole("button", { name: "Save as draft" }).click();
  await expect(page).toHaveURL(/\/claims\/[0-9a-f-]{36}$/u);
  const header = page.getByRole("region", { name: "Claim", exact: true });
  await expect(header.getByText("Draft", { exact: true })).toBeVisible();

  const claimNo = await header.getByRole("heading", { level: 1 }).textContent();

  if (claimNo === null) {
    throw new Error("The draft heading has no claim number");
  }

  return claimNo.trim();
};

test("F05-TEST-25: save, edit, submit, review and delete drafts", async ({
  page,
}) => {
  const suffix = Date.now();
  const title = `Workshop travel ${suffix}`;
  const updated = `Updated workshop travel ${suffix}`;

  await logIn(page, "req.anan");
  const claimNo = await saveDraft(page, title);
  await page.getByRole("link", { name: "Edit", exact: true }).click();
  await expect(page).toHaveURL(/\/edit$/u);
  await expect(page.getByLabel("Title")).toHaveValue(title);
  await page.getByLabel("Title").fill(updated);
  await page.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByText(updated, { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Submit claim" }).click();

  const history = page.getByRole("region", { name: "History" });
  await expect(history.getByRole("listitem")).toHaveCount(1);
  await expect(history).toContainText("Draft → Submitted");

  await logOut(page);
  await logIn(page, "rev.malee", /\/reviews$/u);
  await expect(
    page.getByRole("row", { name: new RegExp(claimNo, "u") })
  ).toContainText(updated);
  await openFromReviewQueue(page, claimNo);
  await expect(history.getByRole("listitem")).toHaveCount(1);
  await expect(history).toContainText("Draft → Submitted");

  await logOut(page);
  await logIn(page, "req.anan");
  const deletedNo = await saveDraft(page, `Draft to delete ${suffix}`);
  await page.getByRole("button", { name: "Delete draft" }).click();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Delete", exact: true })
    .click();
  await expect(page).toHaveURL(/\/claims$/u);
  await expect(page.getByRole("table")).toBeVisible();
  await expect(page.getByRole("link", { name: deletedNo })).toHaveCount(0);
});

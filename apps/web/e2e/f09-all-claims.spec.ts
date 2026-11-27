import { expect, test } from "@playwright/test";

import { logIn, logOut, openFromReviewQueue, submitClaim } from "./support";

const reviewerLanding = /\/reviews$/u;

// Other specs add claims to the same database, so this one makes its own: two claims that share a unique word in the
// title, one approved and one left submitted. Both are the newest claims, so they are on the first page.
test("F09-TEST-13: rev.malee filters All Claims by Approved, searches part of a title, and Clear filters brings the full list back", async ({
  page,
}) => {
  const word = `Filterable${Date.now()}`;

  await logIn(page, "req.anan");
  const approved = await submitClaim(page, `${word} taxi`);
  const submitted = await submitClaim(page, `${word} hotel`);

  await logOut(page);
  await logIn(page, "rev.malee", reviewerLanding);
  await openFromReviewQueue(page, approved);
  await page.getByRole("button", { name: "Approve" }).click();
  await page.getByRole("button", { name: "Approve" }).last().click();
  await expect(page.getByText("Claim approved.").first()).toBeVisible();

  await page
    .getByRole("navigation", { name: "Main navigation" })
    .getByRole("link", { name: "All Claims" })
    .click();
  await expect(page).toHaveURL(/\/all-claims$/u);
  await expect(page.getByRole("link", { name: approved })).toBeVisible();
  await expect(page.getByRole("link", { name: submitted })).toBeVisible();

  const rows = page.getByRole("row").filter({ hasNotText: "Claim no." });

  await page.getByRole("combobox", { name: "Status" }).click();
  await page.getByRole("option", { name: "Approved" }).click();
  await expect(page).toHaveURL(/status=APPROVED/u);
  await expect(page.getByRole("link", { name: submitted })).toBeHidden();
  await expect(page.getByRole("link", { name: approved })).toBeVisible();

  // Part of the word, in another case.
  await page.getByLabel("Search title").fill(word.slice(0, -3).toLowerCase());
  await expect(page).toHaveURL(/q=filterable/u);
  await expect(rows).toHaveCount(1);
  await expect(rows.first()).toContainText(approved);
  await expect(rows.first()).toContainText("Approved");

  await page.getByRole("button", { name: "Clear filters" }).click();
  await expect(page).toHaveURL(/\/all-claims$/u);
  await expect(page.getByLabel("Search title")).toHaveValue("");
  await expect(page.getByRole("link", { name: approved })).toBeVisible();
  await expect(page.getByRole("link", { name: submitted })).toBeVisible();
});

import { expect, test } from "@playwright/test";

import { findInReviewQueue, logIn, logOut, submitClaim } from "./support";

const reviewerLanding = /\/reviews$/u;

test("F07-TEST-12: req.anan submits and withdraws a claim with a comment, it shows Withdrawn in My Claims, and rev.malee's Review Queue no longer lists it", async ({
  page,
}) => {
  await logIn(page, "req.anan");
  const claimNo = await submitClaim(page, `Withdraw journey ${Date.now()}`);

  // The claim is in the reviewer's queue before it is withdrawn.
  await logOut(page);
  await logIn(page, "rev.malee", reviewerLanding);
  await expect(await findInReviewQueue(page, claimNo)).toBeVisible();

  await logOut(page);
  await logIn(page, "req.anan");
  await page.getByRole("link", { name: claimNo }).first().click();
  await expect(page).toHaveURL(/\/claims\/[0-9a-f-]{36}$/u);
  await page.getByRole("button", { name: "Withdraw", exact: true }).click();

  const dialog = page.getByRole("dialog");

  await expect(dialog.getByText("0/500")).toBeVisible();
  await dialog.getByLabel("Comment", { exact: true }).fill("No longer needed");
  await dialog.getByRole("button", { name: "Withdraw claim" }).click();
  await expect(page.getByText("Claim withdrawn.").first()).toBeVisible();

  const header = page.getByRole("region", { name: "Claim", exact: true });

  await expect(header.getByText("Withdrawn", { exact: true })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Withdraw", exact: true })
  ).toHaveCount(0);

  const history = page.getByRole("region", { name: "History" });

  await expect(history.getByText("Submitted → Withdrawn")).toBeVisible();
  await expect(history.getByText("No longer needed")).toBeVisible();

  // My Claims lists it as Withdrawn.
  await page.goto("/claims");
  await expect(
    page.getByRole("row", { name: new RegExp(claimNo, "u") })
  ).toContainText("Withdrawn");

  // The reviewer's queue no longer lists it, and the reviewer can still open it.
  await logOut(page);
  await logIn(page, "rev.malee", reviewerLanding);
  await expect(page.getByRole("table")).toBeVisible();
  await expect(page.getByRole("link", { name: claimNo })).toHaveCount(0);
});

import { expect, test } from "@playwright/test";

import { logIn, logOut, openFromReviewQueue, submitClaim } from "./support";

const reviewerLanding = /\/reviews$/u;

test("F08-TEST-18: req.anan submits 15,000.00, rev.malee approves to Pending finance, fin.wanida final-approves from the Finance Queue, and req.anan sees Approved", async ({
  page,
}) => {
  await logIn(page, "req.anan");

  const claimNo = await submitClaim(
    page,
    `Two-stage journey ${Date.now()}`,
    "15000.00"
  );

  await logOut(page);
  await logIn(page, "rev.malee", reviewerLanding);
  await openFromReviewQueue(page, claimNo);
  await page.getByRole("button", { name: "Approve" }).click();
  await page.getByRole("button", { name: "Approve" }).last().click();
  await expect(
    page
      .getByText(
        "Claim approved. It now waits for a final decision by Finance."
      )
      .first()
  ).toBeVisible();
  await expect(
    page.getByText("Pending finance", { exact: true }).first()
  ).toBeVisible();
  await expect(
    page.getByText("Waiting for a final decision by Finance.")
  ).toBeVisible();

  await logOut(page);
  await logIn(page, "fin.wanida", reviewerLanding);
  const navigation = page.getByRole("navigation", { name: "Main navigation" });
  await navigation.getByRole("link", { name: "Finance Queue" }).click();
  await expect(page).toHaveURL(/\/finance$/u);
  await page.getByRole("link", { name: claimNo }).first().click();
  await expect(page).toHaveURL(/\/claims\/[0-9a-f-]{36}$/u);

  await page.getByRole("button", { name: "Final approve" }).click();
  await expect(
    page.getByText(`Approve claim ${claimNo} as the final decision?`)
  ).toBeVisible();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Approve" })
    .click();
  await expect(page.getByText("Claim approved.").first()).toBeVisible();

  // The decided claim leaves the Finance Queue.
  await navigation.getByRole("link", { name: "Finance Queue" }).click();
  await expect(
    page.getByRole("heading", { name: "Finance Queue" })
  ).toBeVisible();
  await expect(page.getByRole("link", { name: claimNo })).toHaveCount(0);

  await logOut(page);
  await logIn(page, "req.anan");
  await page.getByRole("link", { name: claimNo }).first().click();

  await expect(
    page.getByText("Approved", { exact: true }).first()
  ).toBeVisible();

  const history = page.getByRole("region", { name: "History" });

  await expect(history.getByText("Approved", { exact: true })).toBeVisible();
  await expect(history.getByText("Final approval")).toBeVisible();
  await expect(history.getByText("Submitted → Pending finance")).toBeVisible();
  await expect(history.getByText("Pending finance → Approved")).toBeVisible();
});

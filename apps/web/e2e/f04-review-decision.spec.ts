import { expect, test } from "@playwright/test";

import { logIn, logOut, openFromReviewQueue, submitClaim } from "./support";

const reviewerLanding = /\/reviews$/u;

test("F04-TEST-23: req.anan submits, rev.malee approves from the Review Queue, and req.anan sees Approved with SUBMIT and APPROVE in history", async ({
  page,
}) => {
  await logIn(page, "req.anan");
  const claimNo = await submitClaim(page, `Approve journey ${Date.now()}`);

  await logOut(page);
  await logIn(page, "rev.malee", reviewerLanding);
  await openFromReviewQueue(page, claimNo);
  await page.getByRole("button", { name: "Approve" }).click();
  await page.getByRole("button", { name: "Approve" }).last().click();
  await expect(page.getByText("Claim approved.").first()).toBeVisible();

  await logOut(page);
  await logIn(page, "req.anan");
  await page.getByRole("link", { name: claimNo }).first().click();

  await expect(
    page.getByText("Approved", { exact: true }).first()
  ).toBeVisible();

  const history = page.getByRole("region", { name: "History" });

  await expect(history.getByText("– → Submitted")).toBeVisible();
  await expect(history.getByText("Submitted → Approved")).toBeVisible();
  await expect(history.getByText("Anan K.")).toBeVisible();
  await expect(history.getByText("Malee S.")).toBeVisible();
});

test("F04-TEST-24: rev.malee rejects a submitted claim with a reason and req.anan sees Rejected with the reason", async ({
  page,
}) => {
  await logIn(page, "req.anan");
  const claimNo = await submitClaim(page, `Reject journey ${Date.now()}`);

  await logOut(page);
  await logIn(page, "rev.malee", reviewerLanding);
  await openFromReviewQueue(page, claimNo);
  await page.getByRole("button", { name: "Reject" }).click();

  // A reject needs a reason: the field shows the message and the dialog stays open.
  await page.getByRole("button", { name: "Reject" }).last().click();
  await expect(
    page.getByText("Enter a reason (5 to 500 characters).").first()
  ).toBeVisible();

  await page.getByLabel("Reason").fill("Receipt missing");
  await page.getByRole("button", { name: "Reject" }).last().click();
  await expect(page.getByText("Claim rejected.").first()).toBeVisible();

  await logOut(page);
  await logIn(page, "req.anan");
  await page.getByRole("link", { name: claimNo }).first().click();

  await expect(
    page.getByText("Rejected", { exact: true }).first()
  ).toBeVisible();

  const history = page.getByRole("region", { name: "History" });

  await expect(history.getByText("Submitted → Rejected")).toBeVisible();
  await expect(history.getByText("Receipt missing")).toBeVisible();
});

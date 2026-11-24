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

  const dialog = page.getByRole("dialog");
  const reject = dialog.getByRole("button", { name: "Reject" });
  const reason = dialog.getByRole("textbox", { name: "Reason" });
  await expect(reject).toBeDisabled();

  await reason.fill("No photos");
  await expect(reject).toBeDisabled();
  await expect(
    dialog.getByText("Reason must be at least 10 characters.")
  ).toBeVisible();

  await reason.fill("Receipt missing");
  await expect(reject).toBeEnabled();
  await reject.click();
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

import { expect, test } from "@playwright/test";

import { logIn, logOut, openFromReviewQueue, submitClaim } from "./support";

const reviewerLanding = /\/reviews$/u;

test("F06-TEST-15: req.anan submits, rev.malee returns with a reason, req.anan edits and resubmits, rev.malee approves, and the history reads Submit, Return, Resubmit, Approve", async ({
  page,
}) => {
  const reason = "Please attach the taxi receipt";
  const edited = `Returned journey edited ${Date.now()}`;

  await logIn(page, "req.anan");
  const claimNo = await submitClaim(page, `Returned journey ${Date.now()}`);

  await logOut(page);
  await logIn(page, "rev.malee", reviewerLanding);
  await openFromReviewQueue(page, claimNo);
  await page.getByRole("button", { name: "Return" }).click();
  await page.getByLabel("Reason for return").fill(reason);
  await page.getByRole("button", { name: "Return" }).last().click();
  await expect(page.getByText("Claim returned.").first()).toBeVisible();

  await logOut(page);
  await logIn(page, "req.anan");
  await page.getByRole("link", { name: claimNo }).first().click();
  await expect(page.getByText(reason).first()).toBeVisible();
  await expect(page.getByText("Returned 1 / 3")).toBeVisible();

  await page.getByRole("link", { name: "Edit", exact: true }).click();
  await page.getByLabel("Title").fill(edited);
  await page.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByText(edited, { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Resubmit" }).click();
  await expect(page.getByText("Claim resubmitted.").first()).toBeVisible();

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

  await expect(history.getByRole("listitem")).toHaveCount(4);
  await expect(history.getByRole("listitem").nth(0)).toContainText(
    "– → Submitted"
  );
  await expect(history.getByRole("listitem").nth(1)).toContainText(
    "Submitted → Returned"
  );
  await expect(history.getByRole("listitem").nth(2)).toContainText(
    "Returned → Submitted"
  );
  await expect(history.getByRole("listitem").nth(3)).toContainText(
    "Submitted → Approved"
  );
});

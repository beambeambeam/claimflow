import { expect, test } from "@playwright/test";

import { fillItem, logIn, openNewClaim } from "./support";

test("F02-TEST-21: req.anan submits a claim with 2 line items and sees the claim number and Submitted", async ({
  page,
}) => {
  await logIn(page, "req.anan");
  await openNewClaim(page);

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

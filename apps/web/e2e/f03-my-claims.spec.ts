import { expect, test } from "@playwright/test";

import { fillItem, logIn, openNewClaim } from "./support";

test("F03-TEST-12: req.anan submits a claim, finds it first in My Claims, and sees its items and the SUBMIT history entry", async ({
  page,
}) => {
  const title = `Bangkok workshop travel ${Date.now()}`;

  await logIn(page, "req.anan");
  await openNewClaim(page);

  await page.getByLabel("Title").fill(title);
  await page
    .getByLabel("Purpose")
    .fill("Train fares and hotel for the workshop");
  await fillItem(page, 0, { amount: "250.00", description: "Train return" });
  await page.getByRole("button", { name: "Add line item" }).click();
  await fillItem(page, 1, {
    amount: "1000.00",
    description: "Hotel, one night",
  });
  await page.getByRole("button", { name: "Submit" }).click();

  const confirmation = page.getByRole("heading", {
    name: /^Claim CLM-\d{4}-\d{6} submitted$/u,
  });

  await expect(confirmation).toBeVisible();

  const claimNo = /CLM-\d{4}-\d{6}/u.exec(
    (await confirmation.textContent()) ?? ""
  )?.[0];

  expect(claimNo).toBeDefined();

  // The new claim is the first row of My Claims.
  await page.getByRole("link", { name: "View my claims" }).click();
  await expect(page).toHaveURL(/\/claims$/u);
  const firstRow = page.getByRole("row").nth(1);
  await expect(firstRow).toContainText(claimNo ?? "");
  await expect(firstRow).toContainText(title);
  await expect(firstRow).toContainText("฿1,250.00");
  await expect(firstRow).toContainText("Submitted");

  // Its detail shows the line items and the SUBMIT history entry.
  await firstRow.getByRole("link", { name: claimNo ?? "" }).click();
  await expect(page).toHaveURL(/\/claims\/[\w-]+$/u);
  await expect(
    page.getByRole("heading", { name: claimNo ?? "", level: 1 })
  ).toBeVisible();

  const items = page.getByRole("region", { name: "Line items" });
  await expect(items.getByRole("row")).toHaveCount(3);
  await expect(items).toContainText("Train return");
  await expect(items).toContainText("฿250.00");
  await expect(items).toContainText("Hotel, one night");
  await expect(items).toContainText("฿1,000.00");

  const history = page.getByRole("region", { name: "History" });
  await expect(history.getByRole("listitem")).toHaveCount(1);
  await expect(history).toContainText("Submitted");
  await expect(history).toContainText("Anan K.");
});

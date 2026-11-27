import { expect, test } from "@playwright/test";

import { logIn } from "./support";

const reviewerLanding = /\/reviews$/u;

// Needs the load data of F-10/02 (`bun run seed:load`), which has decisions in every month from July to November 2026.
test("F10-TEST-13: rev.malee opens Summary, picks October 2026, and sees the tiles, the totals table, and the average for her assigned claims", async ({
  page,
}) => {
  await logIn(page, "rev.malee", reviewerLanding);
  await page.getByRole("link", { name: "Summary" }).click();
  await expect(page).toHaveURL(/\/reports$/u);
  await expect(page.getByRole("heading", { name: "Summary" })).toBeVisible();

  await page.getByLabel("From").fill("2026-10-01");
  await page.getByLabel("To").fill("2026-10-31");
  await expect(page).toHaveURL(/from=2026-10-01/u);
  await expect(page).toHaveURL(/to=2026-10-31/u);

  // The six status tiles show the current claims, whatever the range.
  const statuses = [
    "Submitted",
    "Returned",
    "Pending finance",
    "Approved",
    "Rejected",
    "Withdrawn",
  ];

  await Promise.all(
    statuses.map(async (status) =>
      expect(
        page.getByRole("heading", {
          name: new RegExp(`^${status}: \\d+$`, "u"),
        })
      ).toBeVisible()
    )
  );

  await expect(
    page.getByText("Current claims, not limited by the date range.")
  ).toBeVisible();

  // The approved totals of October, and the average decision time of the claims decided then.
  await expect(page.getByRole("cell", { name: "Oct 2026" })).toBeVisible();
  await expect(
    page.getByRole("cell", { name: /^฿[\d,]+\.\d{2}$/u }).first()
  ).toBeVisible();
  await expect(
    page.getByText(/^Average decision time: \d+\.\d hours$/u)
  ).toBeVisible();
});

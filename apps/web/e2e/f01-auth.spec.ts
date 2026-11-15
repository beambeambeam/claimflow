import { expect, test } from "@playwright/test";

// The same value `api seed` hashes. Set it in the environment for the test run. It is never committed.
const password = process.env.SEED_USER_PASSWORD ?? "";

if (password === "") {
  throw new Error(
    "SEED_USER_PASSWORD must be set to the value used by `api seed`"
  );
}

test("F01-TEST-24: anonymous claims visit redirects to login, req.anan logs in and out, then claims redirects to login again", async ({
  page,
}) => {
  await page.goto("/claims");
  await expect(page).toHaveURL(/\/login\?redirect=%2Fclaims$/u);

  await page.getByLabel("Username").fill("req.anan");
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Log in" }).click();

  await expect(page).toHaveURL(/\/claims$/u);
  await expect(page.getByRole("heading", { name: "My Claims" })).toBeVisible();
  await expect(page.getByText("Anan K.", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Log out" }).click();
  await expect(page).toHaveURL(/\/login$/u);
  await expect(page.getByRole("button", { name: "Log in" })).toBeVisible();

  await page.goto("/claims");
  await expect(page).toHaveURL(/\/login\?redirect=%2Fclaims$/u);
});

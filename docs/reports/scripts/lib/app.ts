import { mkdirSync } from "node:fs";
import path from "node:path";

import type { Page } from "@playwright/test";

// What a sprint's demo.ts gets: a page on the running local stack and the
// steps every demo needs. The password is typed into the masked field only,
// never written to a file or a screenshot.
export interface App {
  page: Page;
  base: string;
  login: (username: string) => Promise<void>;
  // Ends the session, so the same page can log in as someone else.
  logout: () => Promise<void>;
  shot: (name: string) => Promise<void>;
  // A date n days from today in Bangkok time, as YYYY-MM-DD.
  day: (n: number) => string;
}

export type Demo = (app: App) => Promise<void>;

export const app = (
  page: Page,
  base: string,
  password: string,
  out: string
): App => {
  mkdirSync(out, { recursive: true });

  return {
    base,
    day: (n) => {
      const d = new Date(Date.now() + 7 * 3600e3);
      d.setUTCDate(d.getUTCDate() + n);

      return d.toISOString().slice(0, 10);
    },
    login: async (username) => {
      await page.goto(`${base}/login`);
      await page.getByLabel("Username").fill(username);
      await page.getByLabel("Password").fill(password);
      await page.getByRole("button", { name: "Log in" }).click();
      await page.waitForURL((u) => !u.pathname.startsWith("/login"));
    },
    logout: async () => {
      await page.context().clearCookies();
    },
    page,
    shot: async (name) => {
      await page.screenshot({ fullPage: true, path: path.join(out, name) });
    },
  };
};

import { chromium } from "@playwright/test";
import type { Browser, ElementHandle, Page } from "@playwright/test";

// The installed Chrome, else Edge: neither needs a Playwright browser download,
// and one of them is on every Windows and macOS machine.
export const launch = async (): Promise<Browser> => {
  for (const channel of ["chrome", "msedge"]) {
    try {
      return await chromium.launch({ channel, headless: true });
    } catch {
      // Try the next channel.
    }
  }

  throw new Error("neither Chrome nor Edge could be launched");
};

export interface PageBox {
  x: number;
  y: number;
  width: number;
  height: number;
}

// An element's box in page coordinates (the viewport box plus the scroll).
export const pageBox = async (
  page: Page,
  element: ElementHandle
): Promise<PageBox | null> => {
  const box = await element.boundingBox();

  if (!box) {
    return null;
  }

  const [sx, sy] = await page.evaluate(() => [window.scrollX, window.scrollY]);

  return { ...box, x: box.x + (sx ?? 0), y: box.y + (sy ?? 0) };
};

export const pad = (text: string | number, width: number) =>
  String(text).padEnd(width);

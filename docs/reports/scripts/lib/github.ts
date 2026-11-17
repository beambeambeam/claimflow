import { setTimeout as sleep } from "node:timers/promises";

import type { Page } from "@playwright/test";
import { z } from "zod";

import { groups } from "./match.ts";

// The public repository the report captures. No script logs in.
export const SLUG = "beambeambeam/claimflow";
export const REPO = `https://github.com/${SLUG}`;

// The breadcrumb carries the file path, the box below carries the content, so
// the capture proves which file is on screen.
export const FILE_VIEW = ['[data-testid="breadcrumbs"]', "div[data-hpc]"];

const retry = async <T>(fn: () => Promise<T>, tries = 5): Promise<T> => {
  for (let attempt = 1; ; attempt += 1) {
    try {
      return await fn();
    } catch (error) {
      if (attempt >= tries) {
        throw error;
      }

      await sleep(3000 * attempt);
    }
  }
};

const get = async (url: string) => {
  const res = await fetch(url, { signal: AbortSignal.timeout(30_000) });

  if (!res.ok) {
    throw new Error(`${res.status} ${url}`);
  }

  return res;
};

// The raw markdown at the ref, to count what the page should show.
export const sourceAt = async (ref: string, file: string) => {
  const text = await retry(async () => {
    const res = await get(
      `https://raw.githubusercontent.com/${SLUG}/${ref}/${file}`
    );

    return await res.text();
  });

  return text;
};

export const mermaidCount = async (ref: string, file: string) => {
  const source = await sourceAt(ref, file);

  return source.split("```mermaid").length - 1;
};

const Commit = z.object({ sha: z.string() });

// The commit the ref points at, via the public API.
export const commitAt = async (ref: string) => {
  try {
    return await retry(async () => {
      const res = await get(
        `https://api.github.com/repos/${SLUG}/commits/${ref}`
      );

      const body = Commit.parse(await res.json());

      return body.sha.slice(0, 7);
    });
  } catch {
    return "unknown";
  }
};

export const openFile = async (page: Page, ref: string, file: string) => {
  const url = `${REPO}/blob/${ref}/${file}`;
  await page.goto(url, { timeout: 60_000, waitUntil: "domcontentloaded" });

  for (const sel of FILE_VIEW) {
    await page.waitForSelector(sel, { timeout: 30_000 });
  }

  await page.waitForTimeout(1500);

  return url;
};

export const MERMAID_BLOCKS =
  'section[data-type="mermaid"], section.js-render-needs-enrichment';

// Hide the zoom, pan and copy buttons GitHub overlays on each diagram.
export const hideDiagramControls = async (page: Page) => {
  await page.addStyleTag({
    content: ".js-render-block-actions {display:none !important}",
  });

  for (const frame of page.frames()) {
    if (frame.url().includes("viewscreen.githubusercontent.com")) {
      try {
        await frame.addStyleTag({
          content: "button, [role=button], .btn {display:none !important}",
        });
      } catch {
        // A frame GitHub unloaded has nothing to hide.
      }
    }
  }
};

// Blocks on the page, and blocks whose iframe has drawn (has a real height).
const mermaidDrawn = async (page: Page) =>
  await page.evaluate(
    () =>
      [
        ...document.querySelectorAll(
          '[data-type="mermaid"] iframe, .js-render-enrichment-target iframe'
        ),
      ].filter((f) => f.getBoundingClientRect().height > 60).length
  );

// Messages of diagrams GitHub could not draw. A parse error leaves ink on the
// page (the error and the source), so a pixel test alone calls it drawn.
const diagramErrors = async (page: Page) => {
  const texts = await page.evaluate(() =>
    [
      ...document.querySelectorAll<HTMLElement>(
        ".js-render-enrichment-fallback .flash-error"
      ),
    ].map((e) => e.textContent)
  );

  return texts
    .filter((text) => text.includes("Unable to render"))
    .map((text) => {
      const { line } = groups(/Parse error on line (?<line>\d+)/u.exec(text));

      return line === undefined
        ? "unable to render"
        : `parse error on diagram line ${line}`;
    });
};

// Scroll each diagram into view until `expected` have drawn. GitHub draws each
// one in an iframe after the page loads, and only once it is in view.
export interface Mermaid {
  drawn: number;
  errors: string[];
}

export const waitForMermaid = async (
  page: Page,
  expected: number
): Promise<Mermaid> => {
  if (expected === 0) {
    return { drawn: 0, errors: [] };
  }

  let drawn = 0;

  for (let i = 0; i < 40 && drawn < expected; i += 1) {
    await page.evaluate(() => {
      for (const f of document.querySelectorAll(
        '[data-type="mermaid"], .js-render-enrichment-target'
      )) {
        f.scrollIntoView({ block: "center" });
      }
    });
    await page.waitForTimeout(700);
    drawn = await mermaidDrawn(page);
  }

  // Ask while the diagrams are in view: out of view GitHub may unload them.
  const errors = await diagramErrors(page);
  await page.evaluate(() => {
    window.scrollTo(0, 0);
  });
  await page.waitForTimeout(400);

  return { drawn, errors };
};

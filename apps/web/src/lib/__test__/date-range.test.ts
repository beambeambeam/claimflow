import { describe, expect, it } from "vite-plus/test";

import { isBackwardsRange, rangeMessage } from "#/lib/date-range";

describe("rangeMessage", () => {
  it("is the F-09 message", () => {
    expect(rangeMessage).toBe(
      "The end date must be on or after the start date."
    );
  });
});

describe("isBackwardsRange", () => {
  it.each([
    ["2026-11-10", "2026-11-01", true],
    ["2026-12-01", "2026-11-30", true],
    ["2026-11-05", "2026-11-05", false],
    ["2026-11-01", "2026-11-10", false],
    ["2026-11-10", undefined, false],
    [undefined, "2026-11-01", false],
    [undefined, undefined, false],
  ])("from %s and to %s is backwards: %s", (from, to, backwards) => {
    expect(isBackwardsRange(from, to)).toBe(backwards);
  });
});

import { describe, expect, it } from "vite-plus/test";

import { formatBaht, parseSatang } from "#/lib/format";

describe("parseSatang", () => {
  it.each([
    ["0.01", 1],
    ["250.00", 25_000],
    ["20000.00", 2_000_000],
    ["999999.99", 99_999_999],
  ])("reads %s as %i satang", (text, satang) => {
    expect(parseSatang(text)).toBe(satang);
  });

  it.each(["", "1", "1.5", "1.005", "-1.00", "abc", "1,250.00", "1234567.00"])(
    "rejects %j",
    (text) => {
      expect(parseSatang(text)).toBeNull();
    }
  );
});

describe("formatBaht", () => {
  it.each([
    [0, "฿0.00"],
    [1, "฿0.01"],
    [125_000, "฿1,250.00"],
    [5_000_001, "฿50,000.01"],
  ])("formats %i satang as %s", (satang, text) => {
    expect(formatBaht(satang)).toBe(text);
  });
});

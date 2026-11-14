import { describe, expect, it } from "vite-plus/test";

import {
  formatBaht,
  formatDate,
  formatDateTime,
  formatMoney,
  parseSatang,
} from "#/lib/format";

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

describe("formatMoney", () => {
  it.each([
    ["0.00", "฿0.00"],
    ["1250.00", "฿1,250.00"],
    ["999999.99", "฿999,999.99"],
    ["1234567.50", "฿1,234,567.50"],
  ])("formats the API amount %s as %s", (money, text) => {
    expect(formatMoney(money)).toBe(text);
  });

  it("shows a value that is not an amount as received, not as ฿0.00", () => {
    expect(formatMoney("abc")).toBe("฿abc");
  });
});

describe("formatDate", () => {
  it.each([
    ["2026-11-05T03:00:00Z", "5 Nov 2026"],
    // 20:00 UTC is already the next day in Asia/Bangkok.
    ["2026-11-05T20:00:00Z", "6 Nov 2026"],
    ["2026-12-31T17:30:00Z", "1 Jan 2027"],
    ["2026-11-05", "5 Nov 2026"],
  ])("formats %s as %s in Asia/Bangkok", (value, text) => {
    expect(formatDate(value)).toBe(text);
  });
});

describe("formatDateTime", () => {
  it.each([
    ["2026-11-05T03:00:00Z", "5 Nov 2026, 10:00"],
    ["2026-11-05T20:05:00Z", "6 Nov 2026, 03:05"],
  ])("formats %s as %s in Asia/Bangkok", (value, text) => {
    expect(formatDateTime(value)).toBe(text);
  });
});

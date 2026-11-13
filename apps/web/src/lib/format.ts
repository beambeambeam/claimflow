// Money and date formatting (SDS §7.2).

const amountPattern = /^\d{1,6}\.\d{2}$/u;

const satangPerBaht = 100;

const bahtFormat = new Intl.NumberFormat("en-US", {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

// Reads an amount typed as `^\d{1,6}\.\d{2}$` into satang, or null when the text is not in that format.
export const parseSatang = (text: string): number | null =>
  amountPattern.test(text) ? Number(text.replace(".", "")) : null;

// Formats satang as baht with a symbol and 2 decimals, such as `฿1,250.00`.
export const formatBaht = (satang: number): string =>
  `฿${bahtFormat.format(satang / satangPerBaht)}`;

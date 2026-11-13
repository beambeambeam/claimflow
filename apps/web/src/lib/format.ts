// Money and date formatting (SDS §7.2).

// Reads an amount typed as `^\d{1,6}\.\d{2}$` into satang, or null when the text is not in that format.
export const parseSatang = (_text: string): number | null => null;

// Formats satang as baht with a symbol and 2 decimals, such as `฿1,250.00`.
export const formatBaht = (_satang: number): string => "";

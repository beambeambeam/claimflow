// Money and date formatting (SDS §7.2).

export const amountPattern = /^\d{1,6}\.\d{2}$/u;

const satangPerBaht = 100;

const bahtFormat = new Intl.NumberFormat("en-US", {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

// Any API money value, not only a line-item amount: a claim total can have more than 6 integer digits.
const moneyPattern = /^\d+\.\d{2}$/u;

const bangkok = "Asia/Bangkok";

const dateFormat = new Intl.DateTimeFormat("en-GB", {
  day: "numeric",
  month: "short",
  year: "numeric",
  timeZone: bangkok,
});

const dateTimeFormat = new Intl.DateTimeFormat("en-GB", {
  day: "numeric",
  month: "short",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hourCycle: "h23",
  timeZone: bangkok,
});

// Reads an amount typed as `^\d{1,6}\.\d{2}$` into satang, or null when the text is not in that format.
export const parseSatang = (text: string): number | null =>
  amountPattern.test(text) ? Number(text.replace(".", "")) : null;

// Formats satang as baht with a symbol and 2 decimals, such as `฿1,250.00`.
export const formatBaht = (satang: number): string =>
  `฿${bahtFormat.format(satang / satangPerBaht)}`;

// Formats an API Money string such as `1250.00` as `฿1,250.00`.
// A value that is not an amount breaks the contract, so it is shown as received, not as zero.
export const formatMoney = (money: string): string =>
  moneyPattern.test(money)
    ? formatBaht(Number(money.replace(".", "")))
    : `฿${money}`;

// Formats a timestamp, or a `YYYY-MM-DD` date, in the Asia/Bangkok calendar, such as `5 Nov 2026`.
// A date-only value is midnight UTC, which is 07:00 on the same day in Bangkok.
export const formatDate = (value: string): string =>
  dateFormat.format(new Date(value));

// Formats a timestamp in Asia/Bangkok with the time, such as `5 Nov 2026, 10:00`.
export const formatDateTime = (value: string): string =>
  dateTimeFormat.format(new Date(value));

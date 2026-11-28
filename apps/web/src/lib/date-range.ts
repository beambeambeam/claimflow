// The From and To dates of a range filter (F-09 §1.3, F-10 §2.1).

// The message of the 400 for a range that ends before it starts, so the web shows what the API would say.
export const rangeMessage = "The end date must be on or after the start date.";

// Whether From is after To. ISO dates compare in calendar order as text. The API would reject such a range, so a page
// never sends it.
export const isBackwardsRange = (
  from: string | undefined,
  to: string | undefined
): boolean => from !== undefined && to !== undefined && from > to;

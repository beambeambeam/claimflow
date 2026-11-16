// The named groups of a regex match; a group that did not take part is undefined.
export const groups = (
  match: RegExpExecArray | RegExpMatchArray | null
): Partial<Record<string, string>> => match?.groups ?? {};

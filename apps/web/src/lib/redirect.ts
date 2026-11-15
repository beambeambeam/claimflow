// Accept same-origin paths only. Browsers read `/\` as `//`, so both prefixes are rejected.
export const isSafeRedirect = (target: string | undefined): target is string =>
  target !== undefined &&
  target.startsWith("/") &&
  !target.startsWith("//") &&
  !target.startsWith("/\\");

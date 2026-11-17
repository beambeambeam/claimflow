// Escape plain text for LaTeX.
export const tex = (text: string) =>
  text
    .replaceAll("\\", String.raw`\textbackslash{}`)
    .replaceAll(/[&%_#${}]/gu, (c) => `\\${c}`)
    .replaceAll("~", String.raw`\~{}`)
    .replaceAll("^", String.raw`\^{}`)
    .replaceAll("›", ">");

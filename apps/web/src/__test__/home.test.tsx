import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vite-plus/test";

import { HomePage } from "#/components/home-page";

describe("home page", () => {
  it("renders the welcome heading and edit hint", () => {
    const markup = renderToStaticMarkup(<HomePage />);

    expect(markup).toContain("Welcome to TanStack Start");
    expect(markup).toContain("src/components/home-page.tsx");
  });
});

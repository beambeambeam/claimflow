import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, it } from "vite-plus/test";

import { DateText } from "#/components/date-text";

afterEach(cleanup);

describe("DateText", () => {
  it("shows a date in Asia/Bangkok", () => {
    render(<DateText value="2026-11-05T20:00:00Z" />);

    screen.getByText("6 Nov 2026");
  });

  it("shows a date and time in Asia/Bangkok", () => {
    render(<DateText value="2026-11-05T03:00:00Z" withTime />);

    screen.getByText("5 Nov 2026, 10:00");
  });

  it("shows a dash when there is no date", () => {
    render(<DateText value={null} />);

    screen.getByText("–");
  });
});

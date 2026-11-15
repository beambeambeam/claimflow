import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, it } from "vite-plus/test";

import { Money } from "#/components/money";

afterEach(cleanup);

describe("Money", () => {
  it("shows the API amount in baht with a separator and 2 decimals", () => {
    render(<Money value="1250.00" />);

    screen.getByText("฿1,250.00");
  });

  it("shows a value that is not an amount as received", () => {
    render(<Money value="abc" />);

    screen.getByText("฿abc");
  });
});

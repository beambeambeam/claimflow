import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { delay, http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import {
  afterAll,
  afterEach,
  beforeAll,
  describe,
  expect,
  it,
} from "vite-plus/test";

import { ClaimForm } from "#/features/claims/components/claim-form";
import { client } from "#/lib/api/client.gen";
import type { Claim, ClaimInput } from "#/lib/api/types.gen";
import { zClaimInput } from "#/lib/api/zod.gen";

const claimsUrl = "http://localhost/api/v1/claims";

const submittedClaim: Claim = {
  id: "0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11",
  claimNo: "CLM-2026-000123",
  status: "SUBMITTED",
  title: "Bangkok workshop travel",
  purpose: "Train fares and hotel",
  total: "1250.00",
  returnCount: 0,
  version: 0,
  requester: { id: "usr-1", name: "Anan K.", position: "Staff, Section A" },
  assignedReviewer: {
    id: "usr-2",
    name: "Malee S.",
    position: "Section Head A",
  },
  firstSubmittedAt: "2026-11-05T03:00:00Z",
  submittedAt: "2026-11-05T03:00:00Z",
  decidedAt: null,
  createdAt: "2026-11-05T03:00:00Z",
  updatedAt: "2026-11-05T03:00:00Z",
  items: [],
  history: [],
  allowedActions: [],
};

let bodies: ClaimInput[] = [];

const server = setupServer(
  http.post(claimsUrl, async ({ request }) => {
    bodies.push(zClaimInput.parse(await request.json()));

    return HttpResponse.json(
      { data: submittedClaim },
      { status: 201, headers: { ETag: '"0"' } }
    );
  })
);

beforeAll(() => {
  // Node fetch needs an absolute URL; the browser resolves `/api/v1` against the page.
  client.setConfig({ baseUrl: "http://localhost/api/v1" });
  server.listen({ onUnhandledFrame: "error" });
});

afterEach(() => {
  cleanup();
  server.resetHandlers();
  bodies = [];
});

afterAll(() => {
  server.close();
});

const renderForm = () => {
  render(
    <QueryClientProvider client={new QueryClient()}>
      <ClaimForm />
    </QueryClientProvider>
  );

  return userEvent.setup();
};

type User = ReturnType<typeof userEvent.setup>;

// Fills the `index`-th line item row (0 = first) with a Travel item.
const fillItem = async (
  user: User,
  index: number,
  { amount, description }: { amount: string; description: string }
) => {
  await user.click(screen.getAllByLabelText("Category")[index]);
  await user.click(await screen.findByRole("option", { name: "Travel" }));
  await user.type(screen.getAllByLabelText("Description")[index], description);
  await user.type(
    screen.getAllByLabelText("Expense date")[index],
    "2026-11-03"
  );
  await user.type(screen.getAllByLabelText("Amount")[index], amount);
  await user.type(
    screen.getAllByLabelText("Receipt reference")[index],
    "SRT-1"
  );
};

const getButton = (name: string) =>
  screen.getByRole<HTMLButtonElement>("button", { name });

// Clicks Add line item `count` times, each click after the one before it.
const addRows = async (user: User, count: number): Promise<void> => {
  if (count === 0) {
    return;
  }

  await user.click(getButton("Add line item"));
  await addRows(user, count - 1);
};

describe("F02-TEST-19 (F02-AC-13): New Claim form", () => {
  it("starts with one empty row, a ฿0.00 total, and Remove disabled", () => {
    renderForm();

    expect(screen.getAllByLabelText("Description")).toHaveLength(1);
    screen.getByText("Total ฿0.00");
    expect(getButton("Remove").getAttribute("aria-disabled")).toBe("true");
  });

  it("adds rows up to 20, then disables Add line item; Remove brings it back", async () => {
    const user = renderForm();

    await addRows(user, 19);

    expect(screen.getAllByLabelText("Description")).toHaveLength(20);
    expect(getButton("Add line item").getAttribute("aria-disabled")).toBe(
      "true"
    );

    await user.click(screen.getAllByRole("button", { name: "Remove" })[0]);

    expect(screen.getAllByLabelText("Description")).toHaveLength(19);
    expect(getButton("Add line item").getAttribute("aria-disabled")).not.toBe(
      "true"
    );
  }, 30_000);

  it("shows the running total from the amounts in the right format only", async () => {
    const user = renderForm();

    await user.click(getButton("Add line item"));
    await user.type(screen.getAllByLabelText("Amount")[0], "250.00");
    await user.type(screen.getAllByLabelText("Amount")[1], "1000.00");
    screen.getByText("Total ฿1,250.00");

    await user.type(screen.getAllByLabelText("Amount")[1], "5");
    screen.getByText("Total ฿250.00");
  });

  it("shows a message under each empty field and sends no request", async () => {
    const user = renderForm();

    await user.click(getButton("Submit"));

    await screen.findAllByText("Enter a title (up to 120 characters).");
    screen.getAllByText("Enter a purpose (up to 1,000 characters).");
    screen.getAllByText("Choose a category.");
    screen.getAllByText("Enter an amount with 2 decimals, e.g. 1250.00.");
    expect(bodies).toEqual([]);
  });

  it("asks for the date again after it is cleared", async () => {
    const user = renderForm();

    await user.type(screen.getByLabelText("Expense date"), "2026-11-03");
    await user.clear(screen.getByLabelText("Expense date"));
    await user.click(getButton("Submit"));

    await screen.findAllByText("Enter a date as YYYY-MM-DD.");
    expect(bodies).toEqual([]);
  });

  it("submits the typed claim without a total and shows the claim number and a Submitted badge", async () => {
    const user = renderForm();

    await user.type(screen.getByLabelText("Title"), "Bangkok workshop travel");
    await user.type(screen.getByLabelText("Purpose"), "Train fares and hotel");
    await fillItem(user, 0, { amount: "250.00", description: "Train" });
    await user.click(getButton("Submit"));

    await screen.findByText("Claim CLM-2026-000123 submitted");
    screen.getByText("Submitted");
    screen.getByText("Total ฿1,250.00");
    screen.getByRole("link", { name: "View my claims" });
    expect(bodies).toEqual([
      {
        title: "Bangkok workshop travel",
        purpose: "Train fares and hotel",
        items: [
          {
            category: "TRAVEL",
            description: "Train",
            expenseDate: "2026-11-03",
            amount: "250.00",
            receiptRef: "SRT-1",
          },
        ],
      },
    ]);

    await user.click(getButton("New claim"));
    expect(screen.getByLabelText<HTMLInputElement>("Title").value).toBe("");
  });

  it("disables Submit with a spinner while the request is in flight", async () => {
    server.use(
      http.post(claimsUrl, async () => {
        await delay(300);

        return HttpResponse.json({ data: submittedClaim }, { status: 201 });
      })
    );
    const user = renderForm();

    await user.type(screen.getByLabelText("Title"), "Bangkok workshop travel");
    await user.type(screen.getByLabelText("Purpose"), "Train fares and hotel");
    await fillItem(user, 0, { amount: "250.00", description: "Train" });
    await user.click(getButton("Submit"));

    await waitFor(() => {
      expect(getButton("Submit").getAttribute("aria-busy")).toBe("true");
    });
    await screen.findByText("Claim CLM-2026-000123 submitted");
  });

  it("shows each server error under its field, keeps the values, and re-enables Submit after a 422", async () => {
    server.use(
      http.post(
        claimsUrl,
        () =>
          HttpResponse.json(
            {
              type: "about:blank",
              title: "Some fields are invalid.",
              status: 422,
              code: "VALIDATION_FAILED",
              instance: "/api/v1/claims",
              requestId: "req-claim-422",
              errors: [
                {
                  field: "items[0].expenseDate",
                  rule: "BR-07",
                  message: "Expense date cannot be in the future.",
                },
                {
                  field: "items[1].amount",
                  rule: "BR-12",
                  message: "A Meals item cannot be more than 1,500.00 THB.",
                },
                {
                  field: "total",
                  rule: "BR-06",
                  message: "Claim total cannot be more than 50,000.00 THB.",
                },
              ],
            },
            { status: 422 }
          ),
        { once: true }
      )
    );
    const user = renderForm();

    await user.type(screen.getByLabelText("Title"), "Bangkok workshop travel");
    await user.type(screen.getByLabelText("Purpose"), "Train fares and hotel");
    await fillItem(user, 0, { amount: "250.00", description: "Train" });
    await user.click(getButton("Add line item"));
    await fillItem(user, 1, { amount: "1600.00", description: "Lunch" });
    await user.click(getButton("Submit"));

    await screen.findByText("Expense date cannot be in the future.");
    screen.getByText("A Meals item cannot be more than 1,500.00 THB.");
    screen.getByText("Claim total cannot be more than 50,000.00 THB.");
    expect(screen.queryByText("Something went wrong.")).toBeNull();
    expect(screen.getByLabelText<HTMLInputElement>("Title").value).toBe(
      "Bangkok workshop travel"
    );
    expect(screen.getAllByLabelText<HTMLInputElement>("Amount")[1].value).toBe(
      "1600.00"
    );
    expect(getButton("Submit").getAttribute("aria-busy")).not.toBe("true");

    // The next submit sends the same values again, and the server may accept them now.
    await user.click(getButton("Submit"));
    await screen.findByText("Claim CLM-2026-000123 submitted");
  }, 30_000);

  it("keeps the values and offers Try again with the request ID after a server error", async () => {
    server.use(
      http.post(
        claimsUrl,
        () =>
          HttpResponse.json(
            {
              type: "about:blank",
              title: "Something went wrong.",
              status: 500,
              code: "INTERNAL",
              instance: "/api/v1/claims",
              requestId: "req-claim-500",
            },
            { status: 500 }
          ),
        { once: true }
      )
    );
    const user = renderForm();

    await user.type(screen.getByLabelText("Title"), "Bangkok workshop travel");
    await user.type(screen.getByLabelText("Purpose"), "Train fares and hotel");
    await fillItem(user, 0, { amount: "250.00", description: "Train" });
    await user.click(getButton("Submit"));

    await screen.findByText("Something went wrong.");
    screen.getByText("Request ID: req-claim-500");
    expect(screen.getByLabelText<HTMLInputElement>("Title").value).toBe(
      "Bangkok workshop travel"
    );

    await user.click(getButton("Try again"));
    await screen.findByText("Claim CLM-2026-000123 submitted");
  });

  it("shows the error without a request ID when the network fails", async () => {
    server.use(http.post(claimsUrl, () => HttpResponse.error()));
    const user = renderForm();

    await user.type(screen.getByLabelText("Title"), "Bangkok workshop travel");
    await user.type(screen.getByLabelText("Purpose"), "Train fares and hotel");
    await fillItem(user, 0, { amount: "250.00", description: "Train" });
    await user.click(getButton("Submit"));

    await screen.findByText("Something went wrong.");
    expect(screen.queryByText(/Request ID/u)).toBeNull();
  });
});

import { screen, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";

import { claim, problem, requester } from "./fixtures";

// The first render of a route loads its code chunks, which is slow on a busy machine.
const slow = { timeout: 10_000 };

const returned = claim({
  status: "RETURNED",
  returnCount: 2,
  version: 3,
  decidedAt: null,
  allowedActions: ["edit", "submit", "withdraw"],
  history: [
    {
      id: 2,
      action: "RETURN",
      fromStatus: "SUBMITTED",
      toStatus: "RETURNED",
      actor: { id: "usr-reviewer", name: "Malee R.", position: "Head" },
      comment: "Please attach the receipt.",
      createdAt: "2026-11-06T09:30:00Z",
    },
  ],
});

const path = `/claims/${returned.id}`;
const detailUrl = `${apiUrl}${path}`;

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => HttpResponse.json({ data: requester })),
  http.get(detailUrl, () => HttpResponse.json({ data: returned })),
  http.get(`${apiUrl}/claims`, () =>
    HttpResponse.json({ data: [], page: 1, pageSize: 20, total: 0 })
  )
);

serveApi(server);

describe("Resubmitting a returned claim", { timeout: 20_000 }, () => {
  it("F06-TEST-14 (Edit, Resubmit, toast): the owner of a returned claim sees Edit and Resubmit, not Submit claim, and resubmitting shows the toast and refetches", async () => {
    let current = returned;
    const versions: (string | null)[] = [];
    server.use(
      http.get(detailUrl, () => HttpResponse.json({ data: current })),
      http.post(`${detailUrl}/submit`, ({ request }) => {
        versions.push(request.headers.get("If-Match"));
        current = {
          ...returned,
          status: "SUBMITTED",
          version: 4,
          allowedActions: [],
        };

        return HttpResponse.json({ data: current });
      })
    );

    const { user } = renderAt(path);

    const header = await screen.findByRole("region", { name: "Claim" }, slow);
    within(header).getByRole("link", { name: "Edit" });
    expect(screen.queryByRole("button", { name: "Submit claim" })).toBeNull();

    await user.click(within(header).getByRole("button", { name: "Resubmit" }));

    await screen.findAllByText("Claim resubmitted.");
    await within(header).findByText("Submitted");
    expect(versions).toEqual(['"3"']);
    expect(screen.queryByRole("button", { name: "Resubmit" })).toBeNull();
  });

  it("F06-TEST-14 (422): lists each broken line as Line n and keeps the claim returned", async () => {
    server.use(
      http.post(`${detailUrl}/submit`, () =>
        HttpResponse.json(
          {
            ...problem(422, "VALIDATION_FAILED", "resubmit-invalid"),
            errors: [
              {
                field: "items[1].expenseDate",
                message:
                  "Expense date cannot be more than 90 days before the claim was first submitted.",
                rule: "BR-07",
              },
            ],
          },
          { status: 422 }
        )
      )
    );

    const { user } = renderAt(path);

    await user.click(
      await screen.findByRole("button", { name: "Resubmit" }, slow)
    );

    const alert = await screen.findByText(
      "Line 2: Expense date cannot be more than 90 days before the claim was first submitted."
    );

    expect(
      screen.getByRole("region", { name: "Line items" }).contains(alert)
    ).toBe(true);
    screen.getByRole("button", { name: "Resubmit" });
    expect(screen.queryByText("Claim resubmitted.")).toBeNull();
  });
});

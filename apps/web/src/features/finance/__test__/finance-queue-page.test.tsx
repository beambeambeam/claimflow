import { screen, waitFor, within } from "@testing-library/react";
import { delay, http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { beforeEach, describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";
import type { ClaimPage, CurrentUser } from "#/lib/api/types.gen";

import { financeOfficer, problem, reviewer, summary } from "./fixtures";

// The first render of a route loads its code chunks, which is slow on a busy machine.
const slow = { timeout: 10_000 };

let sessionUser: CurrentUser = financeOfficer;

let listResponse: () => Response = () =>
  HttpResponse.json({ data: [], page: 1, pageSize: 20, total: 0 });

// While set, the queue never answers, as when a request is still in flight.
let stalled = false;

const requestedUrls: URL[] = [];

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => HttpResponse.json({ data: sessionUser })),
  http.get(`${apiUrl}/claims`, async ({ request }) => {
    requestedUrls.push(new URL(request.url));

    if (stalled) {
      await delay("infinite");
    }

    return listResponse();
  })
);

serveApi(server);

beforeEach(() => {
  stalled = false;
  sessionUser = financeOfficer;
  listResponse = () =>
    HttpResponse.json({ data: [], page: 1, pageSize: 20, total: 0 });
  requestedUrls.length = 0;
});

describe("Finance Queue", () => {
  it("F08-TEST-16: lists the pending finance claims in the order the API gave, and opens a claim", async () => {
    listResponse = () =>
      HttpResponse.json({
        data: [
          summary({ id: "claim-1", claimNo: "CLM-2026-000123" }),
          summary({
            id: "claim-2",
            claimNo: "CLM-2026-000124",
            title: "Training course",
            total: "20000.00",
            requester: { id: "usr-ploy", name: "Ploy", position: "Staff" },
            submittedAt: "2026-11-06T20:00:00Z",
          }),
        ],
        page: 1,
        pageSize: 20,
        total: 2,
      } satisfies ClaimPage);
    const { router, user } = renderAt("/finance");

    await screen.findByText("CLM-2026-000124", {}, slow);
    screen.getByRole("heading", { name: "Finance Queue" });
    expect(
      screen.getAllByRole("columnheader").map((header) => header.textContent)
    ).toEqual(["Claim no.", "Requester", "Title", "Total", "Submitted"]);

    // The oldest claim is first: the page shows the API's order and does not sort.
    const [, first, second] = screen.getAllByRole("row");
    within(first).getByText("CLM-2026-000123");
    within(first).getByText("฿15,000.00");
    within(second).getByText("Ploy");
    within(second).getByText("7 Nov 2026");

    expect(requestedUrls[0].searchParams.get("scope")).toBe("finance");

    await user.click(
      within(second).getByRole("link", { name: "CLM-2026-000124" })
    );
    await waitFor(() => {
      expect(router.state.location.pathname).toBe("/claims/claim-2");
    });
  }, 20_000);

  it("F08-TEST-16: shows 5 skeleton rows while the queue loads", async () => {
    stalled = true;
    renderAt("/finance");

    await screen.findByRole("heading", { name: "Finance Queue" }, slow);
    await waitFor(() => {
      expect(screen.getAllByRole("row")).toHaveLength(6);
    });
  }, 20_000);

  it("F08-TEST-16: shows the empty message when no claim waits", async () => {
    renderAt("/finance");

    await screen.findByText(
      "No claims waiting for a final decision.",
      {},
      slow
    );
    expect(screen.queryByRole("table")).toBeNull();
  }, 20_000);

  it('F08-TEST-16: shows "You don\'t have access to this page." to a reviewer on 403', async () => {
    sessionUser = reviewer;
    listResponse = () =>
      HttpResponse.json(problem(403, "FORBIDDEN", "req-finance-403"), {
        status: 403,
      });
    renderAt("/finance");

    await screen.findByText("You don't have access to this page.", {}, slow);
    expect(screen.queryByRole("table")).toBeNull();
  }, 20_000);
});

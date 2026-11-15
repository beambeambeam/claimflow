import { screen, waitFor, within } from "@testing-library/react";
import { delay, http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { beforeEach, describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";
import type { ClaimPage, CurrentUser } from "#/lib/api/types.gen";

import { problem, requester, reviewer, summary } from "./fixtures";

// The first render of a route loads its code chunks, which is slow on a busy machine.
const slow = { timeout: 10_000 };

let sessionUser: CurrentUser = reviewer;

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

const pageOf = (page: number, total: number): ClaimPage => ({
  data: [
    summary({
      id: `claim-page-${page}`,
      claimNo: `CLM-2026-00010${page}`,
      title: `Claim on page ${page}`,
    }),
  ],
  page,
  pageSize: 20,
  total,
});

beforeEach(() => {
  stalled = false;
  sessionUser = reviewer;
  listResponse = () =>
    HttpResponse.json({ data: [], page: 1, pageSize: 20, total: 0 });
  requestedUrls.length = 0;
});

describe("Review Queue", () => {
  it("F04-TEST-20: lists the columns in the order the API gave, with the requester, and opens a claim", async () => {
    listResponse = () =>
      HttpResponse.json({
        data: [
          summary({
            id: "claim-1",
            claimNo: "CLM-2026-000123",
            title: "Bangkok workshop travel",
            submittedAt: "2026-11-05T03:00:00Z",
          }),
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
    const { router, user } = renderAt("/reviews");

    await screen.findByText("CLM-2026-000124", {}, slow);
    expect(
      screen.getAllByRole("columnheader").map((header) => header.textContent)
    ).toEqual(["Claim no.", "Requester", "Title", "Total", "Submitted"]);

    // The oldest claim is first: the page shows the API's order and does not sort.
    const [, first, second] = screen.getAllByRole("row");
    within(first).getByText("CLM-2026-000123");
    within(first).getByText("Anan K.");
    within(first).getByText("฿1,250.00");
    within(first).getByText("5 Nov 2026");
    within(second).getByText("Ploy");
    within(second).getByText("฿20,000.00");
    within(second).getByText("7 Nov 2026");

    expect(requestedUrls[0].searchParams.get("scope")).toBe("review");
    expect(requestedUrls[0].searchParams.get("pageSize")).toBe("20");

    await user.click(
      within(second).getByRole("link", { name: "CLM-2026-000124" })
    );
    await waitFor(() => {
      expect(router.state.location.pathname).toBe("/claims/claim-2");
    });
  }, 20_000);

  it("F04-TEST-20: shows 5 skeleton rows while the queue loads", async () => {
    stalled = true;
    renderAt("/reviews");

    await screen.findByRole("heading", { name: "Review Queue" }, slow);
    await waitFor(() => {
      expect(screen.getAllByRole("row")).toHaveLength(6);
    });
  }, 20_000);

  it("F04-TEST-20: shows the empty message when no claim waits", async () => {
    renderAt("/reviews");

    await screen.findByText("No claims waiting for your review.", {}, slow);
    expect(screen.queryByRole("table")).toBeNull();
  }, 20_000);

  it("F04-TEST-20: shows an inline error with the request ID, and Try again fetches the queue again", async () => {
    listResponse = () =>
      HttpResponse.json(problem(500, "INTERNAL", "req-queue-500"), {
        status: 500,
      });
    const { user } = renderAt("/reviews");

    await screen.findByText("Request ID: req-queue-500", {}, slow);
    expect(screen.queryByRole("table")).toBeNull();

    listResponse = () => HttpResponse.json(pageOf(1, 1));
    await user.click(screen.getByRole("button", { name: "Try again" }));

    await screen.findByText("Claim on page 1");
    expect(screen.queryByText("Request ID: req-queue-500")).toBeNull();
  }, 20_000);

  it('F04-TEST-20: shows "You don\'t have access to this page." to a requester on 403', async () => {
    sessionUser = requester;
    listResponse = () =>
      HttpResponse.json(problem(403, "FORBIDDEN", "req-queue-403"), {
        status: 403,
      });
    renderAt("/reviews");

    await screen.findByText("You don't have access to this page.", {}, slow);
    expect(screen.queryByRole("table")).toBeNull();
  }, 20_000);

  it("F04-TEST-20: pages through the queue, with the page in the URL", async () => {
    listResponse = () => {
      const page = Number(requestedUrls.at(-1)?.searchParams.get("page") ?? 1);

      return HttpResponse.json(pageOf(page, 41));
    };

    const { router, user } = renderAt("/reviews");

    await screen.findByText("Claim on page 1", {}, slow);
    await user.click(screen.getByRole("button", { name: /next/iu }));

    await screen.findByText("Claim on page 2");
    expect(router.state.location.search).toEqual({ page: 2 });
    expect(requestedUrls.at(-1)?.searchParams.get("scope")).toBe("review");
  }, 20_000);
});

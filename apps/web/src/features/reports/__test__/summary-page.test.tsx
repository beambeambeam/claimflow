import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { delay, http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { beforeEach, describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";
import type { CurrentUser, Summary } from "#/lib/api/types.gen";

import {
  noDecisions,
  problem,
  requester,
  reviewer,
  twoMonths,
} from "./fixtures";

// The first render of a route loads its code chunks, which is slow on a busy machine.
const slow = { timeout: 10_000 };

let sessionUser: CurrentUser = reviewer;

let summaryResponse: () => Response = () =>
  HttpResponse.json({ data: twoMonths });

// While set, the summary never answers, as when a request is still in flight.
let stalled = false;

const requestedUrls: URL[] = [];

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => HttpResponse.json({ data: sessionUser })),
  http.get(`${apiUrl}/claims`, () =>
    HttpResponse.json({ data: [], page: 1, pageSize: 20, total: 0 })
  ),
  http.get(`${apiUrl}/reports/summary`, async ({ request }) => {
    requestedUrls.push(new URL(request.url));

    if (stalled) {
      await delay("infinite");
    }

    return summaryResponse();
  })
);

serveApi(server);

beforeEach(() => {
  stalled = false;
  sessionUser = reviewer;
  summaryResponse = () => HttpResponse.json({ data: twoMonths });
  requestedUrls.length = 0;
});

const respondWith = (summary: Summary) => {
  summaryResponse = () => HttpResponse.json({ data: summary });
};

// A tile shows the count as a heading named after its status, such as "Submitted: 4".
const tileCounts = () =>
  screen
    .queryAllByRole("heading", { level: 2 })
    .map((heading) => heading.getAttribute("aria-label"))
    .filter((label) => label !== null);

const cellsOf = (row: HTMLElement) =>
  within(row)
    .getAllByRole("cell")
    .map((cell) => cell.textContent);

describe("Summary", () => {
  it("F10-TEST-08: shows the status tiles, the month by category table, and the average for the range in the URL", async () => {
    renderAt("/reports?from=2026-10-01&to=2026-11-30");

    await screen.findByText("Oct 2026", {}, slow);
    screen.getByRole("heading", { name: "Summary" });

    expect(requestedUrls[0].searchParams.get("from")).toBe("2026-10-01");
    expect(requestedUrls[0].searchParams.get("to")).toBe("2026-11-30");
    expect(screen.getByLabelText<HTMLInputElement>("From").value).toBe(
      "2026-10-01"
    );
    expect(screen.getByLabelText<HTMLInputElement>("To").value).toBe(
      "2026-11-30"
    );

    // Six tiles in lifecycle order, each with its status label and count.
    expect(tileCounts()).toEqual([
      "Submitted: 4",
      "Returned: 1",
      "Pending finance: 2",
      "Approved: 12",
      "Rejected: 3",
      "Withdrawn: 1",
    ]);
    screen.getByText("Current claims, not limited by the date range.");

    // One column per category with an approved total, and the month total last.
    expect(
      screen.getAllByRole("columnheader").map((header) => header.textContent)
    ).toEqual(["Month", "Travel", "Accommodation", "Meals", "Total"]);
    const [, october, november] = screen.getAllByRole("row");
    expect(cellsOf(october)).toEqual([
      "Oct 2026",
      "฿18,250.00",
      "–",
      "฿1,200.00",
      "฿19,450.00",
    ]);
    expect(cellsOf(november)).toEqual([
      "Nov 2026",
      "฿300.50",
      "฿5,000.00",
      "–",
      "฿5,300.50",
    ]);

    screen.getByText("Average decision time: 26.4 hours");
  }, 20_000);

  it("F10-TEST-08: puts a changed From in the URL and asks for the new range", async () => {
    const { router } = renderAt("/reports?from=2026-10-01&to=2026-11-30");
    await screen.findByText("Oct 2026", {}, slow);

    fireEvent.change(screen.getByLabelText("From"), {
      target: { value: "2026-10-15" },
    });

    await waitFor(() => {
      expect(router.state.location.search).toEqual({
        from: "2026-10-15",
        to: "2026-11-30",
      });
    });
    await waitFor(() => {
      expect(requestedUrls.at(-1)?.searchParams.get("from")).toBe("2026-10-15");
    });
  }, 20_000);

  it("F10-TEST-08: puts a changed To in the URL and asks for the new range", async () => {
    const { router } = renderAt("/reports?from=2026-10-01&to=2026-11-30");
    await screen.findByText("Oct 2026", {}, slow);

    fireEvent.change(screen.getByLabelText("To"), {
      target: { value: "2026-10-31" },
    });

    await waitFor(() => {
      expect(router.state.location.search).toEqual({
        from: "2026-10-01",
        to: "2026-10-31",
      });
    });
    await waitFor(() => {
      expect(requestedUrls.at(-1)?.searchParams.get("to")).toBe("2026-10-31");
    });
  }, 20_000);

  it("F10-TEST-08: shows the Summary link with report:read", async () => {
    renderAt("/reports");

    await screen.findByText("Oct 2026", {}, slow);
    screen.getByRole("link", { name: "Summary" });
  }, 20_000);

  it("F10-TEST-08: shows no Summary link to a requester", async () => {
    sessionUser = requester;
    renderAt("/claims");

    await screen.findByRole("link", { name: "My Claims" }, slow);
    expect(screen.queryByRole("link", { name: "Summary" })).toBeNull();
  }, 20_000);

  it("F10-TEST-08 extra: Clear removes both dates from the URL", async () => {
    const { router, user } = renderAt("/reports?from=2026-10-01&to=2026-11-30");
    await screen.findByText("Oct 2026", {}, slow);

    await user.click(screen.getByRole("button", { name: "Clear" }));

    await waitFor(() => {
      expect(router.state.location.search).toEqual({});
    });
    await waitFor(() => {
      expect(requestedUrls.at(-1)?.searchParams.has("from")).toBe(false);
    });
    expect(screen.queryByRole("button", { name: "Clear" })).toBeNull();
  }, 20_000);

  it("F10-TEST-08 extra: does not ask when From is after To", async () => {
    renderAt("/reports?from=2026-11-10&to=2026-11-01");

    await screen.findByText(
      "The end date must be on or after the start date.",
      {},
      slow
    );
    expect(requestedUrls).toEqual([]);
  }, 20_000);

  it("F10-TEST-09: says there is no decision in the range, and shows a dash for the average", async () => {
    respondWith(noDecisions);
    renderAt("/reports?from=2026-12-01");

    await screen.findByText("No decisions in this date range.", {}, slow);
    expect(screen.queryByRole("table")).toBeNull();
    screen.getByText("Average decision time: –");
    // The counts are of current claims, so they stay.
    screen.getByRole("heading", { name: "Approved: 12" });
  }, 20_000);

  it('F10-TEST-09: shows "You don\'t have access to this page." to a requester on 403', async () => {
    sessionUser = requester;
    summaryResponse = () =>
      HttpResponse.json(problem(403, "FORBIDDEN", "req-summary-403"), {
        status: 403,
      });
    renderAt("/reports");

    await screen.findByText("You don't have access to this page.", {}, slow);
    expect(screen.queryByRole("table")).toBeNull();
    expect(tileCounts()).toEqual([]);
    expect(screen.queryByLabelText("From")).toBeNull();
    expect(screen.queryByLabelText("To")).toBeNull();
  }, 20_000);

  it("F10-TEST-09 extra: shows skeleton tiles and rows while the summary loads", async () => {
    stalled = true;
    renderAt("/reports");

    await screen.findByRole("heading", { name: "Summary" }, slow);
    await screen.findByText("Pending finance", {}, slow);
    expect(tileCounts()).toEqual([]);
    expect(screen.getAllByRole("row")).toHaveLength(4);
    expect(screen.queryByText("Oct 2026")).toBeNull();
  }, 20_000);

  it("F10-TEST-09 extra: shows the request ID and Try again after a server error", async () => {
    summaryResponse = () =>
      HttpResponse.json(problem(500, "INTERNAL", "req-summary-500"), {
        status: 500,
      });
    const { user } = renderAt("/reports");

    await screen.findByText("Request ID: req-summary-500", {}, slow);
    summaryResponse = () => HttpResponse.json({ data: twoMonths });
    await user.click(screen.getByRole("button", { name: "Try again" }));

    await screen.findByText("Oct 2026", {}, slow);
  }, 20_000);
});

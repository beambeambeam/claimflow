import { screen, waitFor, within } from "@testing-library/react";
import { delay, http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { beforeEach, describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";
import type { ClaimPage, CurrentUser } from "#/lib/api/types.gen";

import { problem, requester, summary } from "./fixtures";

// The first render of a route loads its code chunks, which is slow on a busy machine.
const slow = { timeout: 10_000 };

const sessionUser: CurrentUser = requester;

const page = (total: number): ClaimPage => ({
  data: total === 0 ? [] : [summary({})],
  page: 1,
  pageSize: 20,
  total,
});

let listResponse: () => Response;

// While set, the list never answers, as when a request is still in flight.
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

// EmptyState also repeats its title in a live region, so the tests find the title by its heading.

const lastParams = () => requestedUrls.at(-1)?.searchParams;

beforeEach(() => {
  stalled = false;
  requestedUrls.length = 0;
  listResponse = () => HttpResponse.json(page(60));
});

describe("All Claims", () => {
  it("F09-TEST-10: fills the controls from the URL and sends the filters", async () => {
    renderAt(
      "/all-claims?status=APPROVED&category=MEALS&from=2026-11-01&to=2026-11-30&q=hotel&page=3"
    );

    await screen.findByText("CLM-2026-000123", {}, slow);
    expect(
      screen.getAllByRole("columnheader").map((header) => header.textContent)
    ).toEqual([
      "Claim no.",
      "Requester",
      "Title",
      "Status",
      "Total",
      "Submitted",
    ]);
    expect(
      screen.getByRole("combobox", { name: "Status" }).textContent
    ).toContain("Approved");
    expect(
      screen.getByRole("combobox", { name: "Category" }).textContent
    ).toContain("Meals");
    expect(screen.getByLabelText<HTMLInputElement>("Search title").value).toBe(
      "hotel"
    );

    expect(Object.fromEntries(lastParams() ?? [])).toEqual({
      scope: "all",
      status: "APPROVED",
      category: "MEALS",
      from: "2026-11-01",
      to: "2026-11-30",
      q: "hotel",
      page: "3",
      pageSize: "20",
    });
  }, 20_000);

  it("F09-TEST-10: drops unknown values in the URL", async () => {
    renderAt("/all-claims?status=PAID&category=FOOD&from=05/11/2026");

    await screen.findByText("CLM-2026-000123", {}, slow);
    expect(Object.fromEntries(lastParams() ?? [])).toEqual({
      scope: "all",
      page: "1",
      pageSize: "20",
    });
    expect(
      screen.getByRole("combobox", { name: "Status" }).textContent
    ).toContain("All");
  }, 20_000);

  it("F09-TEST-10: a category change keeps the other filters and goes back to page 1", async () => {
    const { router, user } = renderAt("/all-claims?status=APPROVED&page=3");

    await screen.findByText("CLM-2026-000123", {}, slow);
    await user.click(screen.getByRole("combobox", { name: "Category" }));
    await user.click(await screen.findByRole("option", { name: "Meals" }));

    await waitFor(() => {
      expect(router.state.location.search).toEqual({
        status: "APPROVED",
        category: "MEALS",
        page: 1,
      });
    });
    await waitFor(() => {
      expect(lastParams()?.get("category")).toBe("MEALS");
    });
    expect(lastParams()?.get("page")).toBe("1");
  }, 20_000);

  it("F09-TEST-10: sends one request with q=taxi after typing stops, not one per key", async () => {
    const { router, user } = renderAt("/all-claims");

    await screen.findByText("CLM-2026-000123", {}, slow);
    await user.type(screen.getByLabelText("Search title"), " taxi");

    await waitFor(() => {
      expect(lastParams()?.get("q")).toBe("taxi");
    });
    expect(
      requestedUrls.flatMap((url) => url.searchParams.getAll("q"))
    ).toEqual(["taxi"]);
    expect(router.state.location.search).toEqual({ q: "taxi", page: 1 });
  }, 20_000);

  it("F09-TEST-10: shows the range error and sends nothing while From is after To", async () => {
    renderAt("/all-claims?from=2026-11-10&to=2026-11-01");

    await screen.findByText(
      "The end date must be on or after the start date.",
      {},
      slow
    );
    expect(requestedUrls).toHaveLength(0);
    expect(screen.queryByRole("table")).toBeNull();
  }, 20_000);

  it("F09-TEST-10: shows a 400 field message under its control", async () => {
    listResponse = () =>
      HttpResponse.json(
        {
          ...problem(400, "BAD_REQUEST", "req-list-400"),
          errors: [
            { field: "status", rule: "enum", message: "Unknown status." },
          ],
        },
        { status: 400 }
      );
    renderAt("/all-claims?status=APPROVED");

    await screen.findByText("Unknown status.", {}, slow);
    expect(screen.queryByText("Something went wrong.")).toBeNull();
  }, 20_000);

  it("F09-TEST-10: shows 5 skeleton rows while the list loads", async () => {
    stalled = true;
    renderAt("/all-claims");

    await screen.findByRole("heading", { name: "All Claims" }, slow);
    await waitFor(() => {
      expect(screen.getAllByRole("row")).toHaveLength(6);
    });
  }, 20_000);

  it("F09-TEST-11: with filters and no results, Clear filters removes every filter", async () => {
    listResponse = () => HttpResponse.json(page(0));

    const { router, user } = renderAt(
      "/all-claims?status=REJECTED&q=taxi&page=2"
    );

    await screen.findByRole(
      "heading",
      { name: "No claims match these filters." },
      slow
    );
    expect(screen.getByLabelText<HTMLInputElement>("Search title").value).toBe(
      "taxi"
    );

    // The second button is the empty state's. The first is in the page heading.
    const [, emptyStateButton] = screen.getAllByRole("button", {
      name: "Clear filters",
    });

    await user.click(emptyStateButton);

    await waitFor(() => {
      expect(router.state.location.search).toEqual({});
    });
    await screen.findByRole("heading", { name: "No claims yet." });
    expect(screen.queryByRole("button", { name: "Clear filters" })).toBeNull();
    expect(screen.getByLabelText<HTMLInputElement>("Search title").value).toBe(
      ""
    );
    expect(
      screen.getByRole("combobox", { name: "Status" }).textContent
    ).toContain("All");
  }, 20_000);

  it("F09-TEST-11: without filters and no claims, shows 'No claims yet.' and no Clear filters", async () => {
    listResponse = () => HttpResponse.json(page(0));
    renderAt("/all-claims");

    await screen.findByRole("heading", { name: "No claims yet." }, slow);
    expect(
      screen.queryByRole("heading", { name: "No claims match these filters." })
    ).toBeNull();
    expect(screen.queryByRole("button", { name: "Clear filters" })).toBeNull();
  }, 20_000);

  it("F09-TEST-10: status and date changes keep the search text and go back to page 1", async () => {
    const { router, user } = renderAt("/all-claims?q=taxi&page=2");

    await screen.findByText("CLM-2026-000123", {}, slow);
    await user.click(screen.getByRole("combobox", { name: "Status" }));
    await user.click(await screen.findByRole("option", { name: "Approved" }));
    await user.type(screen.getByLabelText("From"), "2026-11-01");
    await user.type(screen.getByLabelText("To"), "2026-11-30");

    await waitFor(() => {
      expect(router.state.location.search).toEqual({
        q: "taxi",
        status: "APPROVED",
        from: "2026-11-01",
        to: "2026-11-30",
        page: 1,
      });
    });
  }, 20_000);

  it("F09-TEST-10: search text over 120 characters shows the limit and is not sent", async () => {
    const { router, user } = renderAt("/all-claims");

    await screen.findByText("CLM-2026-000123", {}, slow);
    await user.click(screen.getByLabelText("Search title"));
    await user.paste("a".repeat(121));

    await screen.findByText(
      "Search text must be at most 120 characters.",
      {},
      slow
    );
    // Wait past the search delay, so a send would have happened.
    await delay(500);
    expect(router.state.location.search).toEqual({});
    expect(
      requestedUrls.flatMap((url) => url.searchParams.getAll("q"))
    ).toEqual([]);
  }, 20_000);

  it("F09-TEST-10: a page past the last one is replaced by the last page", async () => {
    const { router } = renderAt("/all-claims?status=APPROVED&page=9");

    await waitFor(() => {
      expect(router.state.location.search).toEqual({
        status: "APPROVED",
        page: 3,
      });
    }, slow);
    await screen.findByText("CLM-2026-000123");
  }, 20_000);

  it("F09-TEST-10: the next page keeps the filters", async () => {
    const { router, user } = renderAt("/all-claims?category=MEALS");

    await screen.findByText("CLM-2026-000123", {}, slow);
    await user.click(screen.getByRole("button", { name: /next/iu }));

    await waitFor(() => {
      expect(router.state.location.search).toEqual({
        category: "MEALS",
        page: 2,
      });
    });
  }, 20_000);

  it("F09-TEST-10: a server error shows Something went wrong, and Try again loads the list", async () => {
    listResponse = () =>
      HttpResponse.json(problem(500, "INTERNAL", "req-list-500"), {
        status: 500,
      });

    const { user } = renderAt("/all-claims");

    await screen.findByText("Something went wrong.", {}, slow);
    listResponse = () => HttpResponse.json(page(60));
    await user.click(screen.getByRole("button", { name: "Try again" }));

    await screen.findByText("CLM-2026-000123");
  }, 20_000);

  it("F09-TEST-10: a row shows the requester and opens Claim Detail", async () => {
    const { router, user } = renderAt("/all-claims");

    const row = await screen.findByRole(
      "row",
      { name: /CLM-2026-000123/u },
      slow
    );

    within(row).getByText("Anan K.");
    await user.click(
      within(row).getByRole("link", { name: "CLM-2026-000123" })
    );
    await waitFor(() => {
      expect(router.state.location.pathname).toBe(
        "/claims/0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11"
      );
    });
  }, 20_000);
});

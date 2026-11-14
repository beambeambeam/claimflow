import { screen, waitFor, within } from "@testing-library/react";
import { delay, http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { beforeEach, describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";
import type { ClaimPage, CurrentUser } from "#/lib/api/types.gen";

import { finance, problem, requester, summary } from "./fixtures";

// The first render of a route loads its code chunks, which is slow on a busy machine.
const slow = { timeout: 10_000 };

let sessionUser: CurrentUser = requester;

let listResponse: () => Response = () =>
  HttpResponse.json({ data: [], page: 1, pageSize: 20, total: 0 });

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

// A live region also announces the page, so read it from the pagination itself.
const pagination = () => screen.getByRole("navigation", { name: "Pagination" });

const pageOf = (page: ClaimPage["page"], total: number): ClaimPage => ({
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
});

describe("My Claims", () => {
  it("F03-TEST-10: lists the columns with baht, Bangkok dates and status badges, and opens a claim", async () => {
    sessionUser = requester;
    requestedUrls.length = 0;
    listResponse = () =>
      HttpResponse.json({
        data: [
          summary({
            id: "claim-2",
            claimNo: "CLM-2026-000124",
            title: "Training course",
            status: "APPROVED",
            total: "20000.00",
            submittedAt: "2026-11-05T20:00:00Z",
          }),
          summary({}),
        ],
        page: 1,
        pageSize: 20,
        total: 2,
      } satisfies ClaimPage);
    const { router, user } = renderAt("/claims");

    await screen.findByText("CLM-2026-000124", {}, slow);
    expect(
      screen.getAllByRole("columnheader").map((header) => header.textContent)
    ).toEqual(["Claim no.", "Title", "Total", "Status", "Submitted"]);

    const [, first, second] = screen.getAllByRole("row");
    within(first).getByText("Training course");
    within(first).getByText("฿20,000.00");
    within(first).getByText("Approved");
    within(first).getByText("6 Nov 2026");
    within(second).getByText("CLM-2026-000123");
    within(second).getByText("฿1,250.00");
    within(second).getByText("Submitted");
    within(second).getByText("5 Nov 2026");

    expect(requestedUrls[0].searchParams.get("scope")).toBe("mine");
    expect(requestedUrls[0].searchParams.get("pageSize")).toBe("20");

    expect(
      screen.getByRole("link", { name: "New claim" }).getAttribute("href")
    ).toBe("/claims/new");

    await user.click(
      within(second).getByRole("link", { name: "CLM-2026-000123" })
    );
    await waitFor(() => {
      expect(router.state.location.pathname).toBe(
        "/claims/0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11"
      );
    });
  }, 20_000);

  it("F03-TEST-10: shows 5 skeleton rows while the list loads", async () => {
    sessionUser = requester;
    stalled = true;
    renderAt("/claims");

    await screen.findByRole("heading", { name: "My Claims" }, slow);
    await waitFor(() => {
      expect(screen.getAllByRole("row")).toHaveLength(6);
    });
    expect(screen.queryByText("No claims yet")).toBeNull();
  }, 20_000);

  it("F03-TEST-10: shows 'No claims yet' with a link to /claims/new when the list is empty", async () => {
    sessionUser = requester;
    listResponse = () =>
      HttpResponse.json({ data: [], page: 1, pageSize: 20, total: 0 });
    renderAt("/claims");

    await screen.findByText("No claims yet", {}, slow);
    expect(
      screen.getByRole("link", { name: "Create claim" }).getAttribute("href")
    ).toBe("/claims/new");
    expect(screen.queryByRole("table")).toBeNull();
  }, 20_000);

  it("F03-TEST-10: shows an inline error with the request ID, and Try again fetches the list again", async () => {
    sessionUser = requester;
    listResponse = () =>
      HttpResponse.json(problem(500, "INTERNAL", "req-list-500"), {
        status: 500,
      });
    const { user } = renderAt("/claims");

    await screen.findByText("Request ID: req-list-500", {}, slow);
    expect(screen.queryByRole("table")).toBeNull();

    listResponse = () => HttpResponse.json(pageOf(1, 1));
    await user.click(screen.getByRole("button", { name: "Try again" }));

    await screen.findByText("Claim on page 1");
    expect(screen.queryByText("Request ID: req-list-500")).toBeNull();
  }, 20_000);

  it("F03-TEST-10: shows the error without a request ID when the network fails", async () => {
    sessionUser = requester;
    listResponse = () => HttpResponse.error();
    renderAt("/claims");

    await screen.findByText("Something went wrong.", {}, slow);
    screen.getByRole("button", { name: "Try again" });
    expect(screen.queryByText(/Request ID/u)).toBeNull();
  }, 20_000);

  it('F03-TEST-10: shows "You don\'t have access to this page." on 403 and no table', async () => {
    sessionUser = finance;
    listResponse = () =>
      HttpResponse.json(problem(403, "FORBIDDEN", "req-list-403"), {
        status: 403,
      });
    renderAt("/claims");

    await screen.findByText("You don't have access to this page.", {}, slow);
    expect(screen.queryByRole("table")).toBeNull();
    expect(screen.queryByRole("link", { name: "New claim" })).toBeNull();
  }, 20_000);

  it("F03-TEST-10: hides New claim from a member without claim:create", async () => {
    sessionUser = { ...requester, permissions: { claim: ["list"] } };
    listResponse = () => HttpResponse.json(pageOf(1, 1));
    renderAt("/claims");

    await screen.findByText("Claim on page 1", {}, slow);
    expect(screen.queryByRole("link", { name: "New claim" })).toBeNull();
  }, 20_000);

  it("F03-TEST-10: pages through 3 pages, with the page in the URL", async () => {
    sessionUser = requester;
    requestedUrls.length = 0;
    listResponse = () => {
      const page = Number(requestedUrls.at(-1)?.searchParams.get("page") ?? 1);

      return HttpResponse.json(pageOf(page, 41));
    };

    const { router, user } = renderAt("/claims");

    await screen.findByText("Claim on page 1", {}, slow);
    within(pagination()).getByText("Page 1 of 3");
    expect(screen.getByRole("button", { name: /previous/iu })).toHaveProperty(
      "ariaDisabled",
      "true"
    );

    await user.click(screen.getByRole("button", { name: /next/iu }));

    await screen.findByText("Claim on page 2");
    within(pagination()).getByText("Page 2 of 3");
    expect(router.state.location.search).toEqual({ page: 2 });
    expect(router.state.location.href).toBe("/claims?page=2");
    expect(requestedUrls.at(-1)?.searchParams.get("page")).toBe("2");

    await user.click(screen.getByRole("button", { name: /previous/iu }));
    await screen.findByText("Claim on page 1");
    expect(router.state.location.search).toEqual({ page: 1 });
  }, 30_000);

  it("F03-TEST-10: opens the page in the URL, and falls back to page 1 for a bad page", async () => {
    sessionUser = requester;
    requestedUrls.length = 0;
    listResponse = () => {
      const page = Number(requestedUrls.at(-1)?.searchParams.get("page") ?? 1);

      return HttpResponse.json(pageOf(page, 41));
    };

    renderAt("/claims?page=3");

    await waitFor(() => {
      within(pagination()).getByText("Page 3 of 3");
    }, slow);
    expect(requestedUrls.at(-1)?.searchParams.get("page")).toBe("3");
  }, 20_000);

  it("F03-TEST-10: reads a page that is not a positive number as page 1", async () => {
    sessionUser = requester;
    requestedUrls.length = 0;
    listResponse = () => HttpResponse.json(pageOf(1, 41));
    renderAt("/claims?page=abc");

    await waitFor(() => {
      within(pagination()).getByText("Page 1 of 3");
    }, slow);
    expect(requestedUrls.at(-1)?.searchParams.get("page")).toBe("1");
  }, 20_000);
});

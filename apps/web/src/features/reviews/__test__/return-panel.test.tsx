import { screen, waitFor, within } from "@testing-library/react";
import { delay, http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { beforeEach, describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";
import { listClaimsOptions } from "#/lib/api/@tanstack/react-query.gen";
import type { Claim } from "#/lib/api/types.gen";

import { claim, problem, reviewer } from "./fixtures";

// The first render of a route loads its code chunks, which is slow on a busy machine.
const slow = { timeout: 10_000 };

const claimId = "0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11";

// A claim that its reviewer may return: SUBMITTED, returned once before.
const returnable = (overrides: Partial<Claim> = {}) =>
  claim({
    version: 2,
    returnCount: 1,
    allowedActions: ["approve", "reject", "return"],
    ...overrides,
  });

let current: Claim = returnable();

let returnResponse: () => Response = () => HttpResponse.json({ data: current });

// While set, the return never answers, as when a request is still in flight.
let stalled = false;

interface Sent {
  ifMatch: string | null;
  body: unknown;
}

const sent: Sent[] = [];
let claimReads = 0;

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => HttpResponse.json({ data: reviewer })),
  http.get(`${apiUrl}/claims/${claimId}`, () => {
    claimReads += 1;

    return HttpResponse.json({ data: current });
  }),
  http.get(`${apiUrl}/claims`, () =>
    HttpResponse.json({ data: [], page: 1, pageSize: 20, total: 0 })
  ),
  http.post(`${apiUrl}/claims/${claimId}/return`, async ({ request }) => {
    sent.push({
      ifMatch: request.headers.get("If-Match"),
      body: await request.json(),
    });

    if (stalled) {
      await delay("infinite");
    }

    return returnResponse();
  })
);

serveApi(server);

const returned = (comment: string) =>
  claim({
    status: "RETURNED",
    version: 3,
    returnCount: 2,
    allowedActions: [],
    history: [
      ...claim().history,
      {
        id: 2,
        action: "RETURN",
        fromStatus: "SUBMITTED",
        toStatus: "RETURNED",
        actor: {
          id: "usr-reviewer",
          name: "Malee R.",
          position: "Head, Section A",
        },
        comment,
        createdAt: "2026-11-06T09:30:00Z",
      },
    ],
  });

const open = async (detail: Claim) => {
  current = detail;

  const rendered = renderAt(`/claims/${claimId}`);
  await screen.findByRole("heading", { name: "CLM-2026-000123" }, slow);

  return rendered;
};

const dialog = () => screen.getByRole("dialog");

// A page of the queue that no screen shows here, so nothing reads it again and only an invalidation can mark it stale.
// Built when used: the client's base URL is set before the tests, not before this module loads.
const unshownQueuePage = () =>
  listClaimsOptions({ query: { scope: "review", page: 2, pageSize: 20 } });

const unprocessable = () =>
  HttpResponse.json(
    problem(422, "VALIDATION_FAILED", "req-return-422", [
      {
        field: "comment",
        rule: "BR-04",
        message: "Reason must be at least 5 characters.",
      },
    ]),
    { status: 422 }
  );

beforeEach(() => {
  current = returnable();
  stalled = false;
  sent.length = 0;
  claimReads = 0;
  returnResponse = () => HttpResponse.json({ data: current });
});

describe("Decision panel: return", () => {
  it("F06-TEST-13: shows Return with the number of returns used", async () => {
    await open(returnable());

    screen.getByRole("button", { name: "Return" });
    screen.getByText("Returned 1 / 3");
    screen.getByRole("button", { name: "Approve" });
    screen.getByRole("button", { name: "Reject" });
  }, 20_000);

  it("F06-TEST-13: shows no Return button when return is not allowed", async () => {
    await open(
      returnable({ returnCount: 0, allowedActions: ["approve", "reject"] })
    );

    expect(screen.queryByRole("button", { name: "Return" })).toBeNull();
    expect(screen.queryByText(/Returned \d \/ 3/u)).toBeNull();
  }, 20_000);

  it("F06-TEST-13: with no returns left it says to approve or reject", async () => {
    await open(
      returnable({ returnCount: 3, allowedActions: ["approve", "reject"] })
    );

    screen.getByText("Returned 3 / 3 – approve or reject.");
    expect(screen.queryByRole("button", { name: "Return" })).toBeNull();
    screen.getByRole("button", { name: "Approve" });
  }, 20_000);

  it("F06-TEST-13: the dialog counts the reason, and a 422 puts the message under the field with the text kept", async () => {
    const { user } = await open(returnable());

    await user.click(screen.getByRole("button", { name: "Return" }));
    within(dialog()).getByRole("heading", { name: "Return claim" });
    within(dialog()).getByText("0/500");

    returnResponse = unprocessable;
    await user.type(
      within(dialog()).getByRole("textbox", { name: "Reason for return" }),
      "No"
    );
    within(dialog()).getByText("2/500");
    await user.click(within(dialog()).getByRole("button", { name: "Return" }));

    await within(dialog()).findByText("Reason must be at least 5 characters.");
    expect(
      within(dialog()).getByRole("textbox", { name: "Reason for return" })
    ).toHaveProperty("value", "No");
    expect(sent).toEqual([{ ifMatch: '"2"', body: { comment: "No" } }]);
  }, 30_000);

  it("F06-TEST-13: the buttons are disabled with a spinner while the return is sent", async () => {
    stalled = true;
    const { user } = await open(returnable());

    await user.click(screen.getByRole("button", { name: "Return" }));
    await user.click(within(dialog()).getByRole("button", { name: "Return" }));

    await waitFor(() => {
      expect(sent).toHaveLength(1);
    });
    expect(
      within(dialog()).getByRole("button", { name: "Cancel" })
    ).toHaveProperty("ariaDisabled", "true");
    expect(
      within(dialog()).getByRole("button", { name: /Return/u })
    ).toHaveProperty("ariaDisabled", "true");
  }, 20_000);

  it("F06-TEST-13: a return shows the toast and goes back to the review queue", async () => {
    const { router, user } = await open(returnable());
    const { queryClient } = router.options.context;
    await queryClient.query(unshownQueuePage());

    await user.click(screen.getByRole("button", { name: "Return" }));
    await user.type(
      within(dialog()).getByRole("textbox", { name: "Reason for return" }),
      "Please attach the receipt."
    );
    current = returned("Please attach the receipt.");
    await user.click(within(dialog()).getByRole("button", { name: "Return" }));

    await screen.findAllByText("Claim returned.");
    await screen.findByRole("heading", { name: "Review Queue" }, slow);
    expect(sent).toEqual([
      {
        ifMatch: '"2"',
        body: { comment: "Please attach the receipt." },
      },
    ]);
    // The claim left the queue, so every cached page of the queue is stale.
    expect(
      queryClient.getQueryState(unshownQueuePage().queryKey)?.isInvalidated
    ).toBe(true);
  }, 30_000);

  it("F06-TEST-13: a 409 shows the changed toast, closes the dialog, and reads the claim again", async () => {
    const { user } = await open(returnable());

    await user.click(screen.getByRole("button", { name: "Return" }));
    returnResponse = () => {
      current = claim({ status: "APPROVED", allowedActions: [] });

      return HttpResponse.json(problem(409, "CONFLICT", "req-return-409"), {
        status: 409,
      });
    };

    await user.click(within(dialog()).getByRole("button", { name: "Return" }));

    await screen.findAllByText(
      "This claim was changed by someone else – reloaded"
    );
    await waitFor(() => {
      expect(claimReads).toBe(2);
    });
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).toBeNull();
      expect(screen.queryByRole("button", { name: "Return" })).toBeNull();
    });
  }, 30_000);
});

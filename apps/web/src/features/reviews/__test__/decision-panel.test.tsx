import { fireEvent, screen, waitFor, within } from "@testing-library/react";
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

let current: Claim = claim();

let decisionResponse: () => Response = () =>
  HttpResponse.json({ data: current });

// While set, the decision never answers, as when a request is still in flight.
let stalled = false;

interface Sent {
  action: string;
  ifMatch: string | null;
  body: unknown;
}

const sent: Sent[] = [];
let claimReads = 0;

const decide = (action: string) =>
  http.post(`${apiUrl}/claims/${claimId}/${action}`, async ({ request }) => {
    sent.push({
      action,
      ifMatch: request.headers.get("If-Match"),
      body: await request.json(),
    });

    if (stalled) {
      await delay("infinite");
    }

    return decisionResponse();
  });

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => HttpResponse.json({ data: reviewer })),
  http.get(`${apiUrl}/claims/${claimId}`, () => {
    claimReads += 1;

    return HttpResponse.json({ data: current });
  }),
  http.get(`${apiUrl}/claims`, () =>
    HttpResponse.json({ data: [], page: 1, pageSize: 20, total: 0 })
  ),
  decide("approve"),
  decide("reject")
);

serveApi(server);

const decided = (status: "APPROVED" | "REJECTED", comment: string | null) =>
  claim({
    status,
    version: 1,
    decidedAt: "2026-11-06T09:30:00Z",
    allowedActions: [],
    history: [
      ...claim().history,
      {
        id: 2,
        action: status === "APPROVED" ? "APPROVE" : "REJECT",
        fromStatus: "SUBMITTED",
        toStatus: status,
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

// The dialog is a modal, so the page behind it is not in the accessibility tree.
// Built when used: the client's base URL is set before the tests, not before this module loads.
const queueQuery = () =>
  listClaimsOptions({ query: { scope: "review", page: 1, pageSize: 20 } });

const dialog = () => screen.getByRole("dialog");

beforeEach(() => {
  current = claim();
  stalled = false;
  sent.length = 0;
  claimReads = 0;
  decisionResponse = () => HttpResponse.json({ data: current });
});

describe("Decision panel", () => {
  it("F04-TEST-21: shows no panel when allowedActions is empty", async () => {
    await open(claim({ allowedActions: [] }));

    expect(screen.queryByRole("button", { name: "Approve" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Reject" })).toBeNull();
  }, 20_000);

  it("F04-TEST-21: shows only the actions the policy gave", async () => {
    await open(claim({ allowedActions: ["approve"] }));

    screen.getByRole("button", { name: "Approve" });
    expect(screen.queryByRole("button", { name: "Reject" })).toBeNull();
  }, 20_000);

  it("F04-TEST-21: approve asks for confirmation, sends the comment with the version, and shows a toast and the decided claim", async () => {
    const { router, user } = await open(claim({ version: 3 }));
    const { queryClient } = router.options.context;
    await queryClient.query(queueQuery());

    await user.click(screen.getByRole("button", { name: "Approve" }));
    within(dialog()).getByText("Approve claim CLM-2026-000123?");
    expect(sent).toHaveLength(0);

    await user.type(
      within(dialog()).getByRole("textbox", { name: "Comment (optional)" }),
      "Receipts match."
    );
    decisionResponse = () =>
      HttpResponse.json({ data: decided("APPROVED", "Receipts match.") });
    current = decided("APPROVED", "Receipts match.");
    await user.click(within(dialog()).getByRole("button", { name: "Approve" }));

    await screen.findAllByText("Claim approved.");
    expect(sent).toEqual([
      {
        action: "approve",
        ifMatch: '"3"',
        body: { comment: "Receipts match." },
      },
    ]);
    // The claim and the queue are read again; the panel goes away with allowedActions.
    await waitFor(() => {
      expect(claimReads).toBe(2);
    });
    // The queue is not on screen, so it is marked stale and loads again when it opens.
    await waitFor(() => {
      expect(
        queryClient.getQueryState(queueQuery().queryKey)?.isInvalidated
      ).toBe(true);
    });
    await waitFor(() => {
      expect(screen.queryByRole("button", { name: "Approve" })).toBeNull();
    });
    screen.getByText("Receipts match.");
  }, 30_000);

  it("F04-TEST-21: cancel closes the dialog and sends nothing", async () => {
    const { user } = await open(claim());

    await user.click(screen.getByRole("button", { name: "Approve" }));
    await user.click(within(dialog()).getByRole("button", { name: "Cancel" }));

    await waitFor(() => {
      expect(screen.queryByRole("dialog")).toBeNull();
    });
    expect(sent).toHaveLength(0);
  }, 20_000);

  it("F04-TEST-21: Escape closes the dialog when nothing is sent, and not while a decision is sent", async () => {
    const { user } = await open(claim());

    await user.click(screen.getByRole("button", { name: "Approve" }));
    fireEvent(dialog(), new Event("cancel", { cancelable: true }));
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).toBeNull();
    });

    stalled = true;
    await user.click(screen.getByRole("button", { name: "Approve" }));
    await user.click(within(dialog()).getByRole("button", { name: "Approve" }));
    await waitFor(() => {
      expect(sent).toHaveLength(1);
    });
    fireEvent(dialog(), new Event("cancel", { cancelable: true }));
    dialog();
  }, 30_000);

  it("F04-TEST-21: disables the dialog buttons with a spinner while the decision is sent", async () => {
    stalled = true;
    const { user } = await open(claim());

    await user.click(screen.getByRole("button", { name: "Approve" }));
    await user.click(within(dialog()).getByRole("button", { name: "Approve" }));

    await waitFor(() => {
      expect(sent).toHaveLength(1);
    });
    expect(
      within(dialog()).getByRole("button", { name: "Cancel" })
    ).toHaveProperty("ariaDisabled", "true");
    expect(
      within(dialog()).getByRole("button", { name: /Approve/u })
    ).toHaveProperty("ariaDisabled", "true");
  }, 20_000);

  it("F04-TEST-21: reject shows the counter, and a 422 puts the message under the reason with the text kept", async () => {
    const { user } = await open(claim());

    await user.click(screen.getByRole("button", { name: "Reject" }));
    within(dialog()).getByText("Reject claim CLM-2026-000123?");
    within(dialog()).getByText("0/500");

    decisionResponse = () =>
      HttpResponse.json(
        problem(422, "VALIDATION_FAILED", "req-reject-422", [
          {
            field: "comment",
            rule: "BR-04",
            message: "Reason must be at least 5 characters.",
          },
        ]),
        { status: 422 }
      );
    await user.type(
      within(dialog()).getByRole("textbox", { name: "Reason" }),
      "No"
    );
    within(dialog()).getByText("2/500");
    await user.click(within(dialog()).getByRole("button", { name: "Reject" }));

    await within(dialog()).findByText("Reason must be at least 5 characters.");
    expect(
      within(dialog()).getByRole("textbox", { name: "Reason" })
    ).toHaveProperty("value", "No");
    expect(sent).toEqual([
      { action: "reject", ifMatch: '"0"', body: { comment: "No" } },
    ]);
  }, 30_000);

  it("F04-TEST-21: reject succeeds with a toast and the reason in the history", async () => {
    const { user } = await open(claim());

    await user.click(screen.getByRole("button", { name: "Reject" }));
    await user.type(
      within(dialog()).getByRole("textbox", { name: "Reason" }),
      "Receipt is missing."
    );
    current = decided("REJECTED", "Receipt is missing.");
    decisionResponse = () => HttpResponse.json({ data: current });
    await user.click(within(dialog()).getByRole("button", { name: "Reject" }));

    await screen.findAllByText("Claim rejected.");
    await screen.findByText("Receipt is missing.");
    expect(screen.queryByRole("button", { name: "Reject" })).toBeNull();
  }, 30_000);

  it("F04-TEST-21: a 409 shows the changed toast, closes the dialog, and reads the claim again", async () => {
    const { user } = await open(claim());

    await user.click(screen.getByRole("button", { name: "Approve" }));
    decisionResponse = () => {
      current = decided("APPROVED", null);

      return HttpResponse.json(problem(409, "CONFLICT", "req-approve-409"), {
        status: 409,
      });
    };

    await user.click(within(dialog()).getByRole("button", { name: "Approve" }));

    await screen.findAllByText(
      "This claim was changed by someone else – reloaded"
    );
    await waitFor(() => {
      expect(claimReads).toBe(2);
    });
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).toBeNull();
      expect(screen.queryByRole("button", { name: "Approve" })).toBeNull();
    });
  }, 30_000);

  it("F04-TEST-21: a 403 closes the dialog and shows the access page", async () => {
    const { user } = await open(claim());

    await user.click(screen.getByRole("button", { name: "Approve" }));
    decisionResponse = () => {
      server.use(
        http.get(`${apiUrl}/claims/${claimId}`, () =>
          HttpResponse.json(problem(403, "FORBIDDEN", "req-claim-403"), {
            status: 403,
          })
        )
      );

      return HttpResponse.json(problem(403, "FORBIDDEN", "req-approve-403"), {
        status: 403,
      });
    };

    await user.click(within(dialog()).getByRole("button", { name: "Approve" }));

    await screen.findByText("You don't have access to this claim.");
    expect(screen.queryByRole("dialog")).toBeNull();
  }, 30_000);

  it("F04-TEST-21: another failure shows an error toast with the request ID and keeps the dialog", async () => {
    const { user } = await open(claim());

    await user.click(screen.getByRole("button", { name: "Approve" }));
    decisionResponse = () =>
      HttpResponse.json(problem(500, "INTERNAL", "req-approve-500"), {
        status: 500,
      });
    await user.click(within(dialog()).getByRole("button", { name: "Approve" }));

    await screen.findAllByText(
      "Something went wrong. Request ID: req-approve-500"
    );
    dialog();
  }, 30_000);

  it("F04-TEST-21: a network failure shows an error toast without a request ID", async () => {
    const { user } = await open(claim());

    await user.click(screen.getByRole("button", { name: "Approve" }));
    decisionResponse = () => HttpResponse.error();
    await user.click(within(dialog()).getByRole("button", { name: "Approve" }));

    await screen.findAllByText("Something went wrong.");
    dialog();
  }, 30_000);
});

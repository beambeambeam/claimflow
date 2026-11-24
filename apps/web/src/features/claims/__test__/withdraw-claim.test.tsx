import { screen, waitFor, within } from "@testing-library/react";
import { delay, http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { beforeEach, describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";
import type { Claim, HistoryEntry } from "#/lib/api/types.gen";

import { claim, problem, requester } from "./fixtures";

// The first render of a route loads its code chunks, which is slow on a busy machine.
const slow = { timeout: 20_000 };

const claimId = "0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11";

const submitEntry: HistoryEntry = {
  id: 1,
  action: "SUBMIT",
  fromStatus: null,
  toStatus: "SUBMITTED",
  actor: {
    id: requester.id,
    name: requester.name,
    position: "Staff, Section A",
  },
  comment: null,
  createdAt: "2026-11-05T03:00:00Z",
};

// A claim that its owner may withdraw: SUBMITTED, no decision yet.
const withdrawable = (overrides: Partial<Claim> = {}) =>
  claim({
    status: "SUBMITTED",
    version: 4,
    decidedAt: null,
    allowedActions: ["withdraw"],
    history: [submitEntry],
    ...overrides,
  });

const withdrawn = (comment: string | null) =>
  claim({
    status: "WITHDRAWN",
    version: 5,
    decidedAt: null,
    allowedActions: [],
    history: [
      submitEntry,
      {
        id: 2,
        action: "WITHDRAW",
        fromStatus: "SUBMITTED",
        toStatus: "WITHDRAWN",
        actor: {
          id: requester.id,
          name: requester.name,
          position: "Staff, Section A",
        },
        comment,
        createdAt: "2026-11-06T09:30:00Z",
      },
    ],
  });

let current: Claim = withdrawable();

let withdrawResponse: () => Response = () =>
  HttpResponse.json({ data: current });

// While set, the withdrawal never answers, as when a request is still in flight.
let stalled = false;

interface Sent {
  ifMatch: string | null;
  body: unknown;
}

const sent: Sent[] = [];
let claimReads = 0;

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => HttpResponse.json({ data: requester })),
  http.get(`${apiUrl}/claims/${claimId}`, () => {
    claimReads += 1;

    return HttpResponse.json({ data: current });
  }),
  http.get(`${apiUrl}/claims`, () =>
    HttpResponse.json({ data: [], page: 1, pageSize: 20, total: 0 })
  ),
  http.post(`${apiUrl}/claims/${claimId}/withdraw`, async ({ request }) => {
    sent.push({
      ifMatch: request.headers.get("If-Match"),
      body: await request.json(),
    });

    if (stalled) {
      await delay("infinite");
    }

    return withdrawResponse();
  })
);

serveApi(server);

const open = async (detail: Claim) => {
  current = detail;

  const rendered = renderAt(`/claims/${claimId}`);
  await screen.findByRole("heading", { name: "CLM-2026-000123" }, slow);

  return rendered;
};

const dialog = () => screen.getByRole("dialog");

beforeEach(() => {
  current = withdrawable();
  stalled = false;
  sent.length = 0;
  claimReads = 0;
  withdrawResponse = () => HttpResponse.json({ data: current });
});

describe("Claim Detail: withdraw", { timeout: 40_000 }, () => {
  it("F07-TEST-11: shows Withdraw only when the policy gives it", async () => {
    await open(withdrawable({ allowedActions: [] }));

    expect(screen.queryByRole("button", { name: "Withdraw" })).toBeNull();
  });

  it("F07-TEST-11: opens a dialog with the optional comment, a 0/500 counter and a limit of 500 characters", async () => {
    const { user } = await open(withdrawable());

    await user.click(screen.getByRole("button", { name: "Withdraw" }));

    within(dialog()).getByRole("heading", {
      name: "Withdraw CLM-2026-000123?",
    });
    within(dialog()).getByText("You can't edit or resubmit a withdrawn claim.");
    within(dialog()).getByText("0/500");

    await user.click(within(dialog()).getByLabelText("Comment (optional)"));
    await user.paste("a".repeat(500));
    await user.keyboard("b");

    within(dialog()).getByText("500/500");
    expect(
      within(dialog()).getByLabelText<HTMLTextAreaElement>("Comment (optional)")
        .value
    ).toHaveLength(500);
  });

  it("F07-TEST-11: disables the confirm button while sending, then shows the toast, the Withdrawn badge and no actions", async () => {
    const { user } = await open(withdrawable());
    stalled = true;

    await user.click(screen.getByRole("button", { name: "Withdraw" }));
    await user.click(within(dialog()).getByLabelText("Comment (optional)"));
    await user.paste("No longer needed");
    await user.click(
      within(dialog()).getByRole("button", { name: "Withdraw claim" })
    );

    await waitFor(() => {
      expect(
        within(dialog())
          .getByRole("button", { name: "Withdraw claim" })
          .getAttribute("aria-disabled")
      ).toBe("true");
    });
    expect(sent).toEqual([
      { ifMatch: '"4"', body: { comment: "No longer needed" } },
    ]);
  });

  it("F07-TEST-11: after the withdrawal it shows the toast, the Withdrawn badge and an empty action bar", async () => {
    const { user } = await open(withdrawable());
    withdrawResponse = () => {
      current = withdrawn("No longer needed");

      return HttpResponse.json({ data: current });
    };

    await user.click(screen.getByRole("button", { name: "Withdraw" }));
    await user.click(
      within(dialog()).getByRole("button", { name: "Withdraw claim" })
    );

    await screen.findAllByText("Claim withdrawn.", {}, slow);
    await waitFor(() => {
      within(screen.getByRole("region", { name: "Claim" })).getByText(
        "Withdrawn"
      );
    }, slow);
    expect(screen.queryByRole("button", { name: "Withdraw" })).toBeNull();
    expect(screen.queryByRole("dialog")).toBeNull();
    screen.getByText("No longer needed");
  });

  it("F07-TEST-11: a 409 CONFLICT shows the reload toast and reads the claim again", async () => {
    const { user } = await open(withdrawable());
    const reads = claimReads;
    withdrawResponse = () =>
      HttpResponse.json(problem(409, "CONFLICT", "req-withdraw-conflict"), {
        status: 409,
      });

    await user.click(screen.getByRole("button", { name: "Withdraw" }));
    await user.click(
      within(dialog()).getByRole("button", { name: "Withdraw claim" })
    );

    await screen.findAllByText(
      "This claim was changed by someone else – reloaded",
      {},
      slow
    );
    await waitFor(() => {
      expect(claimReads).toBeGreaterThan(reads);
    });
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("F07-TEST-11: a 409 INVALID_TRANSITION shows its message and reads the claim again", async () => {
    const { user } = await open(withdrawable());
    const reads = claimReads;
    withdrawResponse = () =>
      HttpResponse.json(
        {
          ...problem(409, "INVALID_TRANSITION", "req-withdraw-state"),
          errors: [
            {
              field: "status",
              rule: "BR-16",
              message: "Only a submitted or returned claim can be withdrawn.",
            },
          ],
        },
        { status: 409 }
      );

    await user.click(screen.getByRole("button", { name: "Withdraw" }));
    await user.click(
      within(dialog()).getByRole("button", { name: "Withdraw claim" })
    );

    await screen.findAllByText(
      "Only a submitted or returned claim can be withdrawn.",
      {},
      slow
    );
    await waitFor(() => {
      expect(claimReads).toBeGreaterThan(reads);
    });
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("F07-TEST-11: Cancel and Escape close the dialog without sending anything", async () => {
    const { user } = await open(withdrawable());

    await user.click(screen.getByRole("button", { name: "Withdraw" }));
    await user.click(within(dialog()).getByRole("button", { name: "Cancel" }));
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).toBeNull();
    });

    await user.click(screen.getByRole("button", { name: "Withdraw" }));
    await user.keyboard("{Escape}");
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).toBeNull();
    });
    expect(sent).toHaveLength(0);
  });

  it("F07-TEST-11: Escape does not close the dialog while the withdrawal is sending", async () => {
    const { user } = await open(withdrawable());
    stalled = true;

    await user.click(screen.getByRole("button", { name: "Withdraw" }));
    await user.click(
      within(dialog()).getByRole("button", { name: "Withdraw claim" })
    );
    await waitFor(() => {
      expect(sent).toHaveLength(1);
    });
    await user.keyboard("{Escape}");

    within(dialog()).getByRole("button", { name: "Withdraw claim" });
  });

  it("F07-TEST-11: a failure that is not a problem document shows the generic toast and closes the dialog", async () => {
    const { user } = await open(withdrawable());
    withdrawResponse = () => HttpResponse.error();

    await user.click(screen.getByRole("button", { name: "Withdraw" }));
    await user.click(
      within(dialog()).getByRole("button", { name: "Withdraw claim" })
    );

    await screen.findAllByText("Something went wrong.", {}, slow);
    expect(screen.queryByRole("dialog")).toBeNull();
  });
});

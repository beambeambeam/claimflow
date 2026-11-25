import { screen, waitFor, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { beforeEach, describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";
import type { Claim } from "#/lib/api/types.gen";

import { financeOfficer, pendingClaim, problem } from "./fixtures";

// The first render of a route loads its code chunks, which is slow on a busy machine.
const slow = { timeout: 10_000 };

const claimId = "0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11";

let current: Claim = pendingClaim();

let decisionResponse: () => Response = () =>
  HttpResponse.json({ data: current });

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

    return decisionResponse();
  });

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () =>
    HttpResponse.json({ data: financeOfficer })
  ),
  http.get(`${apiUrl}/claims/${claimId}`, () => {
    claimReads += 1;

    return HttpResponse.json({ data: current });
  }),
  decide("final-approve"),
  decide("final-reject")
);

serveApi(server);

const decided = (status: "APPROVED" | "REJECTED") =>
  pendingClaim({
    status,
    version: 2,
    decidedAt: "2026-11-07T09:30:00Z",
    allowedActions: [],
  });

const open = async (detail: Claim) => {
  current = detail;

  const rendered = renderAt(`/claims/${claimId}`);
  await screen.findByRole("heading", { name: "CLM-2026-000123" }, slow);

  return rendered;
};

const dialog = () => screen.getByRole("dialog");

beforeEach(() => {
  current = pendingClaim();
  sent.length = 0;
  claimReads = 0;
  decisionResponse = () => HttpResponse.json({ data: current });
});

describe("Final decision panel", () => {
  it("F08-TEST-16: shows no panel when allowedActions is empty", async () => {
    await open(pendingClaim({ allowedActions: [] }));

    expect(screen.queryByRole("button", { name: "Final approve" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Final reject" })).toBeNull();
  }, 20_000);

  it("F08-TEST-16: shows Final approve and Final reject, and no Return", async () => {
    await open(pendingClaim());

    screen.getByRole("button", { name: "Final approve" });
    screen.getByRole("button", { name: "Final reject" });
    expect(screen.queryByRole("button", { name: "Return" })).toBeNull();
  }, 20_000);

  it("F08-TEST-16: final approve asks for confirmation with an optional comment, then shows a toast and the decided claim", async () => {
    const { user } = await open(pendingClaim());

    await user.click(screen.getByRole("button", { name: "Final approve" }));
    within(dialog()).getByText(
      "Approve claim CLM-2026-000123 as the final decision?"
    );
    within(dialog()).getByText("0/500");
    expect(sent).toHaveLength(0);

    current = decided("APPROVED");
    await user.click(within(dialog()).getByRole("button", { name: "Approve" }));

    await screen.findAllByText("Claim approved.");
    expect(sent).toEqual([
      { action: "final-approve", ifMatch: '"1"', body: { comment: "" } },
    ]);
    await waitFor(() => {
      expect(claimReads).toBe(2);
    });
    await waitFor(() => {
      expect(
        screen.queryByRole("button", { name: "Final approve" })
      ).toBeNull();
    });
  }, 30_000);

  it("F08-TEST-16: final reject is disabled under 10 characters and shows a 422 under the reason", async () => {
    const { user } = await open(pendingClaim());

    await user.click(screen.getByRole("button", { name: "Final reject" }));
    within(dialog()).getByText(
      "Reject claim CLM-2026-000123 as the final decision?"
    );
    const reason = within(dialog()).getByRole("textbox", { name: "Reason" });
    const reject = within(dialog()).getByRole("button", { name: "Reject" });

    await user.type(reason, "abcdefghi");
    expect(reject).toHaveProperty("ariaDisabled", "true");
    within(dialog()).getByText("Reason must be at least 10 characters.");

    decisionResponse = () =>
      HttpResponse.json(
        problem(422, "VALIDATION_FAILED", "req-final-422", [
          {
            field: "comment",
            rule: "BR-04",
            message: "Comment cannot be more than 500 characters.",
          },
        ]),
        { status: 422 }
      );
    await user.type(reason, "j");
    await user.click(reject);

    await within(dialog()).findByText(
      "Comment cannot be more than 500 characters."
    );
    expect(sent).toEqual([
      {
        action: "final-reject",
        ifMatch: '"1"',
        body: { comment: "abcdefghij" },
      },
    ]);
  }, 30_000);

  it("F08-TEST-16: final reject succeeds with a toast", async () => {
    const { user } = await open(pendingClaim());

    await user.click(screen.getByRole("button", { name: "Final reject" }));
    await user.type(
      within(dialog()).getByRole("textbox", { name: "Reason" }),
      "Budget code is wrong"
    );
    current = decided("REJECTED");
    await user.click(within(dialog()).getByRole("button", { name: "Reject" }));

    await screen.findAllByText("Claim rejected.");
    await waitFor(() => {
      expect(screen.queryByRole("button", { name: "Final reject" })).toBeNull();
    });
  }, 30_000);
});

import { screen, waitFor, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";
import type { Claim, FieldError, Problem } from "#/lib/api/types.gen";
import { zClaimInput } from "#/lib/api/zod.gen";

import { claim, problem, requester, summary } from "./fixtures";

const draft = claim({
  status: "DRAFT",
  assignedReviewer: null,
  firstSubmittedAt: null,
  submittedAt: null,
  decidedAt: null,
  history: [],
  allowedActions: ["edit", "delete", "submit"],
});

const submitted = claim({ status: "SUBMITTED", allowedActions: [] });
const path = `/claims/${draft.id}`;
const detailUrl = `${apiUrl}${path}`;
const slow = { timeout: 10_000 };
const expenseError = "Expense date cannot be in the future.";

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => HttpResponse.json({ data: requester })),
  http.get(detailUrl, () => HttpResponse.json({ data: draft })),
  http.get(`${apiUrl}/claims`, () =>
    HttpResponse.json({ data: [], page: 1, pageSize: 20, total: 0 })
  )
);

serveApi(server);

const fieldFailure = (field: string, message: string, rule = "BR-07") =>
  HttpResponse.json(
    {
      ...problem(422, "VALIDATION_FAILED", "draft-invalid"),
      errors: [{ field, message, rule }],
    },
    { status: 422 }
  );

const conflict = () =>
  HttpResponse.json(problem(409, "CONFLICT", "draft-conflict"), {
    status: 409,
  });

const failure = (status: number, code: Problem["code"], requestId: string) =>
  HttpResponse.json(problem(status, code, requestId), { status });

const transitionFailure = (errors?: FieldError[]) =>
  HttpResponse.json(
    {
      ...problem(409, "INVALID_TRANSITION", "draft-state"),
      title: "Draft state changed.",
      errors,
    },
    { status: 409 }
  );

const submitTransitionField: FieldError = {
  field: "status",
  rule: "BR-03",
  message: "Only a draft or a returned claim can be submitted.",
};

const transitionField: FieldError = {
  field: "status",
  rule: "BR-10",
  message: "Only a draft or a returned claim can be edited.",
};

const button = (name: string) => screen.getByRole("button", { name });

describe("Draft workflow", { timeout: 20_000 }, () => {
  it("F05-TEST-21: keeps input on a draft save error, disables both actions while saving, and opens the draft", async () => {
    let released = false;
    let url: URL | undefined;
    let requests = 0;

    server.use(
      http.post(`${apiUrl}/claims`, async ({ request }) => {
        url = new URL(request.url);
        requests += 1;
        await waitFor(() => {
          expect(released).toBe(true);
        }, slow);

        if (requests === 1) {
          return fieldFailure("items[0].expenseDate", expenseError);
        }

        return HttpResponse.json({ data: draft }, { status: 201 });
      })
    );
    const { router, user } = renderAt("/claims/new");
    await user.type(await screen.findByLabelText("Title", {}, slow), "Draft");
    await user.type(screen.getByLabelText("Purpose"), "Office supplies");
    await user.click(screen.getByLabelText("Category", { selector: "button" }));
    await user.click(await screen.findByRole("option", { name: "Travel" }));
    await user.type(screen.getByLabelText("Description"), "Train");
    await user.type(screen.getByLabelText("Expense date"), "2026-11-03");
    await user.type(screen.getByLabelText("Amount"), "250.00");
    await user.type(screen.getByLabelText("Receipt reference"), "R-1");
    await user.click(button("Save as draft"));

    try {
      await waitFor(() => {
        expect(url?.searchParams.get("draft")).toBe("true");
      });
      expect(button("Submit").getAttribute("aria-disabled")).toBe("true");
      expect(button("Save as draft").getAttribute("aria-disabled")).toBe(
        "true"
      );
    } finally {
      released = true;
    }

    await screen.findAllByText(expenseError);
    expect(screen.getByLabelText<HTMLInputElement>("Title").value).toBe(
      "Draft"
    );
    await user.click(button("Save as draft"));
    await screen.findAllByText("Draft saved.");
    await waitFor(() => {
      expect(router.state.location.pathname).toBe(path);
    });
  });

  it("F05-TEST-22: shows draft placeholders and hides actions absent from policy", async () => {
    server.use(
      http.get(detailUrl, () =>
        HttpResponse.json({ data: { ...draft, allowedActions: [] } })
      )
    );
    renderAt(path);
    await screen.findByText("Not submitted yet", {}, slow);
    screen.getByText("Assigned when submitted");
    screen.getByText(
      "No history yet – history starts when the claim is submitted."
    );
    expect(screen.queryByRole("link", { name: "Edit" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Submit claim" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Delete draft" })).toBeNull();
  });

  it("F05-TEST-22: shows each invalid line, then submits with the version and refetches", async () => {
    let current = draft;
    const versions: (string | null)[] = [];
    server.use(
      http.get(detailUrl, () => HttpResponse.json({ data: current })),
      http.post(`${detailUrl}/submit`, ({ request }) => {
        versions.push(request.headers.get("If-Match"));

        if (versions.length === 1) {
          return fieldFailure("items[0].expenseDate", expenseError);
        }

        current = submitted;

        return HttpResponse.json({ data: submitted });
      })
    );
    const { user } = renderAt(path);
    await user.click(
      await screen.findByRole("button", { name: "Submit claim" }, slow)
    );
    const alert = await screen.findByText(`Line 1: ${expenseError}`);
    const lineItems = screen.getByRole("region", { name: "Line items" });
    expect(lineItems.contains(alert)).toBe(true);
    within(lineItems).getByRole("link", { name: "Edit" });

    await user.click(button("Submit claim"));
    await screen.findAllByText("Claim submitted.");
    const header = screen.getByRole("region", { name: "Claim" });
    await within(header).findByText("Submitted");
    expect(versions).toEqual(['"2"', '"2"']);
  });

  it("F05-TEST-22: reloads on conflict and confirms before deleting", async () => {
    let reads = 0;
    let releaseDelete = false;
    const deletes: (string | null)[] = [];
    server.use(
      http.get(detailUrl, () => {
        reads += 1;

        return HttpResponse.json({
          data: {
            ...draft,
            version: reads + 1,
            allowedActions:
              reads === 1 ? draft.allowedActions : ["edit", "delete"],
          },
        });
      }),
      http.post(`${detailUrl}/submit`, conflict),
      http.delete(detailUrl, async ({ request }) => {
        deletes.push(request.headers.get("If-Match"));
        await waitFor(() => {
          expect(releaseDelete).toBe(true);
        }, slow);

        return new HttpResponse(null, { status: 204 });
      })
    );
    const { router, user } = renderAt(path);
    await user.click(
      await screen.findByRole("button", { name: "Submit claim" }, slow)
    );
    await screen.findAllByText(
      "This claim was changed by someone else – reloaded"
    );
    await waitFor(() => {
      expect(reads).toBe(2);
    });

    await waitFor(() => {
      expect(screen.queryByRole("button", { name: "Submit claim" })).toBeNull();
    });
    await user.click(button("Delete draft"));
    await screen.findByRole("dialog");
    await user.keyboard("{Escape}");
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).toBeNull();
    });
    expect(deletes).toEqual([]);

    await user.click(button("Delete draft"));
    const dialog = await screen.findByRole("dialog");
    within(dialog).getByText(
      `Delete draft ${draft.claimNo}? This cannot be undone.`
    );
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(deletes).toEqual([]);

    await user.click(button("Delete draft"));
    await user.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: "Delete",
      })
    );

    try {
      await waitFor(() => {
        expect(deletes).toEqual(['"3"']);
        expect(
          within(screen.getByRole("dialog"))
            .getByRole("button", { name: "Cancel" })
            .getAttribute("aria-disabled")
        ).toBe("true");
      });
      await user.keyboard("{Escape}");
      expect(screen.getByRole("dialog")).toBeDefined();
    } finally {
      releaseDelete = true;
    }

    await screen.findAllByText("Draft deleted.");
    await waitFor(() => {
      expect(router.state.location.pathname).toBe("/claims");
    });
    await screen.findByText("No claims yet");
    expect(deletes).toEqual(['"3"']);
  });

  it("F05-TEST-23: prefills the form, preserves invalid input, and reloads on conflict", async () => {
    let current: Claim = draft;
    const sent: unknown[] = [];
    server.use(
      http.get(detailUrl, () => HttpResponse.json({ data: current })),
      http.put(detailUrl, async ({ request }) => {
        const { title } = zClaimInput.parse(await request.json());
        sent.push({ title, version: request.headers.get("If-Match") });

        if (sent.length === 1) {
          return fieldFailure("items[0].expenseDate", expenseError);
        }

        if (sent.length === 2) {
          current = { ...draft, title: "Changed elsewhere", version: 3 };

          return conflict();
        }

        return HttpResponse.json({ data: current });
      })
    );
    const { router, user } = renderAt(`${path}/edit`);

    const title = await screen.findByLabelText<HTMLInputElement>(
      "Title",
      {},
      slow
    );

    expect(title.value).toBe(draft.title);
    expect(
      screen
        .getAllByLabelText<HTMLInputElement>("Amount")
        .map((input) => input.value)
    ).toEqual(["250.00", "1000.00"]);
    await user.clear(title);
    await user.type(title, "Edited draft");
    await user.click(button("Save changes"));
    await screen.findAllByText(expenseError);
    expect(title.value).toBe("Edited draft");
    await user.click(button("Save changes"));
    await screen.findAllByText(
      "This claim was changed by someone else – reloaded"
    );
    await waitFor(() => {
      expect(screen.getByLabelText<HTMLInputElement>("Title").value).toBe(
        "Changed elsewhere"
      );
    });

    await user.click(button("Save changes"));
    await screen.findAllByText("Changes saved.");
    await waitFor(() => {
      expect(router.state.location.pathname).toBe(path);
    });
    expect(sent).toEqual([
      { title: "Edited draft", version: '"2"' },
      { title: "Edited draft", version: '"2"' },
      { title: "Changed elsewhere", version: '"3"' },
    ]);
  });

  it("F05-TEST-23: declines the edit route without the edit action", async () => {
    server.use(
      http.get(detailUrl, () => HttpResponse.json({ data: submitted }))
    );
    renderAt(`${path}/edit`);
    await screen.findByText("This claim can't be edited.", {}, slow);
    expect(screen.queryByLabelText("Title")).toBeNull();
  });

  it("F05-TEST-24: lists a draft with a neutral badge and a dash in Submitted", async () => {
    server.use(
      http.get(`${apiUrl}/claims`, () =>
        HttpResponse.json({
          data: [summary({ status: "DRAFT", submittedAt: null })],
          page: 1,
          pageSize: 20,
          total: 1,
        })
      )
    );
    renderAt("/claims");
    const badge = await screen.findByText("Draft", {}, slow);
    expect(badge.closest<HTMLElement>("[data-variant]")?.dataset.variant).toBe(
      "neutral"
    );
    const row = screen.getByRole("row", { name: /Bangkok workshop travel/u });
    expect(within(row).getAllByRole("cell").at(-1)?.textContent).toBe("–");
  });
});

describe("Draft recovery", { timeout: 20_000 }, () => {
  it("F05-TEST-22: shows a claim-wide validation message without an unavailable edit action", async () => {
    const message = "Claim total cannot be more than 50,000.00 THB.";
    server.use(
      http.get(detailUrl, () =>
        HttpResponse.json({ data: { ...draft, allowedActions: ["submit"] } })
      ),
      http.post(`${detailUrl}/submit`, () =>
        fieldFailure("total", message, "BR-06")
      )
    );

    const { user } = renderAt(path);
    await user.click(
      await screen.findByRole("button", { name: "Submit claim" }, slow)
    );
    const lineItems = screen.getByRole("region", { name: "Line items" });
    await within(lineItems).findByText(message);
    expect(screen.queryByRole("link", { name: "Edit" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Delete draft" })).toBeNull();
  });

  it.each([
    { errors: [submitTransitionField], message: submitTransitionField.message },
    { errors: undefined, message: "Draft state changed." },
  ])(
    "F05-TEST-22: reloads after a rejected transition and shows $message",
    async ({ errors, message }) => {
      let current = draft;
      server.use(
        http.get(detailUrl, () => HttpResponse.json({ data: current })),
        http.post(`${detailUrl}/submit`, () => {
          current = submitted;

          return transitionFailure(errors);
        })
      );

      const { user } = renderAt(path);
      await user.click(
        await screen.findByRole("button", { name: "Submit claim" }, slow)
      );
      await screen.findAllByText(message);
      await within(screen.getByRole("region", { name: "Claim" })).findByText(
        "Submitted"
      );
    }
  );

  it.each([
    {
      name: "the server returns an error",
      respond: () => failure(500, "INTERNAL", "submit-500"),
      message: "Something went wrong. Request ID: submit-500",
    },
    {
      name: "the network fails",
      respond: () => HttpResponse.error(),
      message: "Something went wrong.",
    },
  ])(
    "F05-TEST-22: reports a failed submission when $name",
    async ({ respond, message }) => {
      server.use(http.post(`${detailUrl}/submit`, respond));

      const { user } = renderAt(path);
      await user.click(
        await screen.findByRole("button", { name: "Submit claim" }, slow)
      );
      await screen.findAllByText(message);
      expect(button("Submit claim").getAttribute("aria-disabled")).not.toBe(
        "true"
      );
    }
  );

  it.each([
    {
      name: "the server refuses access",
      respond: () => failure(403, "FORBIDDEN", "delete-403"),
      message: "You don't have access to this claim.",
    },
    {
      name: "the network fails",
      respond: () => HttpResponse.error(),
      message: "Something went wrong.",
    },
  ])(
    "F05-TEST-22: closes the delete confirmation when $name",
    async ({ respond, message }) => {
      server.use(http.delete(detailUrl, respond));

      const { router, user } = renderAt(path);
      await user.click(
        await screen.findByRole("button", { name: "Delete draft" }, slow)
      );
      await user.click(
        within(await screen.findByRole("dialog")).getByRole("button", {
          name: "Delete",
        })
      );
      await screen.findAllByText(message);
      await waitFor(() => {
        expect(screen.queryByRole("dialog")).toBeNull();
      });
      expect(router.state.location.pathname).toBe(path);
    }
  );

  it.each([
    {
      status: 403,
      code: "FORBIDDEN",
      message: "You don't have access to this claim.",
    },
    { status: 404, code: "NOT_FOUND", message: "Claim not found." },
  ] as const)(
    "F05-TEST-23: shows the access result when saving returns $status",
    async ({ status, code, message }) => {
      server.use(
        http.put(detailUrl, () => failure(status, code, "edit-unavailable"))
      );

      const { user } = renderAt(`${path}/edit`);
      await user.click(
        await screen.findByRole("button", { name: "Save changes" }, slow)
      );
      await screen.findByText(message);
      expect(screen.queryByRole("button", { name: "Save changes" })).toBeNull();
    }
  );

  it.each([
    { errors: [transitionField], message: transitionField.message },
    { errors: undefined, message: "Draft state changed." },
  ])(
    "F05-TEST-23: returns to detail after a rejected edit and shows $message",
    async ({ errors, message }) => {
      let current = draft;
      server.use(
        http.get(detailUrl, () => HttpResponse.json({ data: current })),
        http.put(detailUrl, () => {
          current = submitted;

          return transitionFailure(errors);
        })
      );

      const { router, user } = renderAt(`${path}/edit`);
      await user.click(
        await screen.findByRole("button", { name: "Save changes" }, slow)
      );
      await screen.findAllByText(message);
      await within(
        await screen.findByRole("region", { name: "Claim" })
      ).findByText("Submitted");
      expect(router.state.location.pathname).toBe(path);
    }
  );

  // The reload is held, so the form renders while the save still has its 409.
  it("F05-TEST-23: shows no form error while a conflicting draft reloads", async () => {
    let submittedEdit = false;
    let released = false;
    server.use(
      http.get(detailUrl, async () => {
        if (!submittedEdit) {
          return HttpResponse.json({ data: draft });
        }

        await waitFor(() => {
          expect(released).toBe(true);
        }, slow);

        return HttpResponse.json({
          data: { ...draft, title: "Latest draft", version: 3 },
        });
      }),
      http.put(detailUrl, () => {
        submittedEdit = true;

        return conflict();
      })
    );

    const { user } = renderAt(`${path}/edit`);
    await user.click(
      await screen.findByRole("button", { name: "Save changes" }, slow)
    );

    try {
      await screen.findAllByText(
        "This claim was changed by someone else – reloaded"
      );
      expect(button("Save changes").getAttribute("aria-disabled")).toBe("true");
      expect(screen.queryByText(/draft-conflict/u)).toBeNull();
    } finally {
      released = true;
    }

    await waitFor(() => {
      expect(screen.getByLabelText<HTMLInputElement>("Title").value).toBe(
        "Latest draft"
      );
    });
  });

  it("F05-TEST-23: retries a failed conflict reload and opens the latest draft", async () => {
    let submittedEdit = false;
    let failedReload = false;
    server.use(
      http.get(detailUrl, () => {
        if (!submittedEdit) {
          return HttpResponse.json({ data: draft });
        }

        if (!failedReload) {
          failedReload = true;

          return failure(503, "INTERNAL", "edit-reload");
        }

        return HttpResponse.json({
          data: { ...draft, title: "Latest draft", version: 3 },
        });
      }),
      http.put(detailUrl, () => {
        submittedEdit = true;

        return conflict();
      })
    );

    const { user } = renderAt(`${path}/edit`);
    await user.click(
      await screen.findByRole("button", { name: "Save changes" }, slow)
    );
    await screen.findByText("Request ID: edit-reload");
    await user.click(button("Try again"));
    const title = await screen.findByLabelText<HTMLInputElement>("Title");
    expect(title.value).toBe("Latest draft");
  });
});

# Issue tracker: GitHub Issues

Issues and specs for this repo live as GitHub Issues on `beambeambeam/claimflow`. Use the `gh` CLI.

## Conventions

- Title an implementation ticket `<feature or lab>/<NN>: <title>`, for example `F-05/01: Save and view drafts` or `L5/02: Cloud-ready image, compose stack, and image CI gate`. `NN` starts at `01` for each feature.
- The body follows the existing tickets: Objective, Context, Scope (In scope, Out of scope), Implementation notes, Dependencies and risks, Acceptance criteria, Validation, Related work.
- Record blocking as `Blocked by #N (<ticket>)` under Dependencies and risks.
- Add every issue to the "claimflow's project" board (project 5). Its Status field is the triage state: `Backlog`, `Ready`, `In progress`, `Done`.
- Label lab work with its lab label (for example `lab5`), so lab evidence stays separate from feature work.
- Comments and conversation history are issue comments.

## When a skill says "publish to the issue tracker"

`gh issue create --repo beambeambeam/claimflow`, then `gh project item-add 5 --owner beambeambeam --url <issue URL>`.

## When a skill says "fetch the relevant ticket"

`gh issue view <number> --repo beambeambeam/claimflow`. The user normally passes the issue number.

## Wayfinding operations

Used by `/wayfinder`. The **map** is one issue; each **child** ticket is another issue.

- **Map**: an issue whose body holds the Notes, Decisions-so-far, and Fog sections.
- **Child ticket**: an issue with the question in the body and the map linked under Related work. Its type (`research`, `prototype`, `grilling`, `task`) goes in the title prefix.
- **Blocking**: a `Blocked by #N` line. A ticket is unblocked when every issue it lists is closed.
- **Frontier**: open, unblocked, unassigned child issues; the lowest number wins.
- **Claim**: assign yourself and set Status to `In progress` before any work.
- **Resolve**: post the answer as a comment headed `## Answer`, close the issue, then add a context pointer (gist + link) to the map's Decisions-so-far.

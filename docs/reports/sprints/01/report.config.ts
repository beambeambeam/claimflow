import type { SprintConfig } from "../../scripts/lib/sprint.ts";

// Sprint 1 report (Deliverable 1): what the shared scripts capture and list.
const config: SprintConfig = {
  board: "https://github.com/users/beambeambeam/projects/5",
  contracts: "docs/sprints/sprint-1",
  docs: {
    "adr-0001": "docs/adr/0001-finance-officer-reviews-department-head.md",
    dod: "docs/sprints/sprint-1/definition-of-done.md",
    f01: "docs/sprints/sprint-1/F-01-authentication.md",
    f02: "docs/sprints/sprint-1/F-02-submit-claim.md",
    f03: "docs/sprints/sprint-1/F-03-my-claims.md",
    f04: "docs/sprints/sprint-1/F-04-review-decision.md",
    features: "docs/features.md",
    plan: "docs/sprints/plan.md",
    srs: "docs/srs.md",
    "system-sds": "docs/system-sds.md",
    traceability: "docs/traceability.md",
  },
  manualChecks: {
    "F01-TEST-18": () =>
      String.raw`Manual check on 29fd6fb (code identical to the release): no tracked \texttt{.env} file; gitleaks over the commits of \texttt{sprint-1} reports 11 hits, all Go module checksums in \path{apps/api/*.sum}; every \texttt{.env.schema} variable is in \texttt{.env.example}. pass`,
    "F04-TEST-19": (commit) =>
      `Manual restart check; recorded pass in the F-04 contract (d96f612, 29fd6fb). Rerun at ${commit}: API and PostgreSQL container restarted with the volume kept, the requester's claims, items and history read through the API were byte-identical.`,
  },
  parts: {
    "part-a-srs": [["srs", "Software Requirements Specification"]],
    "part-b-features": [
      ["features", "Feature inventory"],
      ["traceability", "Requirements traceability matrix"],
    ],
    "part-c-sds": [
      ["system-sds", "System-level SDS"],
      ["adr-0001", "ADR 0001"],
    ],
    "part-d-plan": [["plan", "Three-sprint plan"]],
    "part-e-contracts": [
      ["f01", "F-01 contract"],
      ["f02", "F-02 contract"],
      ["f03", "F-03 contract"],
      ["f04", "F-04 contract"],
      ["dod", "Product Definition of Done"],
    ],
  },
  // Chain 1: issue #27 -> PR #59, author Kiatisakk, reviewer beambeambeam.
  // Chain 2: issue #25 -> PR #56, author beambeambeam, reviewer Kiatisakk.
  // Release: PR #69, header only. Ids come from
  //   gh api repos/beambeambeam/claimflow/pulls/<n>/reviews    (id)
  //   gh api repos/beambeambeam/claimflow/pulls/<n>/comments   (id)
  reviews: [
    { anchor: null, group: "pr59", name: "pr59-header", pr: 59 },
    {
      anchor: "pullrequestreview-5896250167",
      group: "pr59",
      name: "pr59-review-comments",
      pr: 59,
    },
    {
      anchor: "discussion_r4567214348",
      group: "pr59",
      name: "pr59-thread-fails-open",
      pr: 59,
    },
    {
      anchor: "pullrequestreview-5896430284",
      group: "pr59",
      name: "pr59-approved",
      pr: 59,
    },
    { anchor: null, group: "pr56", name: "pr56-header", pr: 56 },
    {
      anchor: "pullrequestreview-5890718551",
      group: "pr56",
      name: "pr56-changes-requested",
      pr: 56,
    },
    {
      anchor: "discussion_r4563077254",
      group: "pr56",
      name: "pr56-thread-auth-in-handler",
      pr: 56,
    },
    {
      anchor: "pullrequestreview-5892813259",
      group: "pr56",
      name: "pr56-approved",
      pr: 56,
    },
    { anchor: null, group: "pr69", name: "pr69-header", pr: 69 },
  ],
  tickets: [30, 31, 32, 33, 34],
};

export default config;

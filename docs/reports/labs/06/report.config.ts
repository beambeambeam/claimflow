import type { SprintConfig } from "../../scripts/lib/sprint.ts";

// Lab 6 report (Sprint 1 UAT and defect management). Only render-reviews
// reads this config; the parts are written by hand in content/.
const config: SprintConfig = {
  docs: {},
  parts: {},
  // Issue texts are quoted in content/, not captured: the issue page layout
  // has none of the anchors render-reviews follows.
  // #86 -> PR #92 and #87 -> PR #91, author beambeambeam, reviewer Kiatisakk.
  // Release: PR #93. Ids come from
  //   gh api repos/beambeambeam/claimflow/pulls/<n>/reviews    (id)
  reviews: [
    { anchor: null, group: "pr92", name: "pr92-header", pr: 92 },
    {
      anchor: "pullrequestreview-6003577673",
      group: "pr92",
      name: "pr92-changes-requested",
      pr: 92,
    },
    {
      anchor: "pullrequestreview-6003999150",
      group: "pr92",
      name: "pr92-approved",
      pr: 92,
    },
    { anchor: null, group: "pr91", name: "pr91-header", pr: 91 },
    {
      anchor: "pullrequestreview-6003577468",
      group: "pr91",
      name: "pr91-changes-requested",
      pr: 91,
    },
    {
      anchor: "pullrequestreview-6004085143",
      group: "pr91",
      name: "pr91-approved",
      pr: 91,
    },
    { anchor: null, group: "pr93", name: "pr93-header", pr: 93 },
    {
      anchor: "pullrequestreview-6005457679",
      group: "pr93",
      name: "pr93-approved",
      pr: 93,
    },
  ],
};

export default config;

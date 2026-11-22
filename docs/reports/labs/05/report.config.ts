import type { SprintConfig } from "../../scripts/lib/sprint.ts";

// Lab 5 evidence index. Only `pdf` and `identity` run on it; its figures are the submitted
// screenshots, so it captures nothing and lists no parts, reviews, contracts, or tickets.
const config: SprintConfig = {
  board: "",
  contracts: "",
  docs: {},
  manualChecks: {},
  parts: {},
  reviews: [],
  tickets: [],
};

export default config;

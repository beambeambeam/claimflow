// NFR-07 timed run (F-10 §1.9): sends the main API requests for a number of rounds and prints a Markdown table of the
// response times. The exit status is non-zero when the overall p95 is 1 s or more, or when the run cannot start.
//
//   SEED_USER_PASSWORD=... bun run perf -- --base-url https://claimflow.example --rounds 20 > docs/perf/nfr-07-<tag>.md
//
// The seed users share SEED_USER_PASSWORD. It is read from the environment only and never printed. The database needs
// the base seed and `seed:load`, and at least `--rounds` SUBMITTED load claims assigned to rev.malee: each round
// approves one of them, so reset the database and load again before the next run.
import { parseArgs } from "node:util";

const API_PATH = "/api/v1";
const P95_LIMIT_MS = 1000;
const LOAD_CLAIM_PREFIX = "CLM-LOAD-";
const PAGE_SIZE = 100;

const NOT_ENOUGH =
  "Not enough submitted load claims to approve; reset the database and run seed:load again.";

type Username = "req.anan" | "rev.malee" | "fin.wanida";

// The request types, in the order of a round. The approve is the only one a warm-up round leaves out.
const TYPES = [
  "POST /auth/login (req.anan)",
  "POST /auth/login (rev.malee)",
  "POST /auth/login (fin.wanida)",
  "GET /auth/me",
  "GET /claims?scope=mine",
  "GET /claims?scope=review",
  "GET /claims?scope=finance",
  "GET /claims?scope=all",
  "GET /claims?scope=all (status, category, range, q)",
  "GET /claims/{id}",
  "POST /claims/{id}/approve",
  "GET /reports/summary",
  "GET /reports/summary (range)",
] as const;

type RequestType = (typeof TYPES)[number];

const [
  LOGIN_ANAN,
  LOGIN_MALEE,
  LOGIN_WANIDA,
  ME,
  MINE,
  REVIEW,
  FINANCE,
  ALL,
  FILTERED,
  GET_CLAIM,
  APPROVE,
  SUMMARY,
  SUMMARY_RANGE,
] = TYPES;

interface Reply {
  body: string;
  etag: string | null;
  setCookie: string | null;
}

const { values: args } = parseArgs({
  options: {
    "base-url": { type: "string" },
    rounds: { default: "20", type: "string" },
  },
});

const baseUrl = args["base-url"]?.replace(/\/+$/u, "") ?? "";
const rounds = Number(args.rounds);
const password = process.env.SEED_USER_PASSWORD ?? "";

if (baseUrl === "" || !Number.isInteger(rounds) || rounds < 1) {
  console.error("Usage: bun run perf -- --base-url <url> [--rounds <n>]");
  process.exit(2);
}

if (password === "") {
  console.error("SEED_USER_PASSWORD is not set.");
  process.exit(2);
}

const sessions = new Map<Username, string>();
const samples = new Map<RequestType, number[]>(TYPES.map((type) => [type, []]));

interface ClaimPage {
  data: { claimNo: string; id: string }[];
  total: number;
}

// parsePage and parseHealth read the JSON bodies of GET /claims and GET /health. openapi.yaml fixes both shapes, and a
// body of another shape fails the run at the next request.
const parsePage = (body: string): ClaimPage =>
  // SAFETY: ClaimPage lists the fields of the ClaimPage schema that this script reads.
  // eslint-disable-next-line typescript/no-unsafe-type-assertion
  JSON.parse(body) as ClaimPage;

const parseHealth = (body: string): { version: string } =>
  // SAFETY: the Health schema requires a string version.
  // eslint-disable-next-line typescript/no-unsafe-type-assertion
  JSON.parse(body) as { version: string };

// send makes one request and reads the whole body. It returns the time from send to the last body byte, and fails on a
// status outside 2xx, so a broken run never looks fast.
const send = async (
  method: string,
  path: string,
  options: { body?: unknown; etag?: string; user?: Username } = {}
): Promise<{ ms: number; reply: Reply }> => {
  const headers = new Headers();

  if (options.user !== undefined) {
    headers.set("cookie", sessions.get(options.user) ?? "");
  }

  if (options.body !== undefined) {
    headers.set("content-type", "application/json");
  }

  if (options.etag !== undefined) {
    headers.set("if-match", options.etag);
  }

  const start = performance.now();

  const response = await fetch(`${baseUrl}${API_PATH}${path}`, {
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    headers,
    method,
  });

  const body = await response.text();
  const ms = performance.now() - start;

  if (!response.ok) {
    throw new Error(
      `${method} ${path} answered ${response.status}: ${body.slice(0, 300)}`
    );
  }

  return {
    ms,
    reply: {
      body,
      etag: response.headers.get("etag"),
      setCookie: response.headers.getSetCookie()[0] ?? null,
    },
  };
};

const record = (type: RequestType, ms: number, isTimed: boolean) => {
  if (isTimed) {
    samples.get(type)?.push(ms);
  }
};

const login = async (user: Username, type: RequestType, isTimed: boolean) => {
  const { ms, reply } = await send("POST", "/auth/login", {
    body: { password, username: user },
  });

  const cookie = reply.setCookie?.split(";")[0] ?? "";

  if (cookie === "") {
    throw new Error(`Login of ${user} set no session cookie.`);
  }

  sessions.set(user, cookie);
  record(type, ms, isTimed);
};

const get = async (
  type: RequestType,
  isTimed: boolean,
  path: string,
  user: Username
) => {
  const { ms, reply } = await send("GET", path, { user });

  record(type, ms, isTimed);

  return reply;
};

const FILTERS = new URLSearchParams({
  category: "SUPPLIES",
  from: "2026-07-01",
  q: "supplies",
  scope: "all",
  status: "APPROVED",
  to: "2026-11-30",
});

const RANGE = new URLSearchParams({ from: "2026-07-01", to: "2026-11-30" });

// round sends every request type once, in the order of the contract. A warm-up round (isTimed false) leaves out the
// approve and is not counted. claimId is the claim that GET /claims/{id} reads and the approve decides.
const round = async (claimId: string, isTimed: boolean) => {
  await login("req.anan", LOGIN_ANAN, isTimed);
  await login("rev.malee", LOGIN_MALEE, isTimed);
  await login("fin.wanida", LOGIN_WANIDA, isTimed);

  await get(ME, isTimed, "/auth/me", "req.anan");
  await get(MINE, isTimed, "/claims?scope=mine", "req.anan");
  await get(REVIEW, isTimed, "/claims?scope=review", "rev.malee");
  await get(FINANCE, isTimed, "/claims?scope=finance", "fin.wanida");
  await get(ALL, isTimed, "/claims?scope=all", "fin.wanida");
  await get(FILTERED, isTimed, `/claims?${FILTERS}`, "fin.wanida");

  const claim = await get(
    GET_CLAIM,
    isTimed,
    `/claims/${claimId}`,
    "rev.malee"
  );

  if (isTimed) {
    if (claim.etag === null) {
      throw new Error(`GET /claims/${claimId} sent no ETag.`);
    }

    const { ms } = await send("POST", `/claims/${claimId}/approve`, {
      etag: claim.etag,
      user: "rev.malee",
    });

    record(APPROVE, ms, true);
  }

  await get(SUMMARY, isTimed, "/reports/summary", "fin.wanida");
  await get(SUMMARY_RANGE, isTimed, `/reports/summary?${RANGE}`, "fin.wanida");
};

// submittedLoadClaims lists the SUBMITTED load claims assigned to rev.malee, oldest submission first. It walks every
// page of her review list, so the count does not depend on the page size.
const submittedLoadClaims = async (): Promise<string[]> => {
  const ids: string[] = [];
  let seen = 0;
  let total = 1;

  for (let page = 1; seen < total; page += 1) {
    // The pages are read one after the other, so the order of the list holds.
    // eslint-disable-next-line no-await-in-loop
    const { reply } = await send(
      "GET",
      `/claims?scope=review&page=${page}&pageSize=${PAGE_SIZE}`,
      {
        user: "rev.malee",
      }
    );

    const { data, total: count } = parsePage(reply.body);

    ids.push(
      ...data
        .filter((claim) => claim.claimNo.startsWith(LOAD_CLAIM_PREFIX))
        .map((claim) => claim.id)
    );
    seen += PAGE_SIZE;
    total = count;
  }

  return ids;
};

// percentile is the nearest-rank percentile of numbers, in ms.
const percentile = (numbers: number[], p: number): number => {
  // The copy is sorted, not the caller's array; the root tsconfig targets ES2022, which has no toSorted.
  // eslint-disable-next-line unicorn/no-array-sort
  const sorted = [...numbers].sort((a, b) => a - b);

  return sorted[Math.max(0, Math.ceil((p / 100) * sorted.length) - 1)] ?? 0;
};

const fixed = (value: number): string => value.toFixed(1);

const row = (name: string, numbers: number[]): string =>
  `| ${name} | ${numbers.length} | ${fixed(percentile(numbers, 50))} | ${fixed(percentile(numbers, 95))} | ${fixed(Math.max(...numbers))} |`;

const main = async (): Promise<number> => {
  const health = await send("GET", "/health");

  await login("rev.malee", LOGIN_MALEE, false);

  const claimIds = await submittedLoadClaims();

  if (claimIds.length < rounds) {
    console.error(NOT_ENOUGH);

    return 1;
  }

  // The warm-up reads the first claim and approves nothing. Its results are dropped.
  await round(claimIds[0] ?? "", false);

  for (const claimId of claimIds.slice(0, rounds)) {
    // Each round approves a claim, which changes what the next round lists, so the rounds run one after the other.
    // eslint-disable-next-line no-await-in-loop
    await round(claimId, true);
  }

  const all = [...samples.values()].flat();
  const overallP95 = percentile(all, 95);
  const pass = overallP95 < P95_LIMIT_MS;

  console.log("# NFR-07 timed run\n");
  console.log(`- Base URL: ${baseUrl}`);
  console.log(`- Time: ${new Date().toISOString()}`);
  console.log(`- API version: ${parseHealth(health.reply.body).version}`);
  console.log("- Percentiles: p50 and p95 by nearest rank");
  console.log(
    `- Rounds: ${rounds} timed, after 1 warm-up round that is not counted`
  );
  console.log(
    `- Result: ${pass ? "PASS" : "FAIL"}, overall p95 ${fixed(overallP95)} ms (limit ${P95_LIMIT_MS} ms)\n`
  );
  console.log("| Request | Count | p50 (ms) | p95 (ms) | max (ms) |");
  console.log("| --- | ---: | ---: | ---: | ---: |");

  for (const type of TYPES) {
    console.log(row(type, samples.get(type) ?? []));
  }

  console.log(row("**Overall**", all));

  return pass ? 0 : 1;
};

try {
  process.exitCode = await main();
} catch (error) {
  console.error(error instanceof Error ? error.message : error);
  process.exitCode = 1;
}

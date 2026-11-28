// Generate content/data/ac-table.tex: planned test -> actual test -> result.
//
// Sources, all read from this checkout (so run it on the commit you report):
//   - the test tables of the sprint contracts give each planned TEST id, its AC
//     and its level;
//   - Go tests whose name carries the id (TestF02TEST09MealsLimit), plus those
//     docs/traceability.md names for an id;
//   - web tests (apps/web/{src,e2e}) whose title contains the id;
//   - Go results from <sprint>/build/go.json (bun run go-test). Web, E2E and
//     manual results are not run here; flags record them;
//   - a planned test with no actual test names the open issue that lists it.
//
//   bun run gen-ac-table 01 --commit ac5bce5 --e2e-passed --web-passed [--web-fail <title>]
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import { parseArgs } from "node:util";

import { GoEvent, parseJson } from "./lib/data.ts";
import { SLUG } from "./lib/github.ts";
import { tex } from "./lib/latex.ts";
import { groups } from "./lib/match.ts";
import { ROOT } from "./lib/paths.ts";
import { loadSprint } from "./lib/sprint.ts";

type Kind = "go" | "web";

interface Actual {
  file: string;
  name: string;
  kind: Kind;
}

const read = (file: string) => readFileSync(file, "utf-8");
const rel = (file: string) => path.relative(ROOT, file).replaceAll("\\", "/");

const add = (map: Map<string, Actual[]>, id: string, a: Actual) => {
  const list = map.get(id) ?? [];

  if (!list.some((x) => x.file === a.file && x.name === a.name)) {
    list.push(a);
  }

  map.set(id, list);
};

const walk = (dir: string, keep: (f: string) => boolean): string[] =>
  existsSync(dir)
    ? readdirSync(dir, { recursive: true, withFileTypes: true })
        .filter((e) => e.isFile() && keep(e.name))
        .map((e) => path.join(e.parentPath, e.name))
    : [];

const { values, positionals } = parseArgs({
  allowPositionals: true,
  options: {
    commit: { default: "HEAD", type: "string" },
    // The E2E specs ran on a local stack at --commit and all passed.
    "e2e-passed": { default: false, type: "boolean" },
    // A Vitest test whose title contains this fails when its file runs alone.
    "web-fail": { default: [], multiple: true, type: "string" },
    // The Vitest suite ran at --commit; each file passes when run alone.
    "web-passed": { default: false, type: "boolean" },
  },
});

const sprint = await loadSprint(positionals[0]);
const { commit } = values;

const {
  ciCheck,
  implementation,
  release,
  contracts: contractsConfig,
  manualChecks = {},
  tickets = [],
} = sprint.config;

if (contractsConfig === undefined) {
  throw new Error(`no contracts folder in ${sprint.name}/report.config.ts`);
}

const contractDirs = [contractsConfig].flat();

// TEST id -> [ac ids, level].
const plan = new Map<string, [string, string]>();

// AC id -> the requirements it covers, from the contracts' AC tables.
const acs = new Map<string, string>();

for (const dir of contractDirs) {
  const contracts = path.join(ROOT, dir);

  for (const f of readdirSync(contracts)
    .filter((n) => /^F-0.*\.md$/u.test(n))
    .toSorted()) {
    for (const line of read(path.join(contracts, f)).split("\n")) {
      const {
        id,
        ac = "",
        level = "",
      } = groups(
        /^\|\s*(?<id>F\d\d-TEST-\d\d)\s*\|\s*(?<ac>[^|]+)\|\s*(?<level>[^|]+)\|/u.exec(
          line
        )
      );

      if (id !== undefined) {
        plan.set(id, [ac.trim(), level.trim()]);
      }

      const row = groups(
        /^\|\s*(?<acId>F\d\d-AC-\d\d)\s*\|\s*(?<req>[^|]+)\|/u.exec(line)
      );

      if (row.acId !== undefined) {
        acs.set(row.acId, (row.req ?? "").trim());
      }
    }
  }
}

const actual = new Map<string, Actual[]>();

for (const file of walk(path.join(ROOT, "apps/api"), (n) =>
  n.endsWith("_test.go")
)) {
  for (const func of read(file).matchAll(/^func (?<name>Test\w+)\(/gmu)) {
    const { name = "" } = groups(func);

    const { feature, test } = groups(
      /^TestF(?<feature>\d\d)TEST(?<test>\d\d)/u.exec(name)
    );

    if (feature !== undefined && test !== undefined) {
      add(actual, `F${feature}-TEST-${test}`, {
        file: rel(file),
        kind: "go",
        name,
      });
    }
  }
}

// Sprint 1 rows say `F01-TEST-15: `apps/api/server_test.go › TestProblemResponses``;
// Sprint 2 rows are table cells: `| F05-TEST-26 | F04-AC-19 (seed) | `file` · `Name` |`.
// Accepted only when that function exists in that file.
const traceability = read(path.join(ROOT, "docs/traceability.md"));

const named = [
  ...traceability.matchAll(
    /(?<id>F\d\d-TEST-\d\d)(?: \([^)]*\))?:? `(?<file>[^`]+?\.go) › (?<name>Test\w+)`/gu
  ),
  ...traceability.matchAll(
    /\|\s*(?<id>F\d\d-TEST-\d\d)\s*\|[^|\n]*\|\s*`(?<file>[^`]+?\.go)` · `(?<name>Test\w+)[/`]/gu
  ),
];

for (const row of named) {
  const { id = "", file = "", name = "" } = groups(row);
  const full = path.join(ROOT, file);

  if (
    existsSync(full) &&
    new RegExp(`^func ${name}\\(`, "mu").test(read(full))
  ) {
    add(actual, id, { file, kind: "go", name });
  }
}

for (const sub of ["src", "e2e"]) {
  for (const file of walk(path.join(ROOT, "apps/web", sub), (n) =>
    /\.tsx?$/u.test(n)
  )) {
    for (const literal of read(file).matchAll(
      /["'`](?<title>[^"'`\n]*?(?<id>F\d\d-TEST-\d\d)[^"'`\n]*)["'`]/gu
    )) {
      const { id = "", title = "" } = groups(literal);
      add(actual, id, { file: rel(file), kind: "web", name: title });
    }
  }
}

// Go test function -> pass | fail | skip (top-level only).
const results = new Map<string, string>();
const goJson = path.join(sprint.build, "go.json");

if (existsSync(goJson)) {
  for (const line of read(goJson).split("\n")) {
    if (line.trim() === "") {
      continue;
    }

    const e = parseJson(line, GoEvent);

    if (
      e.Test !== undefined &&
      !e.Test.includes("/") &&
      ["pass", "fail", "skip"].includes(e.Action)
    ) {
      results.set(e.Test, e.Action);
    }
  }
}

// TEST id -> issue number, from the bodies of the sprint's open issues.
const owner = new Map<string, number>();

for (const n of tickets) {
  let body = "";

  try {
    body = execFileSync(
      "gh",
      ["issue", "view", String(n), "-R", SLUG, "--json", "body", "-q", ".body"],
      { encoding: "utf-8" }
    );
  } catch {
    continue;
  }

  for (const listed of body.matchAll(/(?<id>F\d\d-TEST-\d\d) –/gu)) {
    const { id = "" } = groups(listed);

    if (!owner.has(id)) {
      owner.set(id, n);
    }
  }
}

const result = (list: Actual[], level: string) => {
  const gos = list.flatMap((a) =>
    a.kind === "go" ? [results.get(a.name)] : []
  );

  const webs = list.flatMap((a) => (a.kind === "web" ? [a.name] : []));

  if (gos.length > 0 && results.size === 0) {
    return String.raw`\todo{run Go tests}`;
  }

  if (gos.some((r) => r !== "pass")) {
    return String.raw`\textbf{FAIL/SKIP: see output}`;
  }

  const parts = gos.length > 0 ? [`pass (${gos.length} run)`] : [];

  if (webs.length > 0) {
    if (level === "E2E" && values["e2e-passed"]) {
      parts.push(`pass (E2E on a local stack at ${commit}, not in CI)`);
    } else if (level !== "E2E" && values["web-passed"]) {
      parts.push(
        webs.some((w) => values["web-fail"].some((x) => w.includes(x)))
          ? `pass in CI at ${commit}; locally the first case fails on the author's machine (F.3)`
          : `pass (Vitest, CI at ${commit})`
      );
    } else {
      parts.push(
        `recorded pass, ${level === "E2E" ? "on a local stack, not in CI" : "not run here"}`
      );
    }
  }

  return parts.join("; ");
};

const NL = String.raw`\newline `;

const cellActual = (list: Actual[]) => {
  const byFile = new Map<string, string[]>();

  for (const a of list) {
    const f = a.file.replace("apps/", "");
    byFile.set(f, [
      ...(byFile.get(f) ?? []),
      a.kind === "go" ? String.raw`\seqsplit{${tex(a.name)}}` : tex(a.name),
    ]);
  }

  return [...byFile]
    .map(
      ([f, names]) =>
        String.raw`\path{${f}}` +
        NL +
        names.map((n) => String.raw`\hspace*{1em}${n}`).join(NL)
    )
    .join(NL);
};

const auto: [string, string, string, Actual[]][] = [];
const manual: [string, string, string][] = [];
const missing: [string, string, string][] = [];

for (const id of [...plan.keys()].toSorted()) {
  const [ac = "", level = ""] = plan.get(id) ?? [];
  const list = actual.get(id) ?? [];

  if (level.startsWith("Manual")) {
    manual.push([id, ac, level]);
  } else if (list.length === 0) {
    missing.push([id, ac, level]);
  } else {
    auto.push([id, ac, level, list]);
  }
}

const table = (title: string, cols: string, head: string, rows: string[]) => [
  String.raw`\subsubsection*{${title}}`,
  String.raw`{\footnotesize\setlength{\tabcolsep}{3pt}`,
  String.raw`\begin{longtable}{@{}${cols}@{}}`,
  String.raw`\toprule ${head} \\ \midrule \endhead`,
  ...rows,
  String.raw`\bottomrule\end{longtable}}`,
  "",
];

const L = [
  `% Generated by scripts/gen-ac-table.ts at ${commit}. Do not edit.`,
  "",
  ...table(
    `Automated tests (${auto.length} planned tests with an actual test)`,
    "p{1.9cm}p{2.3cm}p{1.8cm}p{6.6cm}p{3.6cm}",
    String.raw`\textbf{Planned test} & \textbf{AC} & \textbf{Level} & \textbf{Actual test (file, name)} & \textbf{Result}`,
    auto.map(
      ([id, ac, level, list]) =>
        String.raw`${id} & ${tex(ac)} & ${tex(level)} & ${cellActual(list)} & ${result(list, level)} \\`
    )
  ),
  ...table(
    `Manual and CI-recorded checks (${manual.length})`,
    "p{1.9cm}p{2.3cm}p{1.8cm}p{10.2cm}",
    String.raw`\textbf{Planned test} & \textbf{AC} & \textbf{Level} & \textbf{Status}`,
    manual.map(([id, ac, level]) => {
      const status = manualChecks[id]?.(commit) ?? ciCheck?.(commit);

      if (status === undefined) {
        throw new Error(
          `${id}: no manualChecks entry and no ciCheck in ${sprint.name}/report.config.ts`
        );
      }

      return String.raw`${id} & ${tex(ac)} & ${tex(level)} & ${status} \\`;
    })
  ),
  ...table(
    `Planned but not implemented yet (${missing.length})`,
    "p{1.9cm}p{4.2cm}p{2.2cm}p{7.9cm}",
    String.raw`\textbf{Planned test} & \textbf{AC} & \textbf{Level} & \textbf{Result}`,
    missing.map(([id, ac, level]) => {
      const n = owner.get(id);

      const who =
        n === undefined
          ? String.raw`not implemented yet; \todo{ticket number}`
          : String.raw`not implemented yet (ticket \#${n}, open)`;

      return String.raw`${id} & ${tex(ac)} & ${tex(level)} & ${who} \\`;
    })
  ),
];

const out = path.join(sprint.data, "ac-table.tex");

writeFileSync(out, `${L.slice(0, -1).join("\n")}\n`);

console.log(
  `${out}: ${auto.length} automated, ${manual.length} manual, ${missing.length} not implemented`
);

const unticketed = missing.flatMap(([id]) => (owner.has(id) ? [] : [id]));

if (unticketed.length > 0) {
  console.log("no ticket found for:", unticketed.join(", "));
}

const passed = (text: string) => text.startsWith("pass");

// Completion table: one row per AC with its implementation, its planned tests,
// their result, and the release that accepted it.
if (implementation !== undefined && release !== undefined) {
  const rows = [...acs.keys()].toSorted().map((ac) => {
    const tests = plan
      .entries()
      .filter(([, [covers]]) => covers.split(/[\s,]+/u).includes(ac))
      .map(([id, [, level]]) => {
        const list = actual.get(id) ?? [];
        const manualCheck = level.startsWith("Manual");

        let state = "none";

        if (list.length > 0) {
          state = result(list, level);
        } else if (manualCheck) {
          state = "manual";
        }

        return { id, level, state };
      })
      .toArray();

    const ok = tests.filter((t) => passed(t.state)).length;
    let summary = String.raw`\textbf{no test}`;

    if (tests.length > 0 && ok === tests.length) {
      summary = `pass (${ok} of ${tests.length})`;
    } else if (tests.length > 0) {
      const rest = tests
        .values()
        .filter((t) => !passed(t.state))
        .map((t) => `${t.id} ${tex(t.state)}`)
        .toArray()
        .join("; ");

      summary = `${ok} of ${tests.length} pass; ${rest}`;
    }

    const levels = [...new Set(tests.map((t) => t.level))].join(", ");

    return [
      ac,
      tex(acs.get(ac) ?? ""),
      tex(implementation[ac] ?? "NOT MAPPED"),
      tests.map((t) => t.id.replace("-TEST-", "-T")).join(", ") +
        (levels === "" ? "" : ` (${tex(levels)})`),
      summary,
      tex(release),
    ];
  });

  const body = [
    `% Generated by scripts/gen-ac-table.ts at ${commit}. Do not edit.`,
    String.raw`{\footnotesize\setlength{\tabcolsep}{3pt}`,
    String.raw`\begin{longtable}{@{}p{1.6cm}p{1.9cm}p{4.1cm}p{3.9cm}p{2.9cm}p{1.6cm}@{}}`,
    String.raw`\toprule \textbf{AC} & \textbf{Covers} & \textbf{Implementation} & \textbf{Executed verification} & \textbf{Result} & \textbf{Accepted in} \\ \midrule \endhead`,
    ...rows.map((r) => `${r.join(" & ")} \\\\`),
    String.raw`\bottomrule\end{longtable}}`,
  ];

  writeFileSync(
    path.join(sprint.data, "completion-table.tex"),
    `${body.join("\n")}\n`
  );
  console.log(`completion-table.tex: ${rows.length} AC`);
}

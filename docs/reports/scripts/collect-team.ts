// Collect the contribution data of F.5 from GitHub, through gh.
//
//   bun run collect-team 01      writes <sprint>/content/data/team.json
//
// Per merged pull request: author, merger, base, and every review by someone
// other than the author with its state. Per closed issue: assignees.
import { execFileSync } from "node:child_process";
import { writeFileSync } from "node:fs";
import path from "node:path";

import { z } from "zod";

import { parseJson } from "./lib/data.ts";
import { SLUG } from "./lib/github.ts";
import { loadSprint } from "./lib/sprint.ts";

const gh = <T>(schema: z.ZodType<T>, ...args: string[]) =>
  parseJson(
    execFileSync("gh", args, { encoding: "utf-8", maxBuffer: 64e6 }),
    schema
  );

const Login = z.object({ login: z.string() });

const Prs = z.array(
  z.object({
    author: Login,
    baseRefName: z.string(),
    mergedAt: z.string(),
    mergedBy: Login.nullable(),
    number: z.number(),
    title: z.string(),
  })
);

const Reviews = z.array(
  z.object({
    state: z.string(),
    user: z.object({ login: z.string(), type: z.string() }),
  })
);

const Issues = z.array(
  z.object({
    assignees: z.array(Login),
    closedAt: z.string(),
    number: z.number(),
    title: z.string(),
  })
);

const sprint = await loadSprint(process.argv[2]);

const prs = gh(
  Prs,
  "pr",
  "list",
  "-R",
  SLUG,
  "--state",
  "merged",
  "--limit",
  "200",
  "--json",
  "number,title,author,baseRefName,mergedBy,mergedAt"
);

const out = {
  issues: gh(
    Issues,
    "issue",
    "list",
    "-R",
    SLUG,
    "--state",
    "closed",
    "--limit",
    "200",
    "--json",
    "number,title,assignees,closedAt"
  )
    .toSorted((a, b) => a.number - b.number)
    .map((i) => ({
      assignees: i.assignees.map((a) => a.login),
      closed_at: i.closedAt,
      number: i.number,
      title: i.title,
    })),
  prs: prs
    .toSorted((a, b) => a.number - b.number)
    .map((pr) => {
      const who = new Map<string, Set<string>>();

      for (const r of gh(
        Reviews,
        "api",
        `repos/${SLUG}/pulls/${pr.number}/reviews`,
        "--paginate"
      )) {
        if (r.user.login !== pr.author.login && r.user.type !== "Bot") {
          who.set(
            r.user.login,
            (who.get(r.user.login) ?? new Set()).add(r.state)
          );
        }
      }

      return {
        author: pr.author.login,
        base: pr.baseRefName,
        merged_at: pr.mergedAt,
        merged_by: pr.mergedBy?.login ?? null,
        number: pr.number,
        reviews: Object.fromEntries(
          [...who].map(([k, v]) => [k, [...v].toSorted()])
        ),
        title: pr.title,
      };
    }),
};

const file = path.join(sprint.data, "team.json");

writeFileSync(file, `${JSON.stringify(out, null, 1)}\n`);

console.log(
  `${file}: ${out.prs.length} merged PRs, ${out.issues.length} closed issues`
);

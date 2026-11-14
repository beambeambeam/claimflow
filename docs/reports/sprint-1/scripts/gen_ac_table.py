"""
Generate content/data/ac-table.tex: planned test -> actual test -> result.

Sources, all read from this checkout (so run it on the commit you report):

  * the Software Test Specification tables in docs/sprints/sprint-1/F-0N-*.md
    give every planned TEST ID, its AC, and its level;
  * the Go sources (apps/api/**/*_test.go): a test whose name carries the ID,
    e.g. TestF02TEST09MealsLimit, is the actual test for F02-TEST-09;
  * the web sources (apps/web/{src,e2e}): a test title that contains the ID,
    e.g. "F01-TEST-20: ...", is the actual test;
  * results of the Go tests come from a `go test -json` run you pass with
    --gojson; web, E2E and manual results are NOT run by this script, so they
    are labelled "recorded" and point at docs/traceability.md;
  * a planned test with no actual test is "not implemented yet" and names the
    open GitHub issue whose body lists it (fetched with `gh issue view`).

    cd apps/api && go test -json -count=1 ./... > /tmp/go.json
    python scripts/gen_ac_table.py --gojson /tmp/go.json --commit a60fcf1
"""

import argparse
import json
import re
import subprocess
from pathlib import Path

HERE = Path(__file__).resolve().parent
REPORT = HERE.parent
ROOT = REPORT.parents[2]
OUT = REPORT / "content" / "data" / "ac-table.tex"
CONTRACTS = sorted((ROOT / "docs/sprints/sprint-1").glob("F-0*.md"))
TICKETS = (30, 31, 32, 33, 34)   # open Sprint 1 issues that list planned tests


def tex(s):
    """Escape for LaTeX, keep code identifiers breakable."""
    s = s.replace("\\", "\\textbackslash{}")
    for a, b in (("&", r"\&"), ("%", r"\%"), ("_", r"\_"), ("#", r"\#"),
                 ("$", r"\$"), ("{", r"\{"), ("}", r"\}"), ("~", r"\~{}"),
                 ("^", r"\^{}"), ("›", ">")):
        s = s.replace(a, b)
    return s


def planned():
    """TEST id -> (ac ids, level)."""
    rows = {}
    for f in CONTRACTS:
        for line in f.read_text(encoding="utf-8").splitlines():
            m = re.match(r"\|\s*(F\d\d-TEST-\d\d)\s*\|\s*([^|]+)\|\s*([^|]+)\|", line)
            if m:
                rows[m.group(1)] = (m.group(2).strip(), m.group(3).strip())
    return rows


def go_tests():
    """TEST id -> [(relative file, test function)]."""
    found = {}
    for path in (ROOT / "apps/api").rglob("*_test.go"):
        rel = path.relative_to(ROOT).as_posix()
        for name in re.findall(r"^func (Test\w+)\(", path.read_text(encoding="utf-8"), re.M):
            m = re.match(r"TestF(\d\d)TEST(\d\d)", name)
            if m:
                found.setdefault(f"F{m[1]}-TEST-{m[2]}", []).append((rel, name))
    return found


def traceability_go(found):
    """Add Go tests that traceability.md names for an ID but whose Go name has none.

    The row says `F01-TEST-15: `apps/api/server_test.go > TestProblemResponses``.
    Only accepted when that function really exists in that file.
    """
    text = (ROOT / "docs/traceability.md").read_text(encoding="utf-8")
    for tid, path, name in re.findall(
            r"(F\d\d-TEST-\d\d)(?: \([^)]*\))?:? `([^`]+?\.go) › (Test\w+)`", text):
        file = ROOT / path
        if file.is_file() and re.search(rf"^func {name}\(", file.read_text(encoding="utf-8"), re.M):
            entry = (path, name)
            if entry not in found.setdefault(tid, []):
                found[tid].append(entry)
    return found


def web_tests():
    """TEST id -> [(relative file, title)]."""
    found = {}
    lit = re.compile(r"""["'`]([^"'`\n]*?(F\d\d-TEST-\d\d)[^"'`\n]*)["'`]""")
    base = ROOT / "apps/web"
    for sub in ("src", "e2e"):
        for path in (base / sub).rglob("*"):
            if path.suffix not in (".ts", ".tsx") or not path.is_file():
                continue
            rel = path.relative_to(ROOT).as_posix()
            seen = set()
            for title, tid in lit.findall(path.read_text(encoding="utf-8")):
                if (tid, title) in seen:
                    continue
                seen.add((tid, title))
                found.setdefault(tid, []).append((rel, title))
    return found


def go_results(gojson):
    """Go test function -> pass | fail | skip (top-level only)."""
    out = {}
    if not gojson:
        return out
    for line in Path(gojson).read_text(encoding="utf-8").splitlines():
        e = json.loads(line)
        if e.get("Test") and "/" not in e["Test"] and e["Action"] in ("pass", "fail", "skip"):
            out[e["Test"]] = e["Action"]
    return out


def tickets():
    """TEST id -> issue number, from the bodies of the open Sprint 1 issues."""
    owner = {}
    for n in TICKETS:
        try:
            body = subprocess.run(
                ["gh", "issue", "view", str(n), "-R", "beambeambeam/claimflow",
                 "--json", "body", "-q", ".body"],
                capture_output=True, text=True, encoding="utf-8", check=True).stdout
        except Exception:
            continue
        for tid in re.findall(r"(F\d\d-TEST-\d\d) \u2013", body):
            owner.setdefault(tid, n)
    return owner


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--gojson", help="output of go test -json run in apps/api")
    ap.add_argument("--commit", default="HEAD")
    args = ap.parse_args()

    plan, go, web = planned(), traceability_go(go_tests()), web_tests()
    res, owner = go_results(args.gojson), tickets()

    auto, manual, missing = [], [], []
    for tid in sorted(plan):
        ac, level = plan[tid]
        actual = [(f, n, "go") for f, n in go.get(tid, [])] + \
                 [(f, t, "web") for f, t in web.get(tid, [])]
        if level.startswith("Manual"):
            manual.append((tid, ac, level, actual))
        elif not actual:
            missing.append((tid, ac, level))
        else:
            auto.append((tid, ac, level, actual))

    def result(actual, level):
        gos = [res.get(n) for f, n, k in actual if k == "go"]
        webs = [1 for f, n, k in actual if k == "web"]
        if gos and not res:
            return r"\todo{run Go tests}"
        if gos and any(r != "pass" for r in gos):
            return r"\textbf{FAIL/SKIP: see output}"
        parts = []
        if gos:
            parts.append(f"pass ({len(gos)} run)")
        if webs:
            tail = "on a local stack, not in CI" if level == "E2E" else "not run here"
            parts.append(f"recorded pass, {tail}")
        return "; ".join(parts)

    def cell_actual(actual):
        by_file = {}
        for f, n, k in actual:
            by_file.setdefault(f.replace("apps/", ""), []).append("\\seqsplit{" + tex(n) + "}" if k == "go" else tex(n))
        NL = "\\newline "
        blocks = []
        for f, names in by_file.items():
            body = NL.join(r"\hspace*{1em}" + n for n in names)
            blocks.append(r"\path{" + f + "}" + NL + body)
        return NL.join(blocks)

    L = []
    L.append(f"% Generated by scripts/gen_ac_table.py at {args.commit}. Do not edit.")
    L.append("")
    n_auto = len(auto)
    L.append(r"\subsubsection*{Automated tests (%d planned tests with an actual test)}" % n_auto)
    L.append(r"{\footnotesize\setlength{\tabcolsep}{3pt}")
    L.append(r"\begin{longtable}{@{}p{1.9cm}p{2.3cm}p{1.8cm}p{6.6cm}p{3.6cm}@{}}")
    L.append(r"\toprule \textbf{Planned test} & \textbf{AC} & \textbf{Level} & \textbf{Actual test (file, name)} & \textbf{Result} \\ \midrule \endhead")
    for tid, ac, level, actual in auto:
        L.append(f"{tid} & {tex(ac)} & {tex(level)} & {cell_actual(actual)} & {result(actual, level)} \\\\")
    L.append(r"\bottomrule\end{longtable}}")
    L.append("")
    L.append(r"\subsubsection*{Manual and CI-recorded checks (%d)}" % len(manual))
    L.append(r"{\footnotesize\setlength{\tabcolsep}{3pt}")
    L.append(r"\begin{longtable}{@{}p{1.9cm}p{2.3cm}p{1.8cm}p{10.2cm}@{}}")
    L.append(r"\toprule \textbf{Planned test} & \textbf{AC} & \textbf{Level} & \textbf{Status} \\ \midrule \endhead")
    for tid, ac, level, actual in manual:
        n = owner.get(tid)
        if tid == "F04-TEST-19":
            status = ("Manual restart check, not run yet: ticket \\#%s" % n) if n else r"\todo{ticket}"
        else:
            status = ("Manual/CI check; recorded in the F-01/01 pull request "
                      "(\\#24) per \\path{docs/traceability.md}; \\todo{release: rerun and record at the release commit}")
        L.append(f"{tid} & {tex(ac)} & {tex(level)} & {status} \\\\")
    L.append(r"\bottomrule\end{longtable}}")
    L.append("")
    L.append(r"\subsubsection*{Planned but not implemented yet (%d)}" % len(missing))
    L.append(r"{\footnotesize\setlength{\tabcolsep}{3pt}")
    L.append(r"\begin{longtable}{@{}p{1.9cm}p{4.2cm}p{2.2cm}p{7.9cm}@{}}")
    L.append(r"\toprule \textbf{Planned test} & \textbf{AC} & \textbf{Level} & \textbf{Result} \\ \midrule \endhead")
    for tid, ac, level in missing:
        n = owner.get(tid)
        feat = tid[:3]
        who = f"not implemented yet (ticket \\#{n}, open)" if n else r"not implemented yet; \todo{ticket number}"
        L.append(f"{tid} & {tex(ac)} & {tex(level)} & {who} \\\\")
    L.append(r"\bottomrule\end{longtable}}")
    OUT.write_text("\n".join(L) + "\n", encoding="utf-8")
    print(f"{OUT}: {len(auto)} automated, {len(manual)} manual, {len(missing)} not implemented")
    unticketed = [t for t, _, _ in missing if t not in owner]
    if unticketed:
        print("no ticket found for:", ", ".join(unticketed))


if __name__ == "__main__":
    main()

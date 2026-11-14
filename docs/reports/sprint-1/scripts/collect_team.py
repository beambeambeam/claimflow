"""
Collect the contribution data of F.5 from GitHub (public API through gh).

    python scripts/collect_team.py     # writes content/data/team.json

Per merged pull request: author, merger, base branch, and every review by a
person other than the author with its state. Per closed issue: assignee.
Dependabot pull requests are listed but not counted for either student.
"""
import json
import subprocess
from pathlib import Path

REPO = "beambeambeam/claimflow"
OUT = Path(__file__).resolve().parent.parent / "content" / "data" / "team.json"


def gh(*args):
    r = subprocess.run(["gh", *args], capture_output=True, text=True, encoding="utf-8", check=True)
    return json.loads(r.stdout)


def main():
    prs = gh("pr", "list", "-R", REPO, "--state", "merged", "--limit", "200",
             "--json", "number,title,author,baseRefName,mergedBy,mergedAt")
    out = {"prs": [], "issues": []}
    for pr in sorted(prs, key=lambda p: p["number"]):
        reviews = gh("api", f"repos/{REPO}/pulls/{pr['number']}/reviews", "--paginate")
        who = {}
        for r in reviews:
            login = r["user"]["login"]
            if login == pr["author"]["login"] or r["user"]["type"] == "Bot":
                continue
            who.setdefault(login, set()).add(r["state"])
        out["prs"].append({
            "number": pr["number"], "title": pr["title"], "author": pr["author"]["login"],
            "base": pr["baseRefName"], "merged_by": (pr["mergedBy"] or {}).get("login"),
            "merged_at": pr["mergedAt"],
            "reviews": {k: sorted(v) for k, v in who.items()}})
    issues = gh("issue", "list", "-R", REPO, "--state", "closed", "--limit", "200",
                "--json", "number,title,assignees,closedAt")
    for i in sorted(issues, key=lambda i: i["number"]):
        out["issues"].append({"number": i["number"], "title": i["title"],
                              "assignees": [a["login"] for a in i["assignees"]],
                              "closed_at": i["closedAt"]})
    OUT.write_text(json.dumps(out, indent=1), encoding="utf-8")
    print(f"{OUT}: {len(out['prs'])} merged PRs, {len(out['issues'])} closed issues")


if __name__ == "__main__":
    main()

# Sprint reports

LaTeX sources and the capture scripts of the ClaimFlow sprint reports. One package serves every sprint; each `sprints/NN/` folder is one report. `sprints/01/README.md` holds that report's plan and evidence index.

```
docs/reports/
├── preamble.tex           shared packages and macros
├── scripts/               capture, cut, generate, check (TypeScript, run by Bun)
├── .env.schema            names, student IDs, and demo settings the scripts need
└── sprints/
    └── 01/                Sprint 1; the scripts call it 01
        ├── main.tex           sets \sprintname and \deliverable, inputs ../../preamble
        ├── report.config.ts   everything sprint-specific the scripts read
        ├── demo.ts            the app walkthrough capture-demo runs
        ├── content/           the parts; content/data/ holds generated evidence
        └── figures/           every image the report uses, committed
```

## Setup

Works the same on Windows and macOS.

1. Install:
   - [Bun](https://bun.sh), then `bun install` at the repo root.
   - XeLaTeX: [MiKTeX](https://miktex.org) or [TeX Live](https://tug.org/texlive/) on Windows, [MacTeX](https://tug.org/mactex/) on macOS. `xelatex` must be on `PATH`. In MiKTeX Console, set **Install missing packages** to **Always**: the build runs non-interactively, so MiKTeX cannot ask.
   - Google Chrome or Microsoft Edge (the capture scripts drive the installed one).
   - [GitHub CLI](https://cli.github.com) for `collect-team` and `gen-ac-table`; Go and Docker for `go-test`.
   - Times New Roman and Courier New ship with both systems.
2. Copy `.env.example` to `.env` in this folder and fill in both students' names and IDs. `.env` is ignored by Git; the names and IDs reach the PDF only through the generated `sprints/NN/build/identity.tex`.
3. In VS Code, install the recommended extensions (LaTeX Workshop is one of them). Saving any `.tex` file under a sprint builds that sprint's `main.tex`: it runs `identity`, then XeLaTeX twice.

## Build

From `docs/reports`:

```sh
bun run pdf 01            # identity + XeLaTeX twice -> sprints/01/main.pdf
bun run check-report 01   # pending figures, TODOs, blank captures, Mermaid, overfull boxes, missing glyphs
```

The captures in `figures/` and the files in `content/data/` are committed, so the PDF builds from a clean clone without network, Docker, or Go. The final PDF needs `check-report` to print `clean`.

`check-report sprint-1 --quick` skips the pixel scans of the captures.

## Get the evidence

Every script takes the sprint number first (`01` is `sprints/01`), reads the rest from its `report.config.ts`, and writes into that folder. Rerun one when its source changes; the PDF picks up the new files on the next build.

| Evidence | Command | Writes | Needs |
| --- | --- | --- | --- |
| Repo documents (Parts A–E) | `render-docs sprint-1 --ref <sha> --label main` | `figures/docs/<name>.png` + `.json` | Chrome or Edge, network |
| Pages of those documents | `slice-tall sprint-1` | `figures/docs/<name>-pN.png` | – |
| Mermaid diagrams at 3x | `capture-diagrams sprint-1 --ref <sha> --label main` | `figures/diagrams/` | Chrome or Edge, network |
| Figure lists of Parts A–E | `gen-part-lists sprint-1` | `content/part-a…e-*.tex` | the two steps above |
| PR reviews and headers | `render-reviews sprint-1` | `figures/github/` | Chrome or Edge, network |
| Project board | `capture-board sprint-1` | `figures/github/board.png` | Chrome or Edge, network |
| Contribution table (F.5) | `collect-team sprint-1`, then `gen-team-table sprint-1` | `content/data/team.json`, `team.tex` | `gh auth login` |
| Go test results | `go-test sprint-1` | `build/go.json` | Go, Docker running |
| AC-to-test table (F.3) | `gen-ac-table sprint-1 --commit <sha> --e2e-passed --web-passed [--web-fail <title>]` | `content/data/ac-table.tex` | `build/go.json`, `gh` |
| App demos (F.2) | `capture-demo sprint-1` | `figures/demo/` | local stack up and seeded, `SEED_USER_PASSWORD` |

Run each as `bun run <command>` from `docs/reports`.

Notes:

- `--ref` pins the commit; `--label` is the name the report cites. Capture from the release commit and cite it as `main`, so every figure shows one commit.
- `gen-ac-table` flags record runs it cannot see: pass `--e2e-passed` after the Playwright specs pass on the stack, `--web-passed` after Vitest passes, and one `--web-fail` per failing Vitest title.
- `capture-demo` types `SEED_USER_PASSWORD` into the masked login field only. Start the stack with the seed first (repo root README).
- Each capture writes a `.json` sidecar. `check-report` reads it to catch blank bands, repeated bands, and Mermaid that GitHub did not draw – rerun that document if it reports one.

### Release day, in order

```sh
bun run render-docs 01 --ref <sha> --label main
bun run slice-tall 01
bun run capture-diagrams 01 --ref <sha> --label main
bun run gen-part-lists 01
bun run render-reviews 01
bun run capture-board 01
bun run collect-team 01
bun run gen-team-table 01
bun run go-test 01
bun run gen-ac-table 01 --commit <sha> --e2e-passed --web-passed
bun run capture-demo 01
bun run pdf 01
bun run check-report 01
```

### Open work

Mark it in LaTeX as `\todo{<who or what>: ...}`, for example `\todo{release: ...}`, `\todo{#30: ...}` or `\todo{capture: ...}`. `check-report` groups open items by that prefix, so the list reads as what each one waits for. A missing figure draws a red "pending" box instead of breaking the build; `check-report` counts those too.

## Start a new sprint

1. Copy `sprints/01/` to `sprints/NN/` (two digits, so folders sort) and empty `content/` and `figures/`.
2. In `main.tex`, set `\sprintname` and `\deliverable`.
3. Rewrite `report.config.ts` for the sprint's documents, reviews, contracts, and tickets, and `demo.ts` for its demos.
4. Run the release-day steps with `NN`.

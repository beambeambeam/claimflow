# Lab 5 evidence index

The LaTeX source of the Lab 5 submission (CPE 334 Lab 5: GCP Deployment and CI/CD), as submitted on 19 November 2026. It is the evidence index, with the headings "Answer Part 0" to "Answer Part 4". It cites commit `a76484c`, the head of `main` when it was submitted, and Cloud Run revision `claimflow-00005-2sb`.

```
labs/05/
├── main.tex           the whole report, with its own preamble
├── report.config.ts   empty config, so the shared pdf and identity scripts accept the folder
└── figures/           the redacted screenshots it uses (GCP Console, Cloud Shell, GitHub, the live app)
```

## Build

From `docs/reports`, with `.env` filled in as for the sprint reports (see `../../README.md`, Setup):

```sh
bun run pdf ../labs/05    # identity + XeLaTeX twice -> labs/05/main.pdf
```

Student names and IDs reach the PDF only through `build/identity.tex`, as in the sprint reports. The PDF and `build/` are ignored by Git.

The baht sign uses Leelawadee UI, which ships with Windows. Where that font is missing, the report prints `THB` instead.

## Figures

The screenshots were cropped from the Lab 5 captures. Before they were committed, each one was checked: the deployer's email is blacked out, the login password is masked, and no figure shows the GCP project number, a secret value, or a connection string.

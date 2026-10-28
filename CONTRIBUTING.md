# Contributing

This project uses GitHub issues and pull requests to plan, review, and ship changes. Keep issue and pull request content in English so the project history remains searchable and consistent.

## Workflow

1. Start from the latest `main` branch.
2. Choose or create a focused issue before making a change.
3. Create a short-lived branch using the pattern `<type>/<short-kebab-name>`.
4. Keep the change focused and link the issue from the pull request.
5. Run the relevant validation commands locally.
6. Open a pull request and complete its description before requesting review.

See [Branch, commit, and release naming](#branch-commit-and-release-naming) for the allowed branch types and release impact.

## Commit messages

Use a short Conventional Commit subject with no scope, body, or footer:

```text
type: description
```

Examples:

```text
feat: add course search
fix: preserve the selected term after refresh
chore: upgrade stable dependencies
```

## Branch, commit, and release naming

This section is the release source of truth for people and agents.

### Branch names

Use one lowercase type and a short kebab-case name:

```text
<type>/<short-kebab-name>
```

Use these branch types:

```text
feat  fix  refactor  chore  docs  test  ci  build  perf
```

Examples:

```text
feat/course-search
fix/semester-picker-state
ci/semantic-release
```

Branch names organize work; commit subjects control releases.

### Release signals

semantic-release reads the final commit subjects on `main`:

| Commit subject | Version impact |
| --- | --- |
| `feat: description` | Minor release (`vG.X.Y.0`) |
| `fix: description` or `perf: description` | Patch release (`vG.X.Y.Z`) |
| `feat!: description` or `fix!: description` | Major release (`vG.X.0.0`) |
| `chore:`, `ci:`, `docs:`, `refactor:`, `test:`, `build:`, or `style:` | No release |

Use `type!: description` for breaking changes. When squash-merging, set the pull request title to the final commit subject because it becomes the release signal.

### Version and tag ownership

- Release tags use `vG.X.Y.Z`, for example `v0.2.0.1`. `G` is the generation; `X.Y.Z` is the SemVer version that semantic-release calculates.
- The generation is fixed in `release.config.js`. Commits never change it. Change it by hand only when the project starts a new generation; the next release then continues the `X.Y.Z` count under the new generation.
- Git tags are the only record of the version. `package.json` files have no `version`, and the release flow makes no release commit and does not publish to npm.
- semantic-release runs after the `main` workflow passes, creates the tag, and publishes the GitHub Release.

Expected result: the release impact is clear from the commit subject before merge.

## Application boundaries

- Add file-based routes in `apps/web/src/routes/` and reusable UI in `apps/web/src/components/`.
- Keep router setup in `apps/web/src/router.tsx` and app startup in `apps/web/src/main.tsx`.
- Treat `apps/web/src/routeTree.gen.ts` as generated output; do not edit it by hand.
- Keep global styles in `apps/web/src/styles.css` and use Tailwind for component styling.

## Definition of ready

- The issue states the context, scope, non-goals, and acceptance criteria.
- Authorization, business rules, and validation needs are identified.
- Relevant loading, empty, error, and success states are defined.
- Translation, link, redirect, security, and operational impact are considered.

## Definition of done

- The implementation satisfies the issue acceptance criteria.
- Tests cover the important behavior and failure paths.
- Relevant documentation and generated route or type artifacts are updated.
- The pull request explains the change, validation, risk, and follow-up work.
- The required CI workflow passes.

## Local validation

Run the checks that match the change from the repository root. For a full change, use:

```sh
bun run lint:fix
bun run format
bun run check
bun run test:coverage
bun run build
```

For Go changes, also run `bun run vuln` in `apps/api`, and `bun run tidy` after dependency changes.

Do not commit secrets, production environment files, database dumps, or generated output that is ignored by the repository.

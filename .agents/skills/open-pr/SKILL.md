---
name: open-pr
description: >
  Opens a well-formed pull request for this repository. Use when asked to
  create, open, or submit a pull request, or to push a branch and request a
  review. Enforces correct branch naming, conventional commit title, signed-off
  commits, the project PR template, and a green CI suite before submitting.
compatibility: Requires git and GitHub CLI (gh)
allowed-tools: Bash(git:*) Bash(gh:*) Bash(pnpm:*) Bash(mise:*)
---

# Open Pull Request Skill

## Pre-flight checks

Before pushing anything, read these files to understand conventions:

- `CONTRIBUTING.md` — setup, tooling, test layout, commit format
- `.commitlintrc.js` — enforced commit types and scopes

Run the full check suite and confirm it is green:

```sh
make provider
golangci-lint run
go test ./provider/... ./tests/...
```

Never open a PR with a failing check suite. Fix the issue first.

## Branch naming

Branch from `main`. Match the branch name to the conventional commit type:

| Type          | Branch pattern                 |
| ------------- | ------------------------------ |
| Bug fix       | `fix/<short-description>`      |
| New feature   | `feat/<short-description>`     |
| Documentation | `docs/<short-description>`     |
| Tooling/build | `build/<short-description>`    |
| Refactor      | `refactor/<short-description>` |

## Commits

Every commit must follow the format enforced by `.commitlintrc.js`:

```text
<type>(<scope>): <Subject in sentence case>
```

**Allowed scopes:** `provider` · `sdk` · `examples` · `tests` · `docs` · `ci` · `deps` · `tooling`

Every commit must include a DCO sign-off (cryptographic signing is configured
globally and applies automatically — see `.agents/skills/commit/SKILL.md`):

```sh
git commit -s -m "feat(provider): Add support for custom claim conditions"
```

## Opening the PR

Push the branch, then create the PR using the project template:

```sh
git push -u origin <branch-name>
gh pr create \
  --title "<type>(<scope>): <Subject in sentence case>" \
  --body-file .github/PULL_REQUEST_TEMPLATE.md \
  --base main
```

## Filling the PR template

The template (`.github/PULL_REQUEST_TEMPLATE.md`) has five sections plus a
checklist. Fill each one as follows.

### Summary

One sentence, present tense, mirroring the primary commit subject:

> Add support for custom claim conditions.

### Why

Explain the motivation — what gap, pain point, or bug triggered this change.
Link the related issue: `Closes #<number>` (auto-closes on merge) or "N/A".

### What changed

3–7 bullet points covering key implementation decisions. Reviewers should
understand the approach without reading every diff line.

### How to validate

Provide copy-pasteable commands:

```sh
make provider
make test
make lint
```

If the change needs a live Pocket-ID instance, add:
`docker compose -f docker-compose.test.yml up -d --wait` first.

### Impact

- **No breaking changes** — for additions and internal fixes.
- **Breaking change** — for any change to the provider's public schema.
  Describe migration steps.

### Checklist — items AI agents often miss

| Item                       | How to satisfy it                                          |
| --------------------------- | ------------------------------------------------------------ |
| Tests added                 | Extend or add a `*_test.go` alongside the changed code    |
| Schema reflects the change  | Regenerate via `make generate_schema` if resources changed |
| README updated              | If the change affects public provider behaviour           |
| Commits signed off          | `git commit -s` on every commit                            |

## Responding to review feedback

For small fixes, amend the commit and force-push:

```sh
git add <files>
git commit --amend --signoff --no-edit
git push --force-with-lease
```

For larger review rounds, prefer a new commit (easier to diff):

```sh
git commit -s -m "fix(provider): Address review: validate claim name length"
```

## CI

The CI runs commit-message linting, `golangci-lint`, `make provider`, and the
unit/acceptance test suite. If any check is red, fix it in a new commit — do
not skip hooks or force-merge.

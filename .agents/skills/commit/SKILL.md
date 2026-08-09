---
name: commit
description: >
  Creates properly formatted Git commits for pulumi-pocket-id. Use whenever the
  user wants to commit changes, stage files, write a commit message, or
  finalize work — "commit this", "commit everything", "make a commit", "stage
  and commit". Also triggers when a coding task is done and the next logical
  step is to persist the change. Enforces Conventional Commits with a
  mandatory single scope, a WHY-focused body, the DCO Signed-off-by trailer,
  and an Assisted-by trailer for AI attribution — no Co-authored-by trailer.
compatibility: Requires git
allowed-tools: Bash(git:*)
---

# pulumi-pocket-id Commit Skill

## Why we commit this way

Commits are a historical record, not just a sync mechanism.

- **Small, atomic commits aid review and bisecting.** One logical change per
  commit — a commit that touches `provider/` and `sdk/` for two unrelated
  reasons is hard to review and hard to revert.
- **The body carries the WHY.** The diff already shows what changed; a commit
  without a reason is only half the story for whoever reads `git log` later.

## Commit format

```text
type(scope): Subject starting with uppercase

Body — one or more paragraphs explaining WHY, wrapped at 80 chars/line.

Assisted-by: <provider>:<model-id>
Signed-off-by: Name <email>
```

For breaking changes:

```text
type(scope)!: Subject starting with uppercase

Body explaining what breaks and why the change is worth it.

BREAKING CHANGE: What breaks and how to migrate.

Assisted-by: <provider>:<model-id>
Signed-off-by: Name <email>
```

This mirrors `.commitlintrc.js` exactly — read that file if this skill and
the config ever appear to disagree; the config is the enforced source of
truth and this skill must be updated to match (see "Keeping this skill in
sync" below).

## Types

| Type       | When to use                                       |
| ---------- | -------------------------------------------------- |
| `feat`     | Introduce new features                             |
| `fix`      | Fix a bug                                           |
| `docs`     | Documentation-only changes                          |
| `style`    | Code style changes (formatting, no logic change)    |
| `refactor` | Code restructuring (no feature, no bug fix)         |
| `perf`     | Performance improvements                            |
| `test`     | Add or update tests                                 |
| `build`    | Build system or external dependency changes         |
| `ci`       | CI/CD configuration changes                         |
| `chore`    | Maintenance not touching src or tests               |
| `revert`   | Revert a previous commit                            |

Only `feat`, `fix`, `refactor`, and `build` may be marked breaking (`!`).

## Scope — required, exactly one

`enableMultipleScopes` is `false` — a commit takes exactly one scope. If a
change legitimately spans two scopes, split it into two commits instead of
picking a multi-scope workaround.

| Scope      | Path / description                              |
| ---------- | ------------------------------------------------- |
| `provider` | Provider logic (`provider/`)                     |
| `sdk`      | SDK libraries (`sdk/`)                           |
| `examples` | Example programs (`examples/`)                   |
| `tests`    | Test suites (`tests/`)                           |
| `docs`     | Documentation (`README.md`, `CODE-OF-CONDUCT.md`, `docs/`) |
| `ci`       | CI/CD configuration (`.github/`, `.ci-mgmt.yaml`) |
| `deps`     | Dependency updates                               |
| `tooling`  | Dev tooling (`.config/mise.toml`, `.goreleaser*.yml`, `scripts/`, `Makefile`, `.commitlintrc.js`) |

### Scope decision tree

```text
Which files changed?
├── provider/*                         → provider
├── sdk/*                              → sdk
├── examples/*                         → examples
├── tests/*                            → tests
├── README.md, CODE-OF-CONDUCT.md,
│   docs/*                             → docs
├── .github/*, .ci-mgmt.yaml           → ci
├── go.mod, go.sum, package.json
│   version bumps                      → deps
├── .config/mise.toml, .goreleaser*.yml,
│   scripts/*, Makefile,
│   .commitlintrc.js                   → tooling
└── ambiguous or spans two scopes      → ask the user, or split into two commits
```

## Subject

- Imperative mood, sentence-case (uppercase first letter), no trailing period
- Max 100 characters (full header line, including `type(scope): `)
- Must not contain a literal `#` — the parser reserves it for a trailing
  issue reference (see "References" below) and a mid-subject `#` breaks
  parsing

## The body — WHY, not WHAT

Required for anything non-trivial. Optional only for genuinely trivial
changes (a version bump, a typo fix, a single obvious line).

- Max 80 characters per line, sentence-case
- Must explain the motivation, the alternatives considered, or the trade-offs
  — never restate what the diff already shows

For good/bad examples and a full commit sample, read
`references/body-guide.md`.

### The WHY must come from the user, not from the diff

The diff shows what changed. Only the human knows why they decided to make
that change. An inferred motivation can sound plausible and still mislead
future readers of the history.

Acceptable sources for the WHY: the user's own words anywhere in the
conversation (even stated several exchanges earlier), or a linked issue/PR
they pointed to. Not acceptable: your interpretation of the diff, or a
motivation you reconstructed from the code itself.

Before writing the body, check:

1. **Is there a WHY at all?** A conclusion ("fix this", "vire ça") is not a
   reason. If the user hasn't said why, ask before writing anything.
2. **Is the WHY specific enough to matter in six months?** A vague reason
   ("for consistency", "to improve the code") tells a future reader nothing.
   Push for the specifics with one targeted question — don't pad with
   generalities.

If context is genuinely missing and cannot be inferred, ask one focused
question: "What problem does this change solve?" or "Why this approach over
X?". As a fallback, `git diff --staged` and `git log --oneline -10` provide
surrounding context — flag any inference with "Based on the diff, this
appears to …".

## References

Reference issues/PRs in the body or footer as plain text (`Closes 42`,
`Refs pull request 17`) rather than a bare `#42` mid-sentence — the header's
own `#ref` capture is reserved for a trailing `(#N)` on the subject line, and
consistency avoids ambiguity elsewhere in the message.

## Footer — Signed-off-by and Assisted-by, no Co-authored-by

### DCO sign-off — required

This project requires a DCO sign-off on every commit (see the "Legal"
section of `.github/PULL_REQUEST_TEMPLATE.md`). Add it with `-s`/`--signoff`,
never by hand-typing the trailer:

```sh
git commit -s -m "<message>"
```

Cryptographic signing (`-S`) is configured globally on this machine
(`commit.gpgsign=true`, SSH format) and applies automatically — no need to
pass `-S` explicitly. For setup details or verifying a signature elsewhere,
read `references/signing.md`.

### Assisted-by — required whenever an AI assisted the commit

Every commit written with AI assistance must carry an `Assisted-by:` trailer
disclosing it, instead of `Co-authored-by:`. `Assisted-by:` is the emerging
open-source convention (Linux Kernel, Fedora, LLVM, OpenTelemetry, OpenInfra,
Rocky Linux) and is semantically more accurate: the human stays the sole
author and legal signer of the DCO; the AI is acknowledged as an assistant,
not a co-author with legal personhood.

Format: `Assisted-by: <provider>:<model-id>`, using the identifier of the
model powering the current session:

| Tool                     | Trailer                                       |
| ------------------------ | ------------------------------------------------ |
| Claude Code (Anthropic)  | `Assisted-by: claude-code:claude-sonnet-5`       |
| GitHub Copilot           | `Assisted-by: github-copilot:<model-id>`         |
| Codex / ChatGPT (OpenAI) | `Assisted-by: codex:<model-id>`                  |

Version numbers use a dot separator (`4.6`, not `4-6`) even if the system's
internal model ID uses hyphens — the trailer always uses the public model
name with dots.

Add the trailer by hand in the commit message body passed to `-m`; `git
commit` has no flag for it. Never substitute `Co-authored-by:` for it — see
the "Bad" example below.

For a breaking change, add a `BREAKING CHANGE:` paragraph in the body before
the footer trailers (blank line before the footer still applies after it).

## Commitlint rules (canonical reference)

This reproduces `.commitlintrc.js` so the skill is self-contained. When that
file changes, update this table in the same commit (see "Keeping this skill
in sync").

| Rule                    | Level | Value                                   |
| ------------------------ | ----- | ---------------------------------------- |
| `header-max-length`      | error | 100                                       |
| `header-full-stop`       | error | never `.`                                 |
| `header-trim`            | error | always                                    |
| `header-case`            | off   | —                                         |
| `type-enum`              | error | see Types table                          |
| `type-case`              | error | lower-case                                |
| `type-empty`             | error | never                                     |
| `scope-enum`             | error | see Scope table                          |
| `scope-case`             | error | lower-case                                |
| `scope-empty`            | error | never — scope is mandatory                |
| `subject-case`           | error | sentence-case                             |
| `subject-empty`          | error | never                                     |
| `subject-full-stop`      | error | never `.`                                 |
| `subject-max-length`     | error | 100                                       |
| `body-leading-blank`     | error | always — blank line before body           |
| `body-max-line-length`   | error | 80                                        |
| `body-case`              | error | sentence-case                             |
| `footer-leading-blank`   | error | always — blank line before footer         |
| `footer-max-line-length` | error | 80                                        |

Parser pattern (from `.commitlintrc.js`): `type(scope)!: Subject (#ref)`,
with `!` only valid for breaking-capable types.

## Keeping this skill in sync

When `.commitlintrc.js` changes (new types, scopes, or rules), update this
skill in the same commit:

1. New type → add to the Types table and the `type-enum` reference row.
2. New scope → add to the Scope table, the decision tree, and `scope-enum`.
3. Rule change → update the Commitlint rules table.
4. Verify — re-read both files and confirm every rule, type, and scope
   matches exactly.

### When commitlint rejects a commit

1. Read the reported rule (e.g. `scope-empty`, `subject-case`).
2. Look it up in the Commitlint rules table above.
3. Fix the message and retry.
4. If the rule itself seems wrong, don't work around it — tell the user
   which rule failed, propose updating `.commitlintrc.js`, and if they
   agree, update the config and this skill together.

| Error                  | Cause                              | Fix                                      |
| ----------------------- | ----------------------------------- | ------------------------------------------ |
| `scope-empty`           | Missing `(scope)`                   | Add a scope from the allowed list          |
| `scope-enum`            | Scope not in the allowed list       | Pick a valid scope or split the commit     |
| `type-enum`             | Unknown type                        | Use only types from the allowed list       |
| `subject-case`          | Subject not sentence-case           | Uppercase the first letter                 |
| `subject-full-stop`     | Subject ends with `.`               | Remove the trailing period                 |
| `header-max-length`     | Header exceeds 100 chars            | Shorten the subject                        |
| `body-leading-blank`    | No blank line before body           | Add an empty line after the subject        |
| `body-max-line-length`  | Body line exceeds 80 chars          | Rewrap the body                            |
| `footer-leading-blank`  | No blank line before footer         | Add an empty line before `Signed-off-by:`  |

## Workflow

### 1. Survey the workspace

```sh
git status
git diff --cached --name-only
git diff --name-only
git --no-pager log --oneline --no-merges -10
```

Staged files (`--cached`) determine what goes into the commit. Mention any
relevant unstaged changes to the user rather than silently including or
dropping them.

### 2. Check for commit splitting

Since scope is single-valued, changes spanning two scopes (e.g. `provider/`
and `sdk/`) need two commits unless one is a trivial side effect of the
other. Ask the user if it's unclear whether the change is truly atomic.

### 3. Select the type and scope

Use the Types table and the scope decision tree. Never guess on an ambiguous
case — ask.

### 4. Draft the subject

Imperative mood, sentence-case, no period, max 100 chars total.

### 5. Write the body

Follow "The body — WHY, not WHAT" above. Skip only for genuinely trivial
changes.

### 6. Stage and commit

```sh
git add <files>
git commit -s -m "$(cat <<'EOF'
type(scope): Subject

Body explaining the motivation and trade-offs.

Assisted-by: claude-code:claude-sonnet-5
EOF
)"
```

`-s` adds the DCO `Signed-off-by` trailer automatically. The `Assisted-by`
trailer must be typed into the message body — there is no flag for it. Never
add a `Co-authored-by` trailer.

## Examples

### Good — simple fix, clear subject and WHY

```sh
git commit -s -m "$(cat <<'EOF'
fix(provider): Reject empty resource names before create

The upstream API accepts empty names and creates an unnamed resource
that cannot be imported or destroyed by name afterwards. Validating
client-side avoids leaving orphaned resources that require manual
cleanup via the API.

Assisted-by: claude-code:claude-sonnet-5
EOF
)"
```

### Good — breaking change

```sh
git commit -s -m "$(cat <<'EOF'
feat(sdk)!: Require explicit base URL in provider config

Defaulting to a placeholder host silently created resources against
the wrong Pocket-ID instance for users who forgot to set it. Making
the field required surfaces the mistake at plan time instead.

BREAKING CHANGE: `baseUrl` must now be set explicitly in provider
config; the implicit default is removed.

Assisted-by: claude-code:claude-sonnet-5
EOF
)"
```

### Bad — no scope, body restates the diff

```sh
git commit -m "fix stuff in provider - added a check for empty names and updated the tests"
```

### Bad — Co-authored-by trailer (not used in this project — use Assisted-by)

```sh
git commit -s -m "$(cat <<'EOF'
fix(provider): Reject empty resource names before create

Added a check for empty names.

Co-authored-by: Claude <claude@anthropic.com>
EOF
)"
```

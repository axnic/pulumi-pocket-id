# Contributing to pulumi-pocket-id

Thanks for your interest in contributing. This document covers setting up a dev
environment, the test layout this project follows, generating the SDKs, commit
conventions, and how a release gets cut.

## Code of Conduct

Please read our [Code of Conduct](CODE-OF-CONDUCT.md) before participating.

## Setting up your development environment

### Devcontainer (recommended)

Open the repo in VS Code with the
[Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
(or a GitHub Codespace) and "Reopen in Container" - everything is ready the moment it
finishes building, no manual setup step:

- Go, the Pulumi CLI, and everything else pinned in `.config/mise.toml` are installed
  via the [mise devcontainer feature](https://github.com/devcontainers-extra/features/tree/main/src/mise)
  (`ghcr.io/devcontainers-extra/features/mise:1`) and `postCreateCommand`.
- An always-on Pocket-ID instance (`.devcontainer/docker-compose.yml`) is already
  running, sharing this container's network namespace, so `http://localhost:1411`
  just works. Unlike some providers, there's no bootstrap step to wait on -
  Pocket-ID's `STATIC_API_KEY` env var grants instant admin access.
- `POCKET_ID_BASE_URL` / `POCKET_ID_API_KEY` are already exported (see the `dev`
  service's `environment:` block), so acceptance tests run immediately.

`make lint` and `make test` work immediately - and because `POCKET_ID_BASE_URL` is
already set, `make test` runs the acceptance tests too (normally skipped without a
live instance), so it's a fuller check inside the devcontainer than outside it.
`make dev-up` / `make test_e2e` also work, via a *separate*, disposable Pocket-ID
stack (see below) reached through the `docker-outside-of-docker` feature - no port
conflict with the always-on instance, since that one publishes no host ports of its
own.

### Without a devcontainer

Any environment with Go, the Pulumi CLI, and Docker works (see `.config/mise.toml` for
exact pinned versions - `mise install` picks all of them up automatically):

```sh
make dev-up    # starts Pocket-ID, prints POCKET_ID_BASE_URL / POCKET_ID_API_KEY
               # to export
pulumi up      # or: cd examples/yaml && pulumi up
make dev-down  # tear it down when you're done
```

`make dev-up` is `make test_e2e`'s setup half, minus the automated test run and
teardown - the instance stays up until you `make dev-down` it. Pin a version the same
way: `POCKET_ID_IMAGE=pocketid/pocket-id:v2.9.0 make dev-up`.

## Tests

Tests live in `tests/`, layered from fastest/most-isolated to slowest/most-realistic:

1. **Fake-backed integration tests** (`tests/{oidc_client,user,usergroup,custom_claims,config}_test.go`) -
   run the real provider through `pulumi-go-provider`'s `integration` test harness
   (`fake_test.go`'s `testServer` helper) against `fakePocketID`, an in-memory
   `httptest.Server` standing in for the Pocket-ID REST API. Fast, hermetic, no Docker
   required.
2. **Acceptance tests** (`tests/acceptance_test.go`) - the same provider harness, but
   against a real Pocket-ID instance. Skipped automatically (`t.Skip`) unless both
   `POCKET_ID_BASE_URL` and `POCKET_ID_API_KEY` are set, so they run inside the
   devcontainer but don't slow down a plain `make test` elsewhere.

Commands:

- `make test` - fast, hermetic unit tests (layer 1, plus layer 2 wherever a live
  instance happens to be reachable).
- `make test_e2e` - starts a real, disposable Pocket-ID instance via Docker Compose
  (`docker-compose.test.yml`), runs the acceptance suite against it, then tears it
  down. Requires Docker. Defaults to `pocketid/pocket-id:v2`; pin another version with
  `POCKET_ID_IMAGE`, e.g. `POCKET_ID_IMAGE=pocketid/pocket-id:v2.9.0 make test_e2e`.
  `STATIC_API_KEY` support requires Pocket-ID ≥ v2.3.0.
- `make lint` - runs `golangci-lint`.

CI (`.github/workflows/ci.yml`) runs commit-message linting, lint, build, and both
test layers on every pull request and push.

## Building and generating the SDKs

Build and install the provider binary:

```sh
make build install
```

After changing resource code (fields, descriptions, new resources), regenerate the
schema (`provider/cmd/pulumi-resource-pocket-id/schema.json`) and all 4 SDKs:

```sh
make codegen
```

This regenerates `sdk/{go,nodejs,python,dotnet}/` from the schema. Only the
generated *source* under `sdk/*/` is committed; per-language build/publish output
(`sdk/python/venv/`, `sdk/nodejs/bin/`, `sdk/dotnet/bin/`, ...) is gitignored (see the
root `.gitignore` - the per-language `.gitignore`/`.gitattributes` files that codegen
itself generates get wiped on every run, so they can't be hand-edited to cover this).

To build a single language's SDK locally (e.g. to smoke-test a packaging change),
`make {nodejs,python,dotnet,go}_sdk` runs codegen plus that language's own build
step (`yarn install && tsc`, a Python venv + `build`, `dotnet build`). There's no
Java/Maven SDK - dropped as not worth the setup cost given how little of the Pulumi
ecosystem uses Java.

Field-level documentation comes from `infer.Annotate` calls in each resource's
`*.go` file (see `provider/oidc_client.go` for the pattern) - these flow straight
into `schema.json` and from there into every generated SDK's doc comments, so
document there, not by hand-editing generated SDK output.

## Commit conventions

Commits follow [Conventional Commits](https://www.conventionalcommits.org/), enforced
by commitlint (`.commitlintrc.js`) in CI:

```
type(scope): Subject starting with uppercase

Body — one or more paragraphs explaining WHY, wrapped at 80 chars/line.

Signed-off-by: Name <email>
```

- **Type** - one of `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`,
  `build`, `ci`, `chore`, `revert`.
- **Scope** - exactly one of `provider`, `sdk`, `examples`, `tests`, `docs`, `ci`,
  `deps`, `tooling` (see `.commitlintrc.js` for the full list with descriptions).
  Required, and only one per commit - a commit touching two unrelated areas for two
  unrelated reasons should be two commits.
- **Subject** - sentence case, no trailing period, ≤100 chars.
- **Body** - required, explains *why*, not just what (the diff already shows what).

Run `npx commitlint --edit` (or let the `commit-msg` hook do it) to validate a message
before committing. See `.agents/skills/commit/SKILL.md` for the full convention this
project's AI assistant follows, including trailers.

## Repo layout

- `provider/` - the provider implementation (`provider.go`, `config.go`,
  `oidc_client.go`, `user.go`, `user_group.go`, `custom_claims.go`, `client.go`),
  plus the codegen entrypoint (`provider/cmd/pulumi-resource-pocket-id/`).
- `sdk/` - the generated SDKs for all 4 languages (`make codegen`).
- `examples/` - the YAML and Go example programs, and their lifecycle tests.
- `tests/` - fake-backed integration tests and the acceptance test suite.
- `docker-compose.test.yml` - the local dev / CI fixture: a disposable Pocket-ID
  instance with a fixed, test-only `STATIC_API_KEY` (not sensitive - the instance is
  ephemeral and local-only).
- `.devcontainer/` - VS Code / Codespaces dev environment (its own
  `docker-compose.yml`, an always-on Pocket-ID instance), see
  [Setting up your development environment](#setting-up-your-development-environment).
- `Makefile` - build, codegen, lint, test, and dev-instance targets.

## Releasing

Publishing the provider binary and the 4 SDKs to their registries is a maintainer
task - see [RELEASING.md](RELEASING.md) for the required secrets and how to cut a
release. You don't need any of this to build, test, or contribute.

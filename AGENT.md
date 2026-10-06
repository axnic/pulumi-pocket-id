# Tech Documentation Analysis

## Project
- Stack: Go (pulumi-go-provider v1.1.2, see go.mod for the Go version), Pulumi CLI (version in `.pulumi.version`), Docker (E2E only)
- Doc Language: en
- Module: github.com/axnic/pulumi-pocket-id
- Provider namespace: axnic; provider name: `pocket-id`
- Resource tokens (`pocket-id:index:<Name>`): User, UserGroup, UserGroupMembers, UserCustomClaims,
  UserGroupCustomClaims, OidcClient, OidcClientSecret, Api, ApiClientGrant,
  ScimServiceProvider, SignupToken, ApplicationConfiguration, ApplicationImage
- Function tokens: getUser, getUserGroup, getOidcClient, getOpenIdConfiguration

## Doc Locations
- Root entry point: README.md (single comprehensive doc: Prerequisites, Quickstart, Resource
  reference, Provider config, Dev & testing, Compatibility, Known limitations)
- CONTRIBUTING.md (dev environment, test layers, SDK generation, commit conventions), RELEASING.md
- No docs/adr, docs/api or docs/guides directories exist in this repo

## Style by Type
| Type | Format | Tone | Example File |
|------|--------|------|--------------|
| README | H2 sections as listed above | Terse, no marketing fluff, no emojis | README.md |
| Go doc comments (Annotate) | Full sentences, authoritative field descriptions | Precise/technical | provider/*_resource.go, provider/*_function.go, provider/config.go |
| Makefile/CI comments | Short imperative comments above targets | Terse | Makefile, .config/mise.toml |

## Architecture
- Hand-written provider with `infer` (never generated from the OpenAPI spec). `provider.go` builds the
  provider from `registeredResources` / `registeredFunctions` (`helpers.go`); each
  `provider/<name>_resource.go` / `<name>_function.go` registers itself from an `init()`.
- Typed REST client: `provider/internal/pocketidclient` (`client.go`: `New`, `Do`, `APIError`,
  `IsNotFound`, `listAll`, `DoMultipart` for file uploads, `GetBytes` for raw downloads), one `<domain>.go` + `<domain>_test.go` per API domain.
- Authentication: `X-API-Key` header (an API key, or the instance's `STATIC_API_KEY`, Pocket-ID >= v2.3.0).
- Conventions: Read returns an empty ID on 404 (drift), Delete tolerates 404, no API call on preview,
  generated secrets carry `provider:"secret"` and are never re-read (kept from state, empty after
  import), immutable fields use `replaceOnChanges`, association resources are authoritative
  (Delete sets the empty set). `ApplicationConfiguration` is a singleton (ID `app-config`,
  Delete is a no-op); its update reads `/all`, overlays the managed fields and PUTs the full set.

- Images: the `image` input of `ApplicationImage` is a native Pulumi asset (`types.AssetOrArchive`, assets
  only). The state carries a `sha256` of the content; Diff compares the input digest to it (the engine may
  pass an asset reduced to its hash, which for assets is the content SHA-256), Read stores the digest of the
  bytes served by the API so drift shows up as a diff, and Update re-uploads (idempotent). Imports leave
  `image` empty (not readable as an asset). `favicon` and `email` cannot be deleted through the API:
  Delete is a no-op for them. In the in-process test harness the returned asset is a plain object, so tests
  re-set it with `asState`.
- OIDC client logos (`OidcClient.logo` / `darkLogo`, optional assets, mutually exclusive with `logoUrl` /
  `darkLogoUrl`, checked in Check/Create/Update): there is no custom Diff, the default diff compares the
  state asset to the input asset by hash. The upload happens after the client create/update (POST
  `/logo?light=`), is skipped when `logoSha256` / `darkLogoSha256` already match, and a logo dropped from the
  program is deleted. Read re-downloads a managed logo (asset or digest in state): changed bytes replace the
  state asset by one reduced to the served digest (so the next diff re-uploads), a 404 drops it. After an
  import the assets are empty.

## Notes for future updates
- Ground truth for resource field descriptions is the `Annotate()` methods. The README's
  "Resource reference" tables are generated from the provider schema (build the provider and run
  `pulumi package get-schema <binary>`), so regenerate them after changing descriptions rather than
  editing by hand. Provider config descriptions live in `provider/config.go`.
- Known scope limits are deliberate and listed in README "Known limitations": no instance
  bootstrapping, no user profile pictures, device login, audit logs, WebAuthn credentials, one-time access
  and other action-type endpoints. Keep that section explicit.
- Only the YAML and Go example programs are exercised; nodejs/python/dotnet SDKs are generated and
  published. Do not reintroduce Java/Maven.
- Never edit `sdk/` or `provider/cmd/pulumi-resource-pocket-id/schema.json` by hand (`make codegen`).
- Tests: `provider/internal/pocketidclient/*_test.go` (httptest), `tests/*_test.go` (provider through
  the integration harness against `fakePocketID`; each domain adds its endpoints from
  `tests/fake_<domain>_test.go` via `fakeRegistrars`), `tests/acceptance_test.go` (build tag `e2e`,
  every resource against a real Pocket-ID, driven with property maps), `examples/*_test.go`
  (example lifecycle, skipped without `POCKET_ID_BASE_URL`).
- Commands run through mise: `mise run ci` (build, tests, coverage), `ci:build`, `ci:test`,
  `ci:coverage` (floor `COVERAGE_FLOOR`, a ratchet: raise it, never lower it to unblock a PR),
  `ci:e2e` (`E2E_VERSION=v2.18.0`, docker compose with `POCKET_ID_IMAGE=pocketid/pocket-id:$E2E_VERSION`,
  `go test -tags e2e ./tests/...`), `ci:e2e:versions` (the 5 latest stable Pocket-ID minors (>= v2.3),
  one `vX.Y.0` each, from the GitHub releases of pocket-id/pocket-id; set `GITHUB_TOKEN` to avoid API rate limits).
  `scripts/get-versions.sh` only resolves the Go/Pulumi versions for mise (vfox-pulumi), not E2E versions.
- Compatibility matrix: one thin caller per Pocket-ID version,
  `.github/workflows/merge_group,pull_request,push.e2e-vX.Y.0.yaml`, generated from
  `.github/templates/e2e-caller.yaml.tmpl` (placeholder `__VERSION__`) by the E2E Sync workflow of
  `axnic/.github` (caller `schedule,workflow_dispatch.e2e-sync.yaml`), together with a row of the
  README's Compatibility table (keep the row format
  `| vX.Y.0 | [![E2E (Pocket-ID vX.Y.0)](...)](...) |`). One file per version because status badges are
  per workflow file.
- E2E fixture: `docker-compose.test.yml` (image from `POCKET_ID_IMAGE`, `STATIC_API_KEY` from
  `POCKET_ID_API_KEY`, port 1411). The E2E suite skips the APIs, SCIM and signup-token subtests when the
  instance does not serve those endpoints (404/405 on create).
- Renovate (`renovate.json`, extending `local>axnic/.github:pulumi`): weekly dependency updates for
  github-actions (pinned by digest), gomod and mise. Minor and patch updates from 1.0 on are grouped;
  each 0.x minor and each major gets its own PR. PR titles are `build(deps): ...` in sentence case
  (scope `deps`, the only dependency scope in `.commitlintrc.js`). The generated `sdk/` is never
  updated by Renovate. What happens to a PR afterwards (approval, auto-merge) is handled by the
  central Dependency Updates workflow in `axnic/.github`, not by a file in this repo. Dependabot
  alerts and security updates stay enabled (Security tab); there is no `dependabot.yml`.
- Commits follow Conventional Commits with a mandatory scope (see `.commitlintrc.js` and CONTRIBUTING.md).

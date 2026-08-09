# Releasing

This document is for maintainers cutting a release of the provider and its SDKs.

## How a release works

Pushing a tag matching `v*` (e.g. `v1.0.0`, or `v1.0.0-alpha.1` for a prerelease)
triggers [`.github/workflows/release.yml`](.github/workflows/release.yml), which:

1. Builds the provider binary for darwin/linux/windows (amd64+arm64) with
   [GoReleaser](.goreleaser.yml) and publishes them as a GitHub Release with
   checksums. **This step alone is enough** for
   `pulumi plugin install resource pocket-id <version>` to work — the provider
   resolves straight from this repo's GitHub Releases
   (`WithPluginDownloadURL` in `provider/provider.go`), no Pulumi Registry
   listing required.
2. Pushes a second, path-prefixed tag
   (`sdk/go/pulumi-pocket-id/vX.Y.Z`) so
   `go get github.com/axnic/pulumi-pocket-id/sdk/go/pulumi-pocket-id@vX.Y.Z`
   resolves to a clean version instead of a pseudo-version (Go's module
   versioning rule for modules living in a subdirectory of a repo — see
   [go.dev/ref/mod#vcs-version](https://go.dev/ref/mod#vcs-version)).
3. Publishes the other 3 SDKs (nodejs, python, dotnet) to their respective
   registries. **PyPI is gated on its secret being configured** — skipped,
   not failed, if you haven't onboarded it yet, so you can cut binary-only
   releases before every registry is wired up. npm and NuGet use trusted
   publishing instead of a stored secret, so those two steps always run
   rather than being skippable this way - see
   [Required secrets](#required-secrets) below. In practice, onboard npm and
   NuGet's registry-side trusted-publisher policies *before* the first tag
   that's meant to reach them, otherwise that step fails loudly instead of
   quietly skipping.

There's no Java/Maven SDK - dropped as not worth the setup cost (Sonatype
namespace verification, GPG-signed releases) given how little of the
Pulumi ecosystem uses Java.

A tag with a prerelease segment (`-alpha.1`, `-beta.2`, ...) is automatically
marked as a prerelease on GitHub (`release.prerelease: auto` in
`.goreleaser.yml`) and published to npm under the `next` dist-tag instead of
`latest` (npm refuses to publish a prerelease version to `latest`).

## Required secrets

Configure these as [repository secrets](https://github.com/axnic/pulumi-pocket-id/settings/secrets/actions).
None are required to release the provider binary itself — only to publish
the corresponding language SDK. npm and NuGet no longer need a stored
long-lived token at all — they publish via each registry's OIDC-based
*trusted publishing*, which exchanges this workflow's own identity for a
short-lived credential at publish time. That needs `permissions: id-token:
write` on the `publish-sdks` job (already set in
[`release.yml`](.github/workflows/release.yml)) plus a one-time
registry-side setup described below - no GitHub secret to rotate or leak.

| Secret | Registry | Used by |
| --- | --- | --- |
| — (trusted publishing) | [npmjs.com](https://www.npmjs.com) | Node.js SDK (`@axnic/pulumi-pocket-id`) |
| `PYPI_API_TOKEN` | [PyPI](https://pypi.org) | Python SDK (`pulumi_pocket_id`) |
| `NUGET_USER` | [NuGet.org](https://www.nuget.org) | .NET SDK (`Axnic.Pulumi.PocketId`) — trusted publishing, `NUGET_USER` is just the NuGet.org username, not a credential |

### Obtaining each secret

- **npm (no secret — trusted publishing)** — on the `@axnic/pulumi-pocket-id`
  package's [npmjs.com](https://www.npmjs.com) settings page, under
  *Trusted Publisher*, add a GitHub Actions publisher pointing at
  `axnic/pulumi-pocket-id`, workflow file `release.yml`. Requires npm CLI
  ≥ 11.5.1 (the mise-pinned Node.js version already ships a newer one - see
  `.config/mise.toml`) and the workflow's `id-token: write` permission,
  which is what lets `npm publish --provenance` mint the OIDC-backed
  credential instead of reading `NODE_AUTH_TOKEN`.
- **`PYPI_API_TOKEN`** — create a [PyPI API
  token](https://pypi.org/help/#apitoken) scoped to the `pulumi_pocket_id`
  project (or your account, before the project exists yet). PyPI trusted
  publishing exists too but isn't set up here yet; this token is the only
  registry still using a stored secret.
- **`NUGET_USER`** (trusted publishing) — on
  [NuGet.org](https://www.nuget.org), configure a Trusted Publishing policy
  for the `Axnic.Pulumi.PocketId` package pointing at `axnic/pulumi-pocket-id`'s
  `release.yml` workflow, then set `NUGET_USER` to your NuGet.org
  username (not a secret in the sensitive sense - it's just what the
  [`NuGet/login`](https://github.com/NuGet/login) action uses to look up
  the right trusted-publishing policy before exchanging this workflow's
  OIDC token for a short-lived API key at publish time).

  pulumi-garage hit a snag naming its NuGet package `Pulumi.Garage`: NuGet
  Trusted Publishing likely can't originate a package ID that's never
  existed before (a fast ~350ms "already exists" 409 with nothing actually
  published, best explained by the OIDC-derived key's scope only covering
  new *versions* of a package the policy already recognizes). Naming this
  package `Axnic.Pulumi.PocketId` from the start sidesteps that, but if the
  first real publish still fails the same way, either confirm NuGet Trusted
  Publishing now supports first-time package creation, or do a one-time
  manual seed push with a regular NuGet API key to create the package,
  after which trusted publishing should work normally for every version
  after.

## Cutting a release

1. Make sure the release branch is green (CI passing) and has everything you
   want to release.
2. Tag and push:
   ```sh
   git tag v1.0.0
   git push origin v1.0.0
   ```
   For a prerelease: `git tag v1.0.0-alpha.1`.
3. Watch the [`Release` workflow
   run](https://github.com/axnic/pulumi-pocket-id/actions/workflows/release.yml).
   The GitHub Release itself (provider binaries) always runs. npm and NuGet
   always attempt to publish (trusted publishing, no secret gate); PyPI only
   runs once its secret is configured - see
   [Required secrets](#required-secrets).
4. Verify: `pulumi plugin install resource pocket-id <version>` (or let `pulumi
   up` resolve it automatically from the provider's `pluginDownloadURL`),
   and check the registries you published to for the new package version.

## Retrying a failed publish

Publishing is not currently idempotent-safe to blindly re-run for every
registry (npm and PyPI both reject re-publishing the same version; NuGet's
`--skip-duplicate` tolerates retries better). If one SDK's publish step
fails:

- Fix the underlying issue (expired token, missing namespace claim, etc.).
- Re-run just the failed job from the Actions UI ("Re-run failed jobs") —
  the GitHub Release and Go SDK tag steps are idempotent and safe to
  re-trigger; check the registry's own state before retrying npm/PyPI if
  partial upload is a concern.

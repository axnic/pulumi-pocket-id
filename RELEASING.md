# Releasing

This document is for maintainers cutting a release of the provider and its SDKs.

## How a release works

A release is started by hand from **Actions > Release** (workflow file
`workflow_dispatch.release.yaml`, a thin caller of the central reusable
workflows in [axnic/.github](https://github.com/axnic/.github)). Pushing a
tag does **not** publish anything. The run has two jobs:

1. `prepare` computes (or validates) the version, runs the local quality gate
   (`mise run ci`: lint, build, tests) and outputs the `tag`, `version` and
   `prerelease` values used by `publish`.
2. `publish` builds the provider binaries for darwin/linux/windows
   (amd64+arm64) with [GoReleaser](.goreleaser.yml), publishes them as a
   GitHub Release with checksums, and publishes the nodejs, python and dotnet
   SDKs to their registries. The binaries alone are enough for
   `pulumi plugin install resource pocket-id <version>` to work: the provider
   resolves straight from this repo's GitHub Releases (`WithPluginDownloadURL`
   in `provider/provider.go`), no Pulumi Registry listing required.

The exact steps live in the central workflows; see the
[axnic/.github wiki](https://github.com/axnic/.github/wiki).

There's no Java/Maven SDK - dropped as not worth the setup cost (Sonatype
namespace verification, GPG-signed releases) given how little of the
Pulumi ecosystem uses Java.

### Choosing the version

The `Release` workflow takes two inputs, and you must set **exactly one** of
them (the run fails immediately otherwise):

- `bump`: `auto`, `patch`, `minor` or `major`. The next version is derived
  from the last tag.
- `version`: an explicit version such as `1.0.0`, or `0.2.0-rc.1` for a
  release candidate.

There is no dedicated release-candidate bump: to release a candidate, pass
`version` with a prerelease segment (`-rc.1`, `-alpha.1`, ...). A version with
a prerelease segment is flagged as a prerelease (`prerelease` output of the
`prepare` job).

An optional `notes` input overrides the generated release notes.

## Required secrets

Configure these as [repository secrets](https://github.com/axnic/pulumi-pocket-id/settings/secrets/actions).
None are required to release the provider binary itself, only to publish the
corresponding language SDK. Each is optional: when one is missing, the
publish of its SDK is skipped rather than failed, so you can cut
binary-only releases before every registry is wired up.

| Secret | Registry | Used by |
| --- | --- | --- |
| `NPM_TOKEN` | [npmjs.com](https://www.npmjs.com) | Node.js SDK (`@axnic/pulumi-pocket-id`) |
| `PYPI_API_TOKEN` | [PyPI](https://pypi.org) | Python SDK (`pulumi_pocket_id`) |
| `NUGET_API_KEY` | [NuGet.org](https://www.nuget.org) | .NET SDK (`Axnic.Pulumi.PocketId`) |

The OIDC / trusted-publisher setup for the registries is described in the
[axnic/.github wiki](https://github.com/axnic/.github/wiki).

Note on the .NET package ID: NuGet may refuse to create a brand-new package ID
from a short-lived or scoped credential. If the first publish of
`Axnic.Pulumi.PocketId` fails with an "already exists" error and nothing was
published, do a one-time manual seed push with a regular NuGet API key, then
retry.

## Cutting a release

1. Make sure `main` is green (CI passing) and has everything you want to
   release.
2. Go to **Actions > Release > Run workflow** and set exactly one of `bump`
   or `version`. Or from the CLI:
   ```sh
   gh workflow run workflow_dispatch.release.yaml --field bump=minor
   gh workflow run workflow_dispatch.release.yaml --field version=0.2.0-rc.1
   ```
3. Watch the [`Release` workflow
   run](https://github.com/axnic/pulumi-pocket-id/actions/workflows/workflow_dispatch.release.yaml).
   The GitHub Release itself (provider binaries) always runs; each SDK
   publish only runs when its secret is configured, see
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
  check the registry's own state before retrying npm/PyPI if
  partial upload is a concern.

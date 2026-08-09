[![GitHub release](https://img.shields.io/github/v/release/axnic/pulumi-pocket-id?logo=github&sort=semver)](https://github.com/axnic/pulumi-pocket-id/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/axnic/pulumi-pocket-id/sdk/go/pulumi-pocket-id.svg)](https://pkg.go.dev/github.com/axnic/pulumi-pocket-id/sdk/go/pulumi-pocket-id)
[![npm version](https://img.shields.io/npm/v/%40axnic%2Fpulumi-pocket-id.svg)](https://www.npmjs.com/package/@axnic/pulumi-pocket-id)
[![PyPI version](https://img.shields.io/pypi/v/pulumi_pocket_id.svg)](https://pypi.org/project/pulumi_pocket_id/)
[![NuGet version](https://img.shields.io/nuget/v/Axnic.Pulumi.PocketId.svg)](https://www.nuget.org/packages/Axnic.Pulumi.PocketId/)
[![License](https://img.shields.io/github/license/axnic/pulumi-pocket-id.svg)](LICENSE)

# Pulumi Pocket-ID Provider

A native [Pulumi](https://www.pulumi.com/) provider for [Pocket-ID](https://github.com/pocket-id/pocket-id),
a passkey-only OIDC identity provider, through its REST API.

It manages four resources: **`OidcClient`**, **`User`**, **`UserGroup`**, and
**`CustomClaims`**. Bootstrapping the Pocket-ID instance itself (deployment, initial
admin passkey enrollment) is out of scope - see [Known limitations](#known-limitations).

## Installing

This package is available in several languages. Only the Go and YAML SDKs are
exercised by this repo's own examples and acceptance tests (see
[Known limitations](#known-limitations)); the others are generated and published for
convenience but are otherwise untested by this project beyond schema-level checks.

### Node.js (JavaScript/TypeScript)

```bash
npm install @axnic/pulumi-pocket-id
```

### Python

```bash
pip install pulumi_pocket_id
```

### Go

```bash
go get github.com/axnic/pulumi-pocket-id/sdk/go/pulumi-pocket-id
```

### .NET

```bash
dotnet add package Axnic.Pulumi.PocketId
```

The provider binary itself doesn't require any of the above - `pulumi plugin install
resource pocket-id <version>` (or the engine's automatic resolution) fetches it
directly from this repo's [GitHub Releases](https://github.com/axnic/pulumi-pocket-id/releases),
no Pulumi Registry listing required.

## Quickstart

Point the provider at your Pocket-ID instance, via stack config:

```bash
pulumi config set pocket-id:baseUrl https://pocketid.example.com
pulumi config set pocket-id:apiKey <api-key> --secret
```

This provider does not deploy or bootstrap a Pocket-ID instance for you - see
[pocket-id/pocket-id](https://github.com/pocket-id/pocket-id) for running one, and
generate an API key from its admin UI (Settings → API Keys).

Then, a minimal YAML program (see [`examples/yaml/Pulumi.yaml`](examples/yaml/Pulumi.yaml),
or [`examples/go/main.go`](examples/go/main.go) for the Go equivalent):

```yaml
name: pocket-id-example-yaml
runtime: yaml

resources:
  myClient:
    type: pocket-id:index:OidcClient
    properties:
      name: My Application
      callbackUrls:
        - https://app.example.com/callback

outputs:
  clientId: ${myClient.clientId}
```

## Configuration

| Key | Description | Secret |
| --- | --- | --- |
| `pocket-id:baseUrl` | Base URL of the Pocket-ID instance, e.g. `https://pocketid.example.com` | no |
| `pocket-id:apiKey` | A Pocket-ID API key with permissions for the resources you manage | yes |

Both are required - the provider fails to configure if either is unset.

## Resources

| Resource | Pocket-ID API |
| --- | --- |
| `pocket-id:index:OidcClient` | `/api/oidc/clients` |
| `pocket-id:index:User` | `/api/users` |
| `pocket-id:index:UserGroup` | `/api/user-groups` |
| `pocket-id:index:CustomClaims` | `/api/custom-claims` |

## Known limitations

This is a v1, personal/small-org provider - scope is intentionally narrow:

- **No instance bootstrapping.** Deploying Pocket-ID and enrolling the first admin
  passkey is not managed by this provider; do that yourself before pointing this
  provider at an instance.
- **No env var config fallback.** Unlike some Pulumi providers, `baseUrl`/`apiKey`
  must be set via Pulumi config (`pulumi config set`) - there's no
  `POCKET_ID_BASE_URL`/`POCKET_ID_API_KEY` fallback read by the provider itself
  (those env vars are only used by this repo's own test suite, see
  [CONTRIBUTING.md](CONTRIBUTING.md)).
- **Only YAML and Go example programs are exercised in CI/E2E.** The nodejs, python,
  and dotnet SDKs are generated and published (see [Installing](#installing)) but
  aren't covered by this repo's example programs or lifecycle tests.
- **No Java/Maven SDK.** Pulumi's Java support has too little adoption to justify the
  extra setup cost (Sonatype namespace verification, GPG-signed releases); the nodejs,
  python, dotnet, and go SDKs cover the ecosystem's actual usage.

## Reference

This provider isn't (yet) listed on the [Pulumi Registry](https://www.pulumi.com/registry/),
so there's no `pulumi.com/registry/packages/pocket-id` page to link to for generated,
per-resource API docs. Until then:

- The **Go SDK reference** on [pkg.go.dev](https://pkg.go.dev/github.com/axnic/pulumi-pocket-id/sdk/go/pulumi-pocket-id)
  is generated from the same schema descriptions the Pulumi Registry would render, and
  is the most complete field-by-field reference available (populated after the first
  tagged release - see [RELEASING.md](RELEASING.md)).
- The [`Quickstart`](#quickstart), [`Configuration`](#configuration), and
  [`Resources`](#resources) sections above cover the four resources and provider
  config end to end; the underlying schema is
  `provider/cmd/pulumi-resource-pocket-id/schema.json` (generated, not hand-maintained).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for building the provider, the test layout
(including how to run the acceptance suite locally), generating the SDKs, and the
devcontainer setup. See also our [Code of Conduct](CODE-OF-CONDUCT.md).

## License

[Apache-2.0](LICENSE)

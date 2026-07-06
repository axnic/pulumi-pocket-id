# Pulumi Pocket-ID Provider

A native [Pulumi](https://www.pulumi.com/) provider for [Pocket-ID](https://github.com/pocket-id/pocket-id), a passkey-only OIDC identity provider.

This provider lets you manage Pocket-ID resources (OIDC clients, users, user groups, and custom claims) declaratively from Pulumi programs.

## Resources

| Resource | Pocket-ID API |
| --- | --- |
| `pocket-id:index:OidcClient` | `/api/oidc/clients` |
| `pocket-id:index:User` | `/api/users` |
| `pocket-id:index:UserGroup` | `/api/user-groups` |
| `pocket-id:index:CustomClaims` | `/api/custom-claims` |

## Configuration

The provider authenticates to Pocket-ID using an API key (sent via the `X-API-Key` header). Set the following configuration values:

| Config | Description | Secret |
| --- | --- | --- |
| `pocket-id:baseUrl` | Base URL of the Pocket-ID instance, e.g. `https://pocketid.example.com` | no |
| `pocket-id:apiKey` | A Pocket-ID API key with permissions for the resources you manage | yes |

### Example (YAML)

```yaml
config:
  pocket-id:baseUrl: https://pocketid.example.com
  pocket-id:apiKey:
    secure: <encrypted-api-key>

resources:
  myClient:
    type: pocket-id:index:OidcClient
    properties:
      name: My Application
      callbackUrls:
        - https://app.example.com/callback
```

## Building

Requires [Go](https://golang.org/) (1.25+) and the [Pulumi CLI](https://www.pulumi.com/docs/install/).

```bash
# Build the provider binary
go build -o pulumi-resource-pocket-id ./provider/cmd/pulumi-resource-pocket-id

# Generate the Pulumi schema
pulumi package get-schema ./pulumi-resource-pocket-id > provider/schema.json

# Run tests
go test ./tests/...
```

## Development

This provider is built with [pulumi-go-provider](https://github.com/pulumi/pulumi-go-provider) (the Layer 3 SDK). Resource implementations live in `provider/`:

- `provider/client.go` - HTTP client with `X-API-Key` authentication.
- `provider/oidc_client.go` - OIDC client resource.
- `provider/user.go` - User resource.
- `provider/user_group.go` - User group resource.
- `provider/custom_claims.go` - Custom claims resource.
- `provider/provider.go` - Provider wiring.

## References

* [Pocket-ID](https://github.com/pocket-id/pocket-id)
* [Pulumi Go Provider](https://github.com/pulumi/pulumi-go-provider)
* [Building a Pulumi provider](https://www.pulumi.com/docs/iac/using-pulumi/pulumi-packages/how-to-author/)

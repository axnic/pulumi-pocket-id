[![GitHub release](https://img.shields.io/github/v/release/axnic/pulumi-pocket-id?logo=github&sort=semver)](https://github.com/axnic/pulumi-pocket-id/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/axnic/pulumi-pocket-id/sdk/go/pulumi-pocket-id.svg)](https://pkg.go.dev/github.com/axnic/pulumi-pocket-id/sdk/go/pulumi-pocket-id)
[![npm version](https://img.shields.io/npm/v/%40axnic%2Fpulumi-pocket-id.svg)](https://www.npmjs.com/package/@axnic/pulumi-pocket-id)
[![PyPI version](https://img.shields.io/pypi/v/pulumi_pocket_id.svg)](https://pypi.org/project/pulumi_pocket_id/)
[![NuGet version](https://img.shields.io/nuget/v/Axnic.Pulumi.PocketId.svg)](https://www.nuget.org/packages/Axnic.Pulumi.PocketId/)
[![License](https://img.shields.io/github/license/axnic/pulumi-pocket-id.svg)](LICENSE)

# Pulumi Pocket-ID Provider

A native [Pulumi](https://www.pulumi.com/) provider for [Pocket-ID](https://github.com/pocket-id/pocket-id),
a passkey-only OIDC identity provider, through its REST API.

It manages users, user groups and their membership, custom claims, OIDC clients and
their secrets, APIs with client grants, SCIM service providers, signup
tokens and the instance-wide application configuration, and offers lookup functions
(`getUser`, `getUserGroup`, `getOidcClient`, `getOpenIdConfiguration`). See the
[Resource reference](#resource-reference). Bootstrapping the Pocket-ID instance itself
(deployment, first admin passkey) is out of scope - see [Known limitations](#known-limitations).

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

## Prerequisites

- A running Pocket-ID instance reachable from where Pulumi runs. Pocket-ID **v2.3.0 or
  later** is the tested baseline (see [Compatibility](#compatibility)).
- An API key with admin rights: either a key created in the admin UI (Settings, API Keys)
  or the instance's `STATIC_API_KEY` (an environment variable of Pocket-ID, available
  from v2.3.0, that authenticates as a synthetic admin).
- Pulumi CLI 3.x.

## Quickstart

Point the provider at your instance, via stack config:

```bash
pulumi config set pocket-id:baseUrl https://pocketid.example.com
pulumi config set pocket-id:apiKey <api-key> --secret
```

or the `POCKET_ID_BASE_URL` / `POCKET_ID_API_KEY` environment variables.

Then a program (see [`examples/yaml/Pulumi.yaml`](examples/yaml/Pulumi.yaml), or
[`examples/go/main.go`](examples/go/main.go) for the Go equivalent) such as this one: a
group with a member, an OIDC client only that group may sign in to, a client secret, and
an API the client may call.

```yaml
name: pocket-id-example-yaml
runtime: yaml

resources:
  engineers:
    type: pocket-id:index:UserGroup
    properties:
      name: engineers
      friendlyName: Engineers
  alice:
    type: pocket-id:index:User
    properties:
      username: alice
      email: alice@example.com
      firstName: Alice
      displayName: Alice Liddell
  engineersMembers:
    type: pocket-id:index:UserGroupMembers
    properties:
      groupId: ${engineers.id}
      userIds:
        - ${alice.id}
  app:
    type: pocket-id:index:OidcClient
    properties:
      name: My Application
      callbackUrls:
        - https://app.example.com/callback
      allowedUserGroupIds:
        - ${engineers.id}
  appSecret:
    type: pocket-id:index:OidcClientSecret
    properties:
      clientId: ${app.id}
  backend:
    type: pocket-id:index:Api
    properties:
      name: Backend API
      resource: https://api.example.com
      permissions:
        - key: read
          name: Read
  appBackendAccess:
    type: pocket-id:index:ApiClientGrant
    properties:
      apiId: ${backend.id}
      clientId: ${app.id}
      clientAccess: true
      clientPermissionKeys:
        - read

outputs:
  clientId: ${app.clientId}
  clientSecret: ${appSecret.secret}
```

## Resource reference

The tables below are generated from the `Annotate()` descriptions in `provider/*.go` (the same text as the schema and the SDK doc comments). Resource tokens are `pocket-id:index:<Name>`.

### Users and groups

Identity objects. `UserGroupMembers` is the owning side of the group membership relation: it is authoritative (it sets the complete member list), so do not also set group memberships from another resource.

#### `User`

A Pocket-ID user. Group membership is managed with the UserGroupMembers resource.

Inputs:

| Property | Type | Description |
| --- | --- | --- |
| `disabled` | boolean | Whether the user is disabled and cannot sign in. |
| `displayName` | string | The name displayed for the user. Pocket-ID may derive it from the first and last names when unset. |
| `email` | string | The email address of the user. |
| `emailVerified` | boolean | Whether the email address is considered verified. |
| `firstName` | string | The first name of the user. |
| `isAdmin` | boolean | Whether the user is an administrator. |
| `lastName` | string | The last name of the user. |
| `locale` | string | The locale of the user, for example en or fr. |
| `username` (required) | string | The unique login name of the user. |

#### `UserGroup`

A Pocket-ID user group. Members are managed with the UserGroupMembers resource and custom claims with the UserGroupCustomClaims resource.

Inputs:

| Property | Type | Description |
| --- | --- | --- |
| `friendlyName` (required) | string | The human-readable name of the group. |
| `name` (required) | string | The unique technical name of the group. |

#### `UserGroupMembers`

The authoritative set of members of a user group. Destroying the resource removes every member from the group.

Inputs:

| Property | Type | Description |
| --- | --- | --- |
| `groupId` (required) | string | The ID of the group. Changing it replaces the resource. |
| `userIds` (required) | string[] | The IDs of all users that belong to the group. The set is authoritative: users not listed are removed from the group. |

#### `UserCustomClaims`

The authoritative set of custom claims of a user. Destroying the resource removes every custom claim from the user.

Inputs:

| Property | Type | Description |
| --- | --- | --- |
| `claims` (required) | map<string,string> | The custom claims of the user, as key/value pairs. The set is authoritative: claims not listed are removed from the user. |
| `userId` (required) | string | The ID of the user. Changing it replaces the resource. |

#### `UserGroupCustomClaims`

The authoritative set of custom claims of a user group. Destroying the resource removes every custom claim from the group.

Inputs:

| Property | Type | Description |
| --- | --- | --- |
| `claims` (required) | map<string,string> | The custom claims of the group, as key/value pairs. The set is authoritative: claims not listed are removed from the group. |
| `userGroupId` (required) | string | The ID of the user group. Changing it replaces the resource. |

### OIDC clients

An OIDC client and its credentials. Which groups may use a client is set on the client itself (`allowedUserGroupIds`).

#### `OidcClient`

An OIDC client of Pocket-ID. The client never creates a secret by itself; use OidcClientSecret to generate one. The allowed user groups are owned by this resource. The logos are set either from a URL (logoUrl, darkLogoUrl) or from an uploaded file (logo, darkLogo), never both for the same variant.

Inputs:

| Property | Type | Description |
| --- | --- | --- |
| `accessTokenDurationMinutes` | integer | The access token lifetime in minutes. Pocket-ID's default applies when unset. |
| `allowedUserGroupIds` | string[] | The IDs of the user groups allowed to use the client. Authoritative: the set is replaced on every update, and the client is restricted to these groups when the list is non-empty (unrestricted when empty). |
| `backchannelLogoutUrl` | string | The URL called for OIDC back-channel logout. Requires Pocket-ID v2.17.0 or later: older versions ignore the field, so the provider fails instead of silently dropping it. |
| `callbackUrls` | string[] | The allowed redirect URLs. |
| `clientId` | string | The ID of the client. Generated by Pocket-ID when unset. Changing it replaces the client. |
| `credentials` | OidcClientCredentials | The credentials of the client managed here (federated identities). |
| `darkLogo` | asset | The dark mode client logo (PNG, JPG or SVG) uploaded from a file, as an asset, e.g. a FileAsset. Archives are not supported. A change of content re-uploads the file and removing it deletes the logo. Mutually exclusive with darkLogoUrl. An asset cannot be read back: after an import it is empty and must be set in the program, which uploads the logo again if it differs from the served one. |
| `darkLogoUrl` | string | A URL from which Pocket-ID fetches the dark mode logo. When unset, the dark logo is removed. It is not read back from the API. Mutually exclusive with darkLogo (uploaded file). |
| `description` | string | A description of the client (150 characters max). |
| `isPublic` | boolean | Whether the client is public (no client secret, PKCE recommended). |
| `launchUrl` | string | The URL used to launch the application from the Pocket-ID dashboard. |
| `logo` | asset | The client logo (PNG, JPG or SVG) uploaded from a file, as an asset, e.g. a FileAsset. Archives are not supported. A change of content re-uploads the file and removing it deletes the logo. Mutually exclusive with logoUrl. An asset cannot be read back: after an import it is empty and must be set in the program, which uploads the logo again if it differs from the served one. |
| `logoUrl` | string | A URL from which Pocket-ID fetches the client logo. When unset, the logo is removed. It is not read back from the API. Mutually exclusive with logo (uploaded file). |
| `logoutCallbackUrls` | string[] | The allowed post-logout redirect URLs. |
| `name` (required) | string | The display name of the client (50 characters max). |
| `pkceEnabled` | boolean | Whether PKCE is required for the client. |
| `refreshTokenDurationMinutes` | integer | The refresh token lifetime in minutes. Pocket-ID's default applies when unset. |
| `requiresPushedAuthorizationRequests` | boolean | Whether the client must use pushed authorization requests (PAR). |
| `requiresReauthentication` | boolean | Whether users must re-authenticate on every authorization. |
| `skipConsent` | boolean | Whether the consent screen is skipped for this client. |

Outputs (in addition to the inputs):

| Property | Type | Description |
| --- | --- | --- |
| `clientType` | string | The client type reported by Pocket-ID (confidential or public). |
| `darkLogoSha256` | string | The hex SHA-256 digest of the uploaded dark mode logo (see darkLogo). At refresh it is the digest of the bytes currently served by Pocket-ID, so out-of-band changes show up as a diff. |
| `hasDarkLogo` | boolean | Whether the client has a dark mode logo. |
| `hasLogo` | boolean | Whether the client has a logo. |
| `isGroupRestricted` | boolean | Whether access is restricted to the allowed user groups. |
| `logoSha256` | string | The hex SHA-256 digest of the uploaded logo (see logo). At refresh it is the digest of the bytes currently served by Pocket-ID, so out-of-band changes show up as a diff. |
| `pkceSupported` | boolean | Whether PKCE is supported for this client. |

#### `OidcClientSecret`

A client secret of an OIDC client. Several secrets can coexist, which allows rotating a secret without downtime: create the new one, then remove the old one. Every property is immutable: changing one replaces the secret.

Inputs:

| Property | Type | Description |
| --- | --- | --- |
| `clientId` (required) | string | The ID of the OIDC client the secret belongs to. Changing it replaces the secret. |
| `expiresAt` | string | The expiration date of the secret (RFC 3339). Never expires when unset. Changing it replaces the secret. |

Outputs (in addition to the inputs):

| Property | Type | Description |
| --- | --- | --- |
| `createdAt` | string | The creation date of the secret. |
| `prefix` | string | The first characters of the secret, as reported by Pocket-ID. |
| `secret` (secret) | string | The plain-text secret. Only returned by Pocket-ID at creation, so it is kept from the state and is empty after an import. |
| `secretId` | string | The ID of the secret. |

### APIs and access

Protected APIs (resources), and the clients allowed to call them. Requires a Pocket-ID version that serves these endpoints; see [Compatibility](#compatibility).

#### `Api`

A Pocket-ID API (OAuth resource server) and its permissions.

Inputs:

| Property | Type | Description |
| --- | --- | --- |
| `name` (required) | string | The display name of the API. |
| `permissions` | ApiPermissionArgs[] | The permissions of the API. The list is authoritative: permissions not listed here are removed. |
| `cimdAccess` | { enabled: boolean, permissionKeys: string[] } | The access granted to clients registered through a Client ID Metadata Document (CIMD). When set it is authoritative, and every key of `permissionKeys` must be declared in `permissions`; when omitted the server value is left untouched. |
| `resource` (required) | string | The resource identifier (audience) of the API, usually a URL. Changing it replaces the API. |

Outputs (in addition to the inputs):

| Property | Type | Description |
| --- | --- | --- |
| `createdAt` | string | The RFC 3339 timestamp at which the API was created. |

#### `ApiClientGrant`

The access of an OIDC client on an API. Authoritative for the (API, client) pair; import with the ID "<apiId>/<clientId>".

Inputs:

| Property | Type | Description |
| --- | --- | --- |
| `apiId` (required) | string | The ID of the API. Changing it replaces the grant. |
| `clientAccess` | boolean | Whether the client may access the API on its own behalf (client credentials flow). |
| `clientId` (required) | string | The ID of the OIDC client. Changing it replaces the grant. |
| `clientPermissionKeys` | string[] | The keys of the API permissions granted to the client on its own behalf. |
| `userDelegatedAccess` | boolean | Whether the client may access the API on behalf of users. |
| `userDelegatedPermissionKeys` | string[] | The keys of the API permissions the client may use on behalf of users. |

### Provisioning

SCIM provisioning to an external service, and invitation tokens for self-service sign-up.

#### `ScimServiceProvider`

A SCIM service provider attached to an OIDC client. To import it, use the ID <oidcClientId>/<serviceProviderId>.

Inputs:

| Property | Type | Description |
| --- | --- | --- |
| `endpoint` (required) | string | The base URL of the SCIM endpoint. |
| `oidcClientId` (required) | string | The ID of the OIDC client the service provider belongs to. Changing it replaces the resource. |
| `token` (secret) | string | The bearer token used to authenticate against the SCIM endpoint. Never read back from the API: it is kept from the state and unset after an import. |

Outputs (in addition to the inputs):

| Property | Type | Description |
| --- | --- | --- |
| `createdAt` | string | The creation date of the service provider. |
| `lastSyncedAt` | string | The date of the last synchronization, empty if it never ran. |

#### `SignupToken`

A Pocket-ID signup token that lets new users create their own account.

Inputs:

| Property | Type | Description |
| --- | --- | --- |
| `ttl` (required) | integer | The lifetime of the token in seconds. Changing it replaces the resource. |
| `usageLimit` (required) | integer | How many times the token can be used to sign up. Changing it replaces the resource. |
| `userGroupIds` | string[] | The IDs of the groups users signing up with this token join. Changing them replaces the resource. |

Outputs (in addition to the inputs):

| Property | Type | Description |
| --- | --- | --- |
| `createdAt` | string | The RFC 3339 timestamp at which the token was created. |
| `expiresAt` | string | The RFC 3339 timestamp at which the token expires. |
| `token` (secret) | string | The secret signup token. It is captured at creation and carried in state; it is not read back from the API, so it is empty after an import. |

### Images

The global branding images of the instance, uploaded from a local file. (The logo of an OIDC client is set on the `OidcClient` itself, with `logo` and `darkLogo`.) The `image` input is a Pulumi asset (for example a `FileAsset`); archives are not supported. The upload is idempotent: a change of the file content updates the resource by uploading it again, and out-of-band changes are detected at refresh by comparing the SHA-256 of the bytes served by Pocket-ID with the one in state. Because the image is not read back as an asset, `image` must be set in the program after an import.

#### `ApplicationImage`

A global Pocket-ID branding image (logo, dark logo, favicon, background, e-mail image or default profile picture) uploaded from a local file or asset. Destroying it deletes the custom image and restores the default, except for the favicon and the e-mail image, which the API cannot delete: destroying those only removes the resource from state. Import with the type, e.g. "favicon". Per-user profile pictures are not covered.

Inputs:

| Property | Type | Description |
| --- | --- | --- |
| `image` (required) | asset | The image file as an asset, e.g. a FileAsset. Archives are not supported. A change of content re-uploads the file. It is not read back as an asset: after an import it is empty and must be set in the program. |
| `type` (required) | string | Which image to manage. Changing it replaces the resource. |

Outputs (in addition to the inputs):

| Property | Type | Description |
| --- | --- | --- |
| `sha256` | string | The hex SHA-256 digest of the image content. At refresh it is the digest of the bytes currently served by Pocket-ID, so out-of-band changes show up as a diff. |

### Instance settings

The instance-wide configuration (general, email, SMTP, LDAP, WebAuthn).

#### `ApplicationConfiguration`

The instance-wide configuration of Pocket-ID. It is a singleton with the fixed ID "app-config": only the fields that are set are managed, the others keep their current value on the server, and destroying the resource leaves the configuration untouched. Use at most one per instance.

Inputs:

| Property | Type | Description |
| --- | --- | --- |
| `accentColor` | string | The UI accent color, as "default" or a CSS color value. |
| `allowOwnAccountEdit` | boolean | Whether users may edit their own account details. |
| `allowUserSignups` | string | Who may sign up by themselves: "disabled", "withToken" or "open". |
| `appName` | string | The application name shown in the UI and in emails (1 to 30 characters). |
| `autoCreateOidcClientSecret` | boolean | Whether a secret is generated automatically when an OIDC client is created. |
| `cimdUrlAllowlist` | string | The allowlist of URLs accepted as OIDC client metadata documents (CIMD), in the format Pocket-ID expects. |
| `disableAnimations` | boolean | Whether to disable the UI animations. |
| `emailApiKeyExpirationEnabled` | boolean | Whether an email is sent when an API key is about to expire. |
| `emailLoginNotificationEnabled` | boolean | Whether an email is sent when a user signs in from a new device. |
| `emailOneTimeAccessAsAdminEnabled` | boolean | Whether administrators may send one-time access links by email. |
| `emailOneTimeAccessAsUnauthenticatedEnabled` | boolean | Whether unauthenticated users may request a one-time access link by email. |
| `emailVerificationEnabled` | boolean | Whether users must verify their email address. |
| `emailsVerified` | boolean | Whether emails are considered verified by default. |
| `homePageUrl` | string | The path of the page users land on after signing in, e.g. "/settings/account". |
| `ldapAdminGroupName` | string | The name of the LDAP group whose members become Pocket-ID administrators. |
| `ldapAttributeGroupMember` | string | The LDAP attribute listing the members of a group. |
| `ldapAttributeGroupName` | string | The LDAP attribute mapped to the group name. |
| `ldapAttributeGroupUniqueIdentifier` | string | The LDAP attribute holding the unique identifier of a group. |
| `ldapAttributeUserDisplayName` | string | The LDAP attribute mapped to the display name. |
| `ldapAttributeUserEmail` | string | The LDAP attribute mapped to the email address. |
| `ldapAttributeUserFirstName` | string | The LDAP attribute mapped to the first name. |
| `ldapAttributeUserLastName` | string | The LDAP attribute mapped to the last name. |
| `ldapAttributeUserProfilePicture` | string | The LDAP attribute mapped to the profile picture. |
| `ldapAttributeUserUniqueIdentifier` | string | The LDAP attribute holding the unique identifier of a user. |
| `ldapAttributeUserUsername` | string | The LDAP attribute mapped to the user name. |
| `ldapBase` | string | The LDAP base DN under which users and groups are searched. |
| `ldapBindDn` | string | The distinguished name used to bind to the LDAP server. |
| `ldapBindPassword` (secret) | string | The LDAP bind password. It is a secret and is never read back from Pocket-ID: the value in state is the one from the program, and it is empty after an import. |
| `ldapEnabled` | boolean | Whether LDAP synchronization is enabled. |
| `ldapSkipCertVerify` | boolean | Whether to skip the verification of the LDAP server certificate. |
| `ldapSoftDeleteUsers` | boolean | Whether users removed from LDAP are disabled instead of deleted. |
| `ldapUrl` | string | The LDAP server URL, e.g. "ldaps://ldap.example.com:636". |
| `ldapUserGroupSearchFilter` | string | The LDAP filter selecting groups, e.g. "(objectClass=groupOfNames)". |
| `ldapUserSearchFilter` | string | The LDAP filter selecting users, e.g. "(objectClass=person)". |
| `requireUserEmail` | boolean | Whether an email address is required for every user. |
| `sessionDuration` | integer | The session lifetime, in minutes. |
| `signupDefaultCustomClaims` | string | The custom claims given to new users, as the JSON array string Pocket-ID expects. |
| `signupDefaultUserGroupIDs` | string | The IDs of the user groups new users join, as the JSON array string Pocket-ID expects. |
| `smtpFrom` | string | The sender address of outgoing emails. |
| `smtpHost` | string | The SMTP server host name. |
| `smtpPassword` (secret) | string | The SMTP password. It is a secret and is never read back from Pocket-ID: the value in state is the one from the program, and it is empty after an import. |
| `smtpPort` | integer | The SMTP server port. |
| `smtpSkipCertVerify` | boolean | Whether to skip the verification of the SMTP server certificate. |
| `smtpTls` | string | The SMTP transport security: "none", "starttls" or "tls". |
| `smtpUser` | string | The SMTP user name. |
| `webauthnAllowSyncedPasskeys` | boolean | Whether passkeys synced between devices are accepted. |
| `webauthnAuthenticatorAttachment` | string | The accepted authenticator attachment: "any", "platform" or "cross-platform". |
| `webauthnUserVerification` | string | The user verification requirement: "required" or "preferred". |

### Functions

Invokes read data without managing it. Function tokens are `pocket-id:index:<name>`.

#### `getUser`

Looks up a Pocket-ID user by ID or by exact username.

Arguments:

| Property | Type | Description |
| --- | --- | --- |
| `userId` | string | The ID of the user. Exactly one of userId or username must be set. |
| `username` | string | The exact username of the user. Exactly one of userId or username must be set. |

Returns:

| Property | Type | Description |
| --- | --- | --- |
| `customClaims` | map<string,string> | The custom claims of the user. |
| `disabled` | boolean | Whether the user is disabled. |
| `displayName` | string | The display name of the user. |
| `email` | string | The email address of the user, empty if unset. |
| `emailVerified` | boolean | Whether the email address is verified. |
| `firstName` | string | The first name of the user. |
| `isAdmin` | boolean | Whether the user is an administrator. |
| `lastName` | string | The last name of the user, empty if unset. |
| `locale` | string | The locale of the user, empty if unset. |
| `userGroupIds` | string[] | The IDs of the groups the user belongs to. |
| `userId` | string | The ID of the user. |
| `username` | string | The username of the user. |

#### `getUserGroup`

Looks up a Pocket-ID user group by ID or by exact name.

Arguments:

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | The exact name of the group. Exactly one of userGroupId or name must be set. |
| `userGroupId` | string | The ID of the group. Exactly one of userGroupId or name must be set. |

Returns:

| Property | Type | Description |
| --- | --- | --- |
| `customClaims` | map<string,string> | The custom claims of the group. |
| `friendlyName` | string | The human-readable name of the group. |
| `name` | string | The name of the group. |
| `userGroupId` | string | The ID of the group. |
| `userIds` | string[] | The IDs of the members of the group. |

#### `getOidcClient`


Arguments:

| Property | Type | Description |
| --- | --- | --- |
| `clientId` (required) | string | The ID of the OIDC client to look up. |

Returns:

| Property | Type | Description |
| --- | --- | --- |
| `accessTokenDurationMinutes` | integer | The access token lifetime in minutes. |
| `allowedUserGroupIds` | string[] | The IDs of the user groups allowed to use the client. |
| `backchannelLogoutUrl` | string | The back-channel logout URL. |
| `callbackUrls` | string[] | The allowed redirect URLs. |
| `clientId` | string | The ID of the client. |
| `clientType` | string | The client type reported by Pocket-ID. |
| `description` | string | The description of the client. |
| `federatedIdentities` | OidcFederatedIdentity[] | The federated identities allowed to authenticate as the client. |
| `hasDarkLogo` | boolean | Whether the client has a dark mode logo. |
| `hasLogo` | boolean | Whether the client has a logo. |
| `isGroupRestricted` | boolean | Whether access is restricted to the allowed user groups. |
| `isPublic` | boolean | Whether the client is public. |
| `launchUrl` | string | The launch URL of the application. |
| `logoutCallbackUrls` | string[] | The allowed post-logout redirect URLs. |
| `name` | string | The display name of the client. |
| `pkceEnabled` | boolean | Whether PKCE is required. |
| `pkceSupported` | boolean | Whether PKCE is supported. |
| `refreshTokenDurationMinutes` | integer | The refresh token lifetime in minutes. |
| `requiresPushedAuthorizationRequests` | boolean | Whether pushed authorization requests (PAR) are required. |
| `requiresReauthentication` | boolean | Whether users must re-authenticate on every authorization. |
| `skipConsent` | boolean | Whether the consent screen is skipped. |

#### `getOpenIdConfiguration`

Reads the OpenID Connect discovery document of the Pocket-ID instance: issuer, endpoints, JWKS URI and supported capabilities. Use it to configure applications that consume the instance as an OIDC provider.

Returns:

| Property | Type | Description |
| --- | --- | --- |
| `authorizationEndpoint` | string | The OAuth 2.0 authorization endpoint. |
| `claimsSupported` | string[] | The supported claims. |
| `codeChallengeMethodsSupported` | string[] | The supported PKCE code challenge methods. |
| `deviceAuthorizationEndpoint` | string | The device authorization endpoint, empty if the instance does not advertise one. |
| `endSessionEndpoint` | string | The RP-initiated logout endpoint, empty if the instance does not advertise one. |
| `grantTypesSupported` | string[] | The supported OAuth 2.0 grant types. |
| `idTokenSigningAlgValuesSupported` | string[] | The supported ID token signing algorithms. |
| `introspectionEndpoint` | string | The token introspection endpoint, empty if the instance does not advertise one. |
| `issuer` | string | The issuer URL, i.e. the base URL of the instance. |
| `jwksUri` | string | The URL of the JSON Web Key Set used to verify issued tokens. |
| `responseTypesSupported` | string[] | The supported response types. |
| `scopesSupported` | string[] | The supported scopes. |
| `subjectTypesSupported` | string[] | The supported subject identifier types. |
| `tokenEndpoint` | string | The OAuth 2.0 token endpoint. |
| `tokenEndpointAuthMethodsSupported` | string[] | The supported client authentication methods at the token endpoint. |
| `userinfoEndpoint` | string | The OIDC userinfo endpoint. |


## Provider config

| Key | Env var fallback | Description |
| --- | --- | --- |
| `pocket-id:baseUrl` | `POCKET_ID_BASE_URL` | The base URL of the Pocket-ID server, e.g. `"https://pocket-id.example.com"`. |
| `pocket-id:apiKey` (secret) | `POCKET_ID_API_KEY` | A Pocket-ID API key sent with every request in the `X-API-Key` header. |

Both are required, one way or the other: the provider fails to configure if neither the
config key nor its environment variable is set.

## Dev & testing

Tests run at four levels, from hermetic to realistic: client tests against `httptest`,
provider tests against an in-memory fake Pocket-ID (`make test`), an end-to-end suite
against a real Pocket-ID (build tag `e2e`), and the example programs. Typical commands:

```sh
make test                          # client + fake-backed provider tests
mise run ci                        # build, tests with coverage, coverage floor
E2E_VERSION=v2.18.0 mise run ci:e2e # E2E suite against a dockerized Pocket-ID (needs Docker)
make dev-up                        # a local Pocket-ID to point `pulumi up` at
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the details, including how the SDKs are generated.

## Compatibility

The 5 latest Pocket-ID minors are tested by the E2E suite (every resource, create / read / update /
delete / import) in its own CI workflow, so its badge reflects that version alone rather
than an aggregate. Older versions are not tested (to keep CI short); v2.3.0 is the first release with
`STATIC_API_KEY`, which the E2E instance authenticates the provider with, so it is the
absolute floor. E2E subtests for the APIs, SCIM
and signup-token endpoints skip themselves on a version that does not serve them.

| Pocket-ID version | Status |
| --- | --- |
| v2.14.0 | [![E2E (Pocket-ID v2.14.0)](https://github.com/axnic/pulumi-pocket-id/actions/workflows/merge_group%2Cpull_request%2Cpush.e2e-v2.14.0.yaml/badge.svg?branch=main)](https://github.com/axnic/pulumi-pocket-id/actions/workflows/merge_group%2Cpull_request%2Cpush.e2e-v2.14.0.yaml) |
| v2.15.0 | [![E2E (Pocket-ID v2.15.0)](https://github.com/axnic/pulumi-pocket-id/actions/workflows/merge_group%2Cpull_request%2Cpush.e2e-v2.15.0.yaml/badge.svg?branch=main)](https://github.com/axnic/pulumi-pocket-id/actions/workflows/merge_group%2Cpull_request%2Cpush.e2e-v2.15.0.yaml) |
| v2.16.0 | [![E2E (Pocket-ID v2.16.0)](https://github.com/axnic/pulumi-pocket-id/actions/workflows/merge_group%2Cpull_request%2Cpush.e2e-v2.16.0.yaml/badge.svg?branch=main)](https://github.com/axnic/pulumi-pocket-id/actions/workflows/merge_group%2Cpull_request%2Cpush.e2e-v2.16.0.yaml) |
| v2.17.0 | [![E2E (Pocket-ID v2.17.0)](https://github.com/axnic/pulumi-pocket-id/actions/workflows/merge_group%2Cpull_request%2Cpush.e2e-v2.17.0.yaml/badge.svg?branch=main)](https://github.com/axnic/pulumi-pocket-id/actions/workflows/merge_group%2Cpull_request%2Cpush.e2e-v2.17.0.yaml) |
| v2.18.0 | [![E2E (Pocket-ID v2.18.0)](https://github.com/axnic/pulumi-pocket-id/actions/workflows/merge_group%2Cpull_request%2Cpush.e2e-v2.18.0.yaml/badge.svg?branch=main)](https://github.com/axnic/pulumi-pocket-id/actions/workflows/merge_group%2Cpull_request%2Cpush.e2e-v2.18.0.yaml) |

Each workflow (`.github/workflows/merge_group,pull_request,push.e2e-v*.yaml`) is a thin
caller of the central reusable E2E workflow in `axnic/.github` with that version pinned.
New versions are added by that repo's E2E Sync workflow, which opens a PR with the caller
(from `.github/templates/e2e-caller.yaml.tmpl`) and the badge row. GitHub Actions status
badges are per workflow file, not per matrix leg, which is why this is one small file per
version rather than a `strategy.matrix` job.

## Known limitations

This is a personal/small-org provider, with a deliberately bounded scope:

- **No instance bootstrapping.** Deploying Pocket-ID and enrolling the first admin
  passkey is not managed by this provider; do that before pointing it at an instance.
- **Not covered by the API surface used:** user profile pictures, the device login flow, audit logs, and
  WebAuthn credentials and passkey enrollment (passkeys are enrolled by users, not by IaC).
  One-time access tokens and e-mails, the LDAP sync trigger and the SMTP test e-mail are
  actions, not state, and are not modelled either.
- **`ApplicationConfiguration` is a singleton and is never deleted.** Only the fields you
  set are managed; removing a field from the program leaves its server value as it is, and
  destroying the resource leaves the configuration untouched. Use at most one per instance.
- **Secrets are not read back.** The API returns secrets (client secret, signup token, SMTP and LDAP passwords) only at creation or not at all, so they are kept
  from state and are empty after an import.
- **Association resources are authoritative.** `UserGroupMembers`, `UserCustomClaims` and
  `UserGroupCustomClaims` set the complete list, and deleting them empties it; only one
  resource should own each relation.
- **No API key resource.** Pocket-ID refuses to manage API keys (`/api/api-keys`) when the request
  itself is authenticated with an API key (403 `api_key_auth_not_allowed`), which is the only way this
  provider authenticates, so such a resource could never work.
- **APIs, SCIM and signup tokens depend on the Pocket-ID version.** They exist only on
  versions that serve those endpoints (see [Compatibility](#compatibility)).
- **Only YAML and Go example programs are exercised in CI/E2E.** The nodejs, python and
  dotnet SDKs are generated and published (see [Installing](#installing)) but are not
  covered by this repo's example programs or lifecycle tests.
- **No Java/Maven SDK.** Pulumi's Java support has too little adoption to justify the
  extra setup cost (Sonatype namespace verification, GPG-signed releases).

## Reference

This provider isn't (yet) listed on the [Pulumi Registry](https://www.pulumi.com/registry/),
so there is no `pulumi.com/registry/packages/pocket-id` page. The
[Resource reference](#resource-reference) above, the **Go SDK reference** on
[pkg.go.dev](https://pkg.go.dev/github.com/axnic/pulumi-pocket-id/sdk/go/pulumi-pocket-id)
and the generated schema (`provider/cmd/pulumi-resource-pocket-id/schema.json`) carry the
same descriptions.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for building the provider, the test layout
(including how to run the E2E suite locally), generating the SDKs, and the devcontainer
setup. See also our [Code of Conduct](CODE-OF-CONDUCT.md).

## License

[Apache-2.0](LICENSE)

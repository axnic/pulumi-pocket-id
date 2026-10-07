// Copyright 2025, axnic.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"errors"
	"slices"

	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi-go-provider/infer/types"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"

	"github.com/axnic/pulumi-pocket-id/provider/internal/pocketidclient"
)

func init() { registerResource(infer.Resource(&OidcClient{})) }

// OidcClient manages a Pocket-ID OIDC client. It never creates a client secret
// (see OidcClientSecret) and owns the set of user groups allowed on the client
// through allowedUserGroupIds.
type OidcClient struct{}

// OidcFederatedIdentity is a federated identity accepted as a credential of an OIDC client.
type OidcFederatedIdentity struct {
	Issuer           string  `pulumi:"issuer"`
	Subject          *string `pulumi:"subject,optional"`
	Audience         *string `pulumi:"audience,optional"`
	Jwks             *string `pulumi:"jwks,optional"`
	ReplayProtection *bool   `pulumi:"replayProtection,optional"`
}

// OidcClientCredentials holds the credentials of an OIDC client managed by Pulumi.
type OidcClientCredentials struct {
	FederatedIdentities []OidcFederatedIdentity `pulumi:"federatedIdentities,optional"`
}

// OidcClientArgs are the inputs of the OidcClient resource.
type OidcClientArgs struct {
	ClientID                            *string                `pulumi:"clientId,optional" provider:"replaceOnChanges"`
	Name                                string                 `pulumi:"name"`
	Description                         *string                `pulumi:"description,optional"`
	CallbackUrls                        []string               `pulumi:"callbackUrls,optional"`
	LogoutCallbackUrls                  []string               `pulumi:"logoutCallbackUrls,optional"`
	LaunchURL                           *string                `pulumi:"launchUrl,optional"`
	Logo                                *types.AssetOrArchive  `pulumi:"logo,optional"`
	DarkLogo                            *types.AssetOrArchive  `pulumi:"darkLogo,optional"`
	IsPublic                            bool                   `pulumi:"isPublic,optional"`
	PkceEnabled                         bool                   `pulumi:"pkceEnabled,optional"`
	RequiresReauthentication            bool                   `pulumi:"requiresReauthentication,optional"`
	RequiresPushedAuthorizationRequests bool                   `pulumi:"requiresPushedAuthorizationRequests,optional"`
	SkipConsent                         bool                   `pulumi:"skipConsent,optional"`
	AccessTokenDurationMinutes          *int                   `pulumi:"accessTokenDurationMinutes,optional"`
	RefreshTokenDurationMinutes         *int                   `pulumi:"refreshTokenDurationMinutes,optional"`
	BackchannelLogoutURL                *string                `pulumi:"backchannelLogoutUrl,optional"`
	Credentials                         *OidcClientCredentials `pulumi:"credentials,optional"`
	AllowedUserGroupIDs                 []string               `pulumi:"allowedUserGroupIds,optional"`
}

// OidcClientState is persisted in the Pulumi state.
type OidcClientState struct {
	OidcClientArgs
	ClientType        string  `pulumi:"clientType"`
	PkceSupported     bool    `pulumi:"pkceSupported"`
	HasLogo           bool    `pulumi:"hasLogo"`
	HasDarkLogo       bool    `pulumi:"hasDarkLogo"`
	IsGroupRestricted bool    `pulumi:"isGroupRestricted"`
	LogoSHA256        *string `pulumi:"logoSha256,optional"`
	DarkLogoSHA256    *string `pulumi:"darkLogoSha256,optional"`
}

var _ infer.Annotated = (*OidcClient)(nil)

// Annotate describes the resource.
func (*OidcClient) Annotate(a infer.Annotator) {
	a.Describe(&OidcClient{}, "An OIDC client of Pocket-ID. The client never creates a secret by itself; "+
		"use OidcClientSecret to generate one. The allowed user groups are owned by this resource. "+
		"The logos are uploaded from files (logo, darkLogo).")
}

var _ infer.Annotated = (*OidcFederatedIdentity)(nil)

// Annotate describes the fields of a federated identity.
func (f *OidcFederatedIdentity) Annotate(a infer.Annotator) {
	a.Describe(&f.Issuer, "The issuer URL of the federated identity.")
	a.Describe(&f.Subject, "The expected subject of the federated identity.")
	a.Describe(&f.Audience, "The expected audience of the federated identity.")
	a.Describe(&f.Jwks, "The URL or content of the JWKS used to verify the federated identity.")
	a.Describe(&f.ReplayProtection, "Whether tokens issued by the federated identity can only be used once.")
}

var _ infer.Annotated = (*OidcClientCredentials)(nil)

// Annotate describes the fields of the credentials.
func (c *OidcClientCredentials) Annotate(a infer.Annotator) {
	a.Describe(&c.FederatedIdentities, "The federated identities allowed to authenticate as the client.")
}

var _ infer.Annotated = (*OidcClientArgs)(nil)

// Annotate describes the inputs.
func (a *OidcClientArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ClientID, "The ID of the client. Generated by Pocket-ID when unset. Changing it replaces the client.")
	an.Describe(&a.Name, "The display name of the client (50 characters max).")
	an.Describe(&a.Description, "A description of the client (150 characters max).")
	an.Describe(&a.CallbackUrls, "The allowed redirect URLs.")
	an.Describe(&a.LogoutCallbackUrls, "The allowed post-logout redirect URLs.")
	an.Describe(&a.LaunchURL, "The URL used to launch the application from the Pocket-ID dashboard.")
	an.Describe(&a.Logo, "The client logo (PNG, JPG or SVG) uploaded from a file, as an asset, e.g. a FileAsset. "+
		"Archives are not supported. A change of content re-uploads the file and removing it deletes the logo. "+
		"An asset cannot be read back: after an import it is empty and must be "+
		"set in the program, which uploads the logo again if it differs from the served one.")
	an.Describe(&a.DarkLogo, "The dark mode client logo (PNG, JPG or SVG) uploaded from a file, as an asset, "+
		"e.g. a FileAsset. Archives are not supported. A change of content re-uploads the file and removing it "+
		"deletes the logo. An asset cannot be read back: after an import "+
		"it is empty and must be set in the program, which uploads the logo again if it differs from the "+
		"served one.")
	an.Describe(&a.IsPublic, "Whether the client is public (no client secret, PKCE recommended).")
	an.Describe(&a.PkceEnabled, "Whether PKCE is required for the client.")
	an.Describe(&a.RequiresReauthentication, "Whether users must re-authenticate on every authorization.")
	an.Describe(&a.RequiresPushedAuthorizationRequests, "Whether the client must use pushed authorization requests (PAR).")
	an.Describe(&a.SkipConsent, "Whether the consent screen is skipped for this client.")
	an.Describe(&a.AccessTokenDurationMinutes, "The access token lifetime in minutes. Pocket-ID's default applies when "+
		"unset.")
	an.Describe(&a.RefreshTokenDurationMinutes, "The refresh token lifetime in minutes. Pocket-ID's default applies "+
		"when unset.")
	an.Describe(&a.BackchannelLogoutURL, "The URL called for OIDC back-channel logout. "+
		"Requires Pocket-ID v2.17.0 or later: older versions ignore the field, "+
		"so the provider fails instead of silently dropping it.")
	an.Describe(&a.Credentials, "The credentials of the client managed here (federated identities).")
	an.Describe(&a.AllowedUserGroupIDs, "The IDs of the user groups allowed to use the client. Authoritative: "+
		"the set is replaced on every update, and the client is restricted to these groups when the list is "+
		"non-empty (unrestricted when empty).")
}

var _ infer.Annotated = (*OidcClientState)(nil)

// Annotate describes the outputs.
func (s *OidcClientState) Annotate(a infer.Annotator) {
	a.Describe(&s.ClientType, "The client type reported by Pocket-ID (confidential or public).")
	a.Describe(&s.PkceSupported, "Whether PKCE is supported for this client.")
	a.Describe(&s.HasLogo, "Whether the client has a logo.")
	a.Describe(&s.HasDarkLogo, "Whether the client has a dark mode logo.")
	a.Describe(&s.IsGroupRestricted, "Whether access is restricted to the allowed user groups.")
	a.Describe(&s.LogoSHA256, "The hex SHA-256 digest of the uploaded logo (see logo). At refresh it is the digest "+
		"of the bytes currently served by Pocket-ID, so out-of-band changes show up as a diff.")
	a.Describe(&s.DarkLogoSHA256, "The hex SHA-256 digest of the uploaded dark mode logo (see darkLogo). At "+
		"refresh it is the digest of the bytes currently served by Pocket-ID, so out-of-band changes show up as "+
		"a diff.")
}

// hasImage reports whether an optional image input carries an asset.
func hasImage(img *types.AssetOrArchive) bool { return img != nil && !isEmptyImage(*img) }

// logoOp is what a create or an update must do to one logo variant.
type logoOp struct {
	light  bool
	upload *imageUpload // set: upload these bytes
	delete bool         // true: remove the logo
	sha    *string      // the digest to record in the state
}

// planLogo decides what to do with one logo variant. The upload is skipped when the digest recorded in the
// state (the last known content of the logo, refreshed at every read) is the one of the new asset. A logo
// that was managed (uploaded) and is no longer wanted is deleted: Pocket-ID keeps it
// when an update omits it.
func planLogo(
	light bool, now *types.AssetOrArchive, wasSHA *string, wasManaged bool,
) (logoOp, error) {
	op := logoOp{light: light}
	switch {
	case hasImage(now):
		up, err := readImage(*now)
		if err != nil {
			return op, err
		}
		op.sha = &up.sha256
		if wasSHA == nil || *wasSHA != up.sha256 {
			op.upload = &up
		}
	case wasManaged:
		op.delete = true
	}
	return op, nil
}

func planLogos(in OidcClientArgs, prev *OidcClientState) ([2]logoOp, error) {
	var was OidcClientState
	if prev != nil {
		was = *prev
	}
	var ops [2]logoOp
	var err error
	if ops[0], err = planLogo(true, in.Logo, was.LogoSHA256,
		hasImage(was.Logo) || was.LogoSHA256 != nil); err != nil {
		return ops, errors.Join(errors.New("invalid logo"), err)
	}
	if ops[1], err = planLogo(false, in.DarkLogo, was.DarkLogoSHA256,
		hasImage(was.DarkLogo) || was.DarkLogoSHA256 != nil); err != nil {
		return ops, errors.Join(errors.New("invalid darkLogo"), err)
	}
	return ops, nil
}

func applyLogos(ctx context.Context, id string, ops [2]logoOp) error {
	api := clientFromContext(ctx)
	for _, op := range ops {
		switch {
		case op.upload != nil:
			if err := api.SetOidcClientLogo(ctx, id, op.light, op.upload.filename, op.upload.data); err != nil {
				return err
			}
		case op.delete:
			if err := api.DeleteOidcClientLogo(ctx, id, op.light); err != nil && !isGone(err) {
				return err
			}
		}
	}
	return nil
}

// previewLogoSHA is the digest of a logo asset when it is already known (nil otherwise).
func previewLogoSHA(img *types.AssetOrArchive) *string {
	if !hasImage(img) {
		return nil
	}
	return oidcNonEmpty(imageSHA256(*img))
}

// Create creates the client, removes any secret Pocket-ID generated for it and applies the allowed groups.
func (*OidcClient) Create(
	ctx context.Context, req infer.CreateRequest[OidcClientArgs],
) (infer.CreateResponse[OidcClientState], error) {
	if req.DryRun {
		id := req.Name
		if req.Inputs.ClientID != nil {
			id = *req.Inputs.ClientID
		}
		return infer.CreateResponse[OidcClientState]{ID: id, Output: OidcClientState{
			OidcClientArgs: req.Inputs,
			LogoSHA256:     previewLogoSHA(req.Inputs.Logo), DarkLogoSHA256: previewLogoSHA(req.Inputs.DarkLogo),
		}}, nil
	}
	// Everything that can be checked locally is checked before the client exists.
	logos, err := planLogos(req.Inputs, nil)
	if err != nil {
		return infer.CreateResponse[OidcClientState]{}, err
	}
	api := clientFromContext(ctx)
	created, err := api.CreateOidcClient(ctx, oidcClientRequest(req.Inputs))
	if err != nil {
		return infer.CreateResponse[OidcClientState]{}, err
	}
	// The client resource never owns a secret: drop the one Pocket-ID may have generated.
	if created.CreatedSecret != nil {
		if err := api.DeleteOidcClientSecret(ctx, created.ID, created.CreatedSecret.ID); err != nil && !isGone(err) {
			return infer.CreateResponse[OidcClientState]{}, err
		}
	}
	if len(req.Inputs.AllowedUserGroupIDs) > 0 {
		if err := api.SetOidcClientAllowedUserGroups(ctx, created.ID, req.Inputs.AllowedUserGroupIDs); err != nil {
			return infer.CreateResponse[OidcClientState]{}, err
		}
	}
	if err := applyLogos(ctx, created.ID, logos); err != nil {
		return infer.CreateResponse[OidcClientState]{}, err
	}
	got, err := api.GetOidcClient(ctx, created.ID)
	if err != nil {
		return infer.CreateResponse[OidcClientState]{}, err
	}
	if err := checkBackchannelSupported(req.Inputs, got); err != nil {
		// Do not leave a half-configured client behind.
		_ = api.DeleteOidcClient(ctx, created.ID)
		return infer.CreateResponse[OidcClientState]{}, err
	}
	state := oidcClientState(got, req.Inputs, false)
	state.LogoSHA256, state.DarkLogoSHA256 = logos[0].sha, logos[1].sha
	return infer.CreateResponse[OidcClientState]{ID: got.ID, Output: state}, nil
}

// Read refreshes the client from the API. An empty ID means the client is gone.
func (*OidcClient) Read(
	ctx context.Context, req infer.ReadRequest[OidcClientArgs, OidcClientState],
) (infer.ReadResponse[OidcClientArgs, OidcClientState], error) {
	got, err := clientFromContext(ctx).GetOidcClient(ctx, req.ID)
	if isGone(err) {
		return infer.ReadResponse[OidcClientArgs, OidcClientState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[OidcClientArgs, OidcClientState]{}, err
	}
	// An empty state means an import: there is no previous input to preserve.
	state := oidcClientState(got, req.State.OidcClientArgs, req.State.Name == "")
	var errLogo, errDark error
	state.Logo, state.LogoSHA256, errLogo = refreshLogo(
		ctx, req.ID, true, got.HasLogo, req.State.Logo, req.State.LogoSHA256)
	state.DarkLogo, state.DarkLogoSHA256, errDark = refreshLogo(
		ctx, req.ID, false, got.HasDarkLogo, req.State.DarkLogo, req.State.DarkLogoSHA256)
	if err := errors.Join(errLogo, errDark); err != nil {
		return infer.ReadResponse[OidcClientArgs, OidcClientState]{}, err
	}
	return infer.ReadResponse[OidcClientArgs, OidcClientState]{ID: got.ID, Inputs: state.OidcClientArgs, State: state}, nil
}

// refreshLogo re-reads an uploaded logo to detect drift. A logo that is not managed as a file (no asset and no
// digest in the state: imported) is left alone, as an asset cannot be read back. When the
// served bytes differ from the recorded ones, the asset is replaced by one reduced to the served digest so
// that the next diff schedules a new upload; a logo that is gone (404, or no logo reported) is dropped.
func refreshLogo(
	ctx context.Context, id string, light, present bool, img *types.AssetOrArchive, sha *string,
) (*types.AssetOrArchive, *string, error) {
	if !hasImage(img) && sha == nil {
		return nil, nil, nil
	}
	if !present {
		return nil, nil, nil
	}
	data, err := clientFromContext(ctx).GetOidcClientLogo(ctx, id, light)
	if isGone(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	served := sha256Hex(data)
	if hasImage(img) && imageSHA256(*img) == served {
		return img, &served, nil
	}
	return &types.AssetOrArchive{Asset: &resource.Asset{Sig: resource.AssetSig, Hash: served}}, &served, nil
}

// Update replaces the client settings, the allowed groups and the logos.
func (*OidcClient) Update(
	ctx context.Context, req infer.UpdateRequest[OidcClientArgs, OidcClientState],
) (infer.UpdateResponse[OidcClientState], error) {
	if req.DryRun {
		out := req.State
		out.OidcClientArgs = req.Inputs
		out.LogoSHA256, out.DarkLogoSHA256 = previewLogoSHA(req.Inputs.Logo), previewLogoSHA(req.Inputs.DarkLogo)
		return infer.UpdateResponse[OidcClientState]{Output: out}, nil
	}
	logos, err := planLogos(req.Inputs, &req.State)
	if err != nil {
		return infer.UpdateResponse[OidcClientState]{}, err
	}
	api := clientFromContext(ctx)
	if _, err := api.UpdateOidcClient(ctx, req.ID, oidcClientRequest(req.Inputs)); err != nil {
		return infer.UpdateResponse[OidcClientState]{}, err
	}
	if err := api.SetOidcClientAllowedUserGroups(ctx, req.ID, req.Inputs.AllowedUserGroupIDs); err != nil {
		return infer.UpdateResponse[OidcClientState]{}, err
	}
	if err := applyLogos(ctx, req.ID, logos); err != nil {
		return infer.UpdateResponse[OidcClientState]{}, err
	}
	got, err := api.GetOidcClient(ctx, req.ID)
	if err != nil {
		return infer.UpdateResponse[OidcClientState]{}, err
	}
	if err := checkBackchannelSupported(req.Inputs, got); err != nil {
		return infer.UpdateResponse[OidcClientState]{}, err
	}
	state := oidcClientState(got, req.Inputs, false)
	state.LogoSHA256, state.DarkLogoSHA256 = logos[0].sha, logos[1].sha
	return infer.UpdateResponse[OidcClientState]{Output: state}, nil
}

// Delete deletes the client.
func (*OidcClient) Delete(ctx context.Context, req infer.DeleteRequest[OidcClientState]) (infer.DeleteResponse, error) {
	if err := clientFromContext(ctx).DeleteOidcClient(ctx, req.ID); err != nil && !isGone(err) {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

func oidcClientRequest(a OidcClientArgs) pocketidclient.OidcClientRequest {
	r := pocketidclient.OidcClientRequest{
		ID:                                  oidcDeref(a.ClientID),
		Name:                                a.Name,
		Description:                         oidcDeref(a.Description),
		CallbackURLs:                        a.CallbackUrls,
		LogoutCallbackURLs:                  a.LogoutCallbackUrls,
		LaunchURL:                           oidcDeref(a.LaunchURL),
		IsPublic:                            a.IsPublic,
		PkceEnabled:                         a.PkceEnabled,
		RequiresReauthentication:            a.RequiresReauthentication,
		RequiresPushedAuthorizationRequests: a.RequiresPushedAuthorizationRequests,
		SkipConsent:                         a.SkipConsent,
		IsGroupRestricted:                   len(a.AllowedUserGroupIDs) > 0,
		AccessTokenDurationMinutes:          a.AccessTokenDurationMinutes,
		RefreshTokenDurationMinutes:         a.RefreshTokenDurationMinutes,
		BackchannelLogoutURL:                oidcDeref(a.BackchannelLogoutURL),
	}
	if a.Credentials != nil {
		for _, f := range a.Credentials.FederatedIdentities {
			r.Credentials.FederatedIdentities = append(r.Credentials.FederatedIdentities, pocketidclient.FederatedIdentity{
				Issuer: f.Issuer, Subject: oidcDeref(f.Subject), Audience: oidcDeref(f.Audience),
				JWKS: oidcDeref(f.Jwks), ReplayProtection: oidcDeref(f.ReplayProtection),
			})
		}
	}
	return r
}

// oidcClientState maps an API client to the resource state. prev holds the
// inputs known so far: values the API does not return (the optional
// client ID) are carried over from it, and server defaults are not turned into
// inputs the user never set. imported is true when there is no previous input.
func oidcClientState(c *pocketidclient.OidcClient, prev OidcClientArgs, imported bool) OidcClientState {
	a := OidcClientArgs{
		ClientID:                            prev.ClientID,
		Name:                                c.Name,
		Description:                         oidcNonEmpty(c.Description),
		CallbackUrls:                        oidcNonEmptySlice(c.CallbackURLs),
		LogoutCallbackUrls:                  oidcNonEmptySlice(c.LogoutCallbackURLs),
		LaunchURL:                           oidcNonEmpty(c.LaunchURL),
		Logo:                                prev.Logo,
		DarkLogo:                            prev.DarkLogo,
		IsPublic:                            c.IsPublic,
		PkceEnabled:                         c.PkceEnabled,
		RequiresReauthentication:            c.RequiresReauthentication,
		RequiresPushedAuthorizationRequests: c.RequiresPushedAuthorizationRequests,
		SkipConsent:                         c.SkipConsent,
		BackchannelLogoutURL:                oidcNonEmpty(oidcDeref(c.BackchannelLogoutURL)),
		AllowedUserGroupIDs:                 oidcNonEmptySlice(c.AllowedUserGroupIDs()),
	}
	if imported {
		a.ClientID = &c.ID
	}
	if imported || prev.AccessTokenDurationMinutes != nil {
		a.AccessTokenDurationMinutes = oidcNonZero(c.AccessTokenDurationMinutes)
	}
	if imported || prev.RefreshTokenDurationMinutes != nil {
		a.RefreshTokenDurationMinutes = oidcNonZero(c.RefreshTokenDurationMinutes)
	}
	// Keep the previous order when the set of groups is unchanged.
	if len(a.AllowedUserGroupIDs) == len(prev.AllowedUserGroupIDs) {
		sorted := func(s []string) []string { return slices.Sorted(slices.Values(s)) }
		if slices.Equal(sorted(a.AllowedUserGroupIDs), sorted(prev.AllowedUserGroupIDs)) {
			a.AllowedUserGroupIDs = prev.AllowedUserGroupIDs
		}
	}
	if fis := c.Credentials.FederatedIdentities; len(fis) > 0 {
		creds := &OidcClientCredentials{}
		for _, f := range fis {
			creds.FederatedIdentities = append(creds.FederatedIdentities, OidcFederatedIdentity{
				Issuer: f.Issuer, Subject: oidcNonEmpty(f.Subject), Audience: oidcNonEmpty(f.Audience),
				Jwks: oidcNonEmpty(f.JWKS), ReplayProtection: oidcNonFalse(f.ReplayProtection),
			})
		}
		a.Credentials = creds
	}
	return OidcClientState{
		OidcClientArgs:    a,
		ClientType:        c.ClientType,
		PkceSupported:     c.PkceSupported,
		HasLogo:           c.HasLogo,
		HasDarkLogo:       c.HasDarkLogo,
		IsGroupRestricted: c.IsGroupRestricted,
	}
}

func oidcDeref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

// checkBackchannelSupported fails when backchannelLogoutUrl is set but the
// server does not know the field (Pocket-ID < v2.17.0 silently ignores it,
// which would otherwise show up as a permanent drift).
func checkBackchannelSupported(in OidcClientArgs, got *pocketidclient.OidcClient) error {
	if oidcDeref(in.BackchannelLogoutURL) != "" && got.BackchannelLogoutURL == nil {
		return errors.New("backchannelLogoutUrl is not supported by this Pocket-ID version (requires v2.17.0 or later)")
	}
	return nil
}

func oidcNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func oidcNonZero(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}

func oidcNonFalse(b bool) *bool {
	if !b {
		return nil
	}
	return &b
}

func oidcNonEmptySlice(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

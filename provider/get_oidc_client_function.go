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

	"github.com/pulumi/pulumi-go-provider/infer"
)

func init() { registerFunction(infer.Function(&GetOidcClient{})) }

// GetOidcClient looks up an OIDC client by ID.
type GetOidcClient struct{}

// GetOidcClientArgs are the inputs of the getOidcClient function.
type GetOidcClientArgs struct {
	ID string `pulumi:"clientId"`
}

// GetOidcClientResult is the client returned by getOidcClient. Secrets are never included.
type GetOidcClientResult struct {
	ID                                  string                  `pulumi:"clientId"`
	Name                                string                  `pulumi:"name"`
	Description                         string                  `pulumi:"description"`
	ClientType                          string                  `pulumi:"clientType"`
	CallbackUrls                        []string                `pulumi:"callbackUrls"`
	LogoutCallbackUrls                  []string                `pulumi:"logoutCallbackUrls"`
	LaunchURL                           string                  `pulumi:"launchUrl"`
	HasLogo                             bool                    `pulumi:"hasLogo"`
	HasDarkLogo                         bool                    `pulumi:"hasDarkLogo"`
	IsPublic                            bool                    `pulumi:"isPublic"`
	PkceEnabled                         bool                    `pulumi:"pkceEnabled"`
	PkceSupported                       bool                    `pulumi:"pkceSupported"`
	RequiresReauthentication            bool                    `pulumi:"requiresReauthentication"`
	RequiresPushedAuthorizationRequests bool                    `pulumi:"requiresPushedAuthorizationRequests"`
	SkipConsent                         bool                    `pulumi:"skipConsent"`
	IsGroupRestricted                   bool                    `pulumi:"isGroupRestricted"`
	AccessTokenDurationMinutes          int                     `pulumi:"accessTokenDurationMinutes"`
	RefreshTokenDurationMinutes         int                     `pulumi:"refreshTokenDurationMinutes"`
	BackchannelLogoutURL                string                  `pulumi:"backchannelLogoutUrl"`
	FederatedIdentities                 []OidcFederatedIdentity `pulumi:"federatedIdentities"`
	AllowedUserGroupIDs                 []string                `pulumi:"allowedUserGroupIds"`
}

var _ infer.Annotated = (*GetOidcClientArgs)(nil)

// Annotate describes the inputs.
func (a *GetOidcClientArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ID, "The ID of the OIDC client to look up.")
}

var _ infer.Annotated = (*GetOidcClientResult)(nil)

// Annotate describes the outputs.
func (r *GetOidcClientResult) Annotate(a infer.Annotator) {
	a.Describe(&r.ID, "The ID of the client.")
	a.Describe(&r.Name, "The display name of the client.")
	a.Describe(&r.Description, "The description of the client.")
	a.Describe(&r.ClientType, "The client type reported by Pocket-ID.")
	a.Describe(&r.CallbackUrls, "The allowed redirect URLs.")
	a.Describe(&r.LogoutCallbackUrls, "The allowed post-logout redirect URLs.")
	a.Describe(&r.LaunchURL, "The launch URL of the application.")
	a.Describe(&r.HasLogo, "Whether the client has a logo.")
	a.Describe(&r.HasDarkLogo, "Whether the client has a dark mode logo.")
	a.Describe(&r.IsPublic, "Whether the client is public.")
	a.Describe(&r.PkceEnabled, "Whether PKCE is required.")
	a.Describe(&r.PkceSupported, "Whether PKCE is supported.")
	a.Describe(&r.RequiresReauthentication, "Whether users must re-authenticate on every authorization.")
	a.Describe(&r.RequiresPushedAuthorizationRequests, "Whether pushed authorization requests (PAR) are required.")
	a.Describe(&r.SkipConsent, "Whether the consent screen is skipped.")
	a.Describe(&r.IsGroupRestricted, "Whether access is restricted to the allowed user groups.")
	a.Describe(&r.AccessTokenDurationMinutes, "The access token lifetime in minutes.")
	a.Describe(&r.RefreshTokenDurationMinutes, "The refresh token lifetime in minutes.")
	a.Describe(&r.BackchannelLogoutURL, "The back-channel logout URL.")
	a.Describe(&r.FederatedIdentities, "The federated identities allowed to authenticate as the client.")
	a.Describe(&r.AllowedUserGroupIDs, "The IDs of the user groups allowed to use the client.")
}

// Invoke fetches the client.
func (*GetOidcClient) Invoke(
	ctx context.Context, req infer.FunctionRequest[GetOidcClientArgs],
) (infer.FunctionResponse[GetOidcClientResult], error) {
	c, err := clientFromContext(ctx).GetOidcClient(ctx, req.Input.ID)
	if err != nil {
		return infer.FunctionResponse[GetOidcClientResult]{}, err
	}
	s := oidcClientState(c, OidcClientArgs{}, true)
	r := GetOidcClientResult{
		ID: c.ID, Name: c.Name, Description: c.Description, ClientType: c.ClientType,
		CallbackUrls: oidcNilToEmpty(c.CallbackURLs), LogoutCallbackUrls: oidcNilToEmpty(c.LogoutCallbackURLs),
		LaunchURL: c.LaunchURL, HasLogo: c.HasLogo, HasDarkLogo: c.HasDarkLogo, IsPublic: c.IsPublic,
		PkceEnabled: c.PkceEnabled, PkceSupported: c.PkceSupported,
		RequiresReauthentication:            c.RequiresReauthentication,
		RequiresPushedAuthorizationRequests: c.RequiresPushedAuthorizationRequests,
		SkipConsent:                         c.SkipConsent, IsGroupRestricted: c.IsGroupRestricted,
		AccessTokenDurationMinutes: c.AccessTokenDurationMinutes, RefreshTokenDurationMinutes: c.RefreshTokenDurationMinutes,
		BackchannelLogoutURL: c.BackchannelLogoutURL, AllowedUserGroupIDs: oidcNilToEmpty(c.AllowedUserGroupIDs()),
		FederatedIdentities: []OidcFederatedIdentity{},
	}
	if s.Credentials != nil {
		r.FederatedIdentities = s.Credentials.FederatedIdentities
	}
	return infer.FunctionResponse[GetOidcClientResult]{Output: r}, nil
}

func oidcNilToEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

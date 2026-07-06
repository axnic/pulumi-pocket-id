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

// OidcClient manages a Pocket-ID OIDC client.
type OidcClient struct{}

// FederatedIdentity configures a federated identity provider for an OIDC client.
type FederatedIdentity struct {
	Issuer           string `pulumi:"issuer" json:"issuer"`
	Subject          string `pulumi:"subject,optional" json:"subject,omitempty"`
	Audience         string `pulumi:"audience,optional" json:"audience,omitempty"`
	Jwks             string `pulumi:"jwks,optional" json:"jwks,omitempty"`
	ReplayProtection bool   `pulumi:"replayProtection,optional" json:"replayProtection,omitempty"`
}

// OidcClientCredentials holds the credentials of an OIDC client.
type OidcClientCredentials struct {
	FederatedIdentities []FederatedIdentity `pulumi:"federatedIdentities,optional" json:"federatedIdentities,omitempty"`
}

// OidcClientArgs are the inputs for an OIDC client.
type OidcClientArgs struct {
	// ClientId optionally sets the OIDC client ID. If omitted, Pocket-ID generates one.
	ClientId string `pulumi:"clientId,optional" json:"id,omitempty"`
	Name     string `pulumi:"name" json:"name"`

	CallbackUrls                       []string               `pulumi:"callbackUrls,optional" json:"callbackURLs,omitempty"`
	LogoutCallbackUrls                 []string               `pulumi:"logoutCallbackUrls,optional" json:"logoutCallbackURLs,omitempty"`
	IsPublic                           bool                   `pulumi:"isPublic,optional" json:"isPublic,omitempty"`
	PkceEnabled                        bool                   `pulumi:"pkceEnabled,optional" json:"pkceEnabled,omitempty"`
	RequiresReauthentication           bool                   `pulumi:"requiresReauthentication,optional" json:"requiresReauthentication,omitempty"`
	RequiresPushedAuthorizationRequests bool                   `pulumi:"requiresPushedAuthorizationRequests,optional" json:"requiresPushedAuthorizationRequests,omitempty"`
	SkipConsent                        bool                   `pulumi:"skipConsent,optional" json:"skipConsent,omitempty"`
	LaunchUrl                          *string                `pulumi:"launchUrl,optional" json:"launchURL,omitempty"`
	IsGroupRestricted                  bool                   `pulumi:"isGroupRestricted,optional" json:"isGroupRestricted,omitempty"`
	Credentials                        *OidcClientCredentials `pulumi:"credentials,optional" json:"credentials,omitempty"`
}

// OidcClientState is persisted in Pulumi state.
type OidcClientState struct {
	OidcClientArgs
	HasLogo       bool `pulumi:"hasLogo,optional" json:"hasLogo,omitempty"`
	HasDarkLogo   bool `pulumi:"hasDarkLogo,optional" json:"hasDarkLogo,omitempty"`
	PkceSupported bool `pulumi:"pkceSupported,optional" json:"pkceSupported,omitempty"`
}

// oidcClientResponse mirrors the OidcClientDto response (with embedded metadata).
type oidcClientResponse struct {
	Id                                 string                 `json:"id"`
	Name                               string                 `json:"name"`
	HasLogo                            bool                   `json:"hasLogo"`
	HasDarkLogo                        bool                   `json:"hasDarkLogo"`
	LaunchURL                          *string                `json:"launchURL"`
	RequiresReauthentication           bool                   `json:"requiresReauthentication"`
	CallbackURLs                       []string               `json:"callbackURLs"`
	LogoutCallbackURLs                 []string               `json:"logoutCallbackURLs"`
	IsPublic                           bool                   `json:"isPublic"`
	PkceEnabled                        bool                   `json:"pkceEnabled"`
	RequiresPushedAuthorizationRequests bool                   `json:"requiresPushedAuthorizationRequests"`
	SkipConsent                        bool                   `json:"skipConsent"`
	Credentials                        *OidcClientCredentials `json:"credentials"`
	IsGroupRestricted                  bool                   `json:"isGroupRestricted"`
	PkceSupported                      bool                   `json:"pkceSupported,omitempty"`
}

func (resp oidcClientResponse) toState() OidcClientState {
	return OidcClientState{
		OidcClientArgs: OidcClientArgs{
			Name:                               resp.Name,
			CallbackUrls:                       resp.CallbackURLs,
			LogoutCallbackUrls:                 resp.LogoutCallbackURLs,
			IsPublic:                           resp.IsPublic,
			PkceEnabled:                        resp.PkceEnabled,
			RequiresReauthentication:           resp.RequiresReauthentication,
			RequiresPushedAuthorizationRequests: resp.RequiresPushedAuthorizationRequests,
			SkipConsent:                        resp.SkipConsent,
			LaunchUrl:                          resp.LaunchURL,
			IsGroupRestricted:                  resp.IsGroupRestricted,
			Credentials:                        resp.Credentials,
		},
		HasLogo:       resp.HasLogo,
		HasDarkLogo:   resp.HasDarkLogo,
		PkceSupported: resp.PkceSupported,
	}
}

// Create creates a new OIDC client.
func (*OidcClient) Create(
	ctx context.Context,
	req infer.CreateRequest[OidcClientArgs],
) (infer.CreateResponse[OidcClientState], error) {
	if req.DryRun {
		id := req.Inputs.ClientId
		if id == "" {
			id = req.Name
		}
		return infer.CreateResponse[OidcClientState]{
			ID:     id,
			Output: OidcClientState{OidcClientArgs: req.Inputs},
		}, nil
	}

	client := clientFromContext(ctx)
	var resp oidcClientResponse
	if err := client.do(ctx, "POST", "/api/oidc/clients", req.Inputs, &resp); err != nil {
		return infer.CreateResponse[OidcClientState]{}, err
	}
	state := resp.toState()
	return infer.CreateResponse[OidcClientState]{ID: resp.Id, Output: state}, nil
}

// Read fetches the current state of an OIDC client.
func (*OidcClient) Read(
	ctx context.Context,
	req infer.ReadRequest[OidcClientArgs, OidcClientState],
) (infer.ReadResponse[OidcClientArgs, OidcClientState], error) {
	client := clientFromContext(ctx)
	var resp oidcClientResponse
	if err := client.do(ctx, "GET", "/api/oidc/clients/"+req.ID, nil, &resp); err != nil {
		return infer.ReadResponse[OidcClientArgs, OidcClientState]{}, err
	}
	state := resp.toState()
	return infer.ReadResponse[OidcClientArgs, OidcClientState]{ID: resp.Id, Inputs: state.OidcClientArgs, State: state}, nil
}

// Update modifies an existing OIDC client.
func (*OidcClient) Update(
	ctx context.Context,
	req infer.UpdateRequest[OidcClientArgs, OidcClientState],
) (infer.UpdateResponse[OidcClientState], error) {
	client := clientFromContext(ctx)
	var resp oidcClientResponse
	if err := client.do(ctx, "PUT", "/api/oidc/clients/"+req.ID, req.Inputs, &resp); err != nil {
		return infer.UpdateResponse[OidcClientState]{}, err
	}
	return infer.UpdateResponse[OidcClientState]{Output: resp.toState()}, nil
}

// Delete removes an OIDC client.
func (*OidcClient) Delete(
	ctx context.Context,
	req infer.DeleteRequest[OidcClientState],
) (infer.DeleteResponse, error) {
	client := clientFromContext(ctx)
	if err := client.do(ctx, "DELETE", "/api/oidc/clients/"+req.ID, nil, nil); err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

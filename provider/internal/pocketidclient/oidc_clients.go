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

package pocketidclient

import (
	"context"
	"net/http"
	"net/url"
)

// FederatedIdentity is a federated identity accepted as a client credential.
type FederatedIdentity struct {
	Issuer           string `json:"issuer"`
	Subject          string `json:"subject,omitempty"`
	Audience         string `json:"audience,omitempty"`
	JWKS             string `json:"jwks,omitempty"`
	ReplayProtection bool   `json:"replayProtection,omitempty"`
}

// OidcClientCredentials holds the credentials of an OIDC client. Only the
// federated identities are writable; secrets are managed separately.
type OidcClientCredentials struct {
	FederatedIdentities []FederatedIdentity `json:"federatedIdentities"`
}

// OidcClientRequest is the body of POST /api/oidc/clients (OidcClientCreateDto)
// and PUT /api/oidc/clients/{id} (OidcClientUpdateDto). ID is only meaningful
// on creation and must be left empty on update.
type OidcClientRequest struct {
	ID                                  string                `json:"id,omitempty"`
	Name                                string                `json:"name"`
	Description                         string                `json:"description"`
	CallbackURLs                        []string              `json:"callbackURLs"`
	LogoutCallbackURLs                  []string              `json:"logoutCallbackURLs"`
	LaunchURL                           string                `json:"launchURL,omitempty"`
	LogoURL                             string                `json:"logoUrl,omitempty"`
	DarkLogoURL                         string                `json:"darkLogoUrl,omitempty"`
	HasLogo                             bool                  `json:"hasLogo"`
	HasDarkLogo                         bool                  `json:"hasDarkLogo"`
	IsPublic                            bool                  `json:"isPublic"`
	PkceEnabled                         bool                  `json:"pkceEnabled"`
	RequiresReauthentication            bool                  `json:"requiresReauthentication"`
	RequiresPushedAuthorizationRequests bool                  `json:"requiresPushedAuthorizationRequests"`
	SkipConsent                         bool                  `json:"skipConsent"`
	IsGroupRestricted                   bool                  `json:"isGroupRestricted"`
	AccessTokenDurationMinutes          *int                  `json:"accessTokenDurationMinutes,omitempty"`
	RefreshTokenDurationMinutes         *int                  `json:"refreshTokenDurationMinutes,omitempty"`
	BackchannelLogoutURL                string                `json:"backchannelLogoutURL,omitempty"`
	Credentials                         OidcClientCredentials `json:"credentials"`
}

// UserGroupRef is the minimal user group representation embedded in a client.
type UserGroupRef struct {
	ID string `json:"id"`
}

// OidcClient mirrors OidcClientWithAllowedUserGroupsDto. The client secrets
// are never exposed here.
type OidcClient struct {
	ID                                  string                `json:"id"`
	Name                                string                `json:"name"`
	Description                         string                `json:"description"`
	ClientType                          string                `json:"clientType"`
	CallbackURLs                        []string              `json:"callbackURLs"`
	LogoutCallbackURLs                  []string              `json:"logoutCallbackURLs"`
	LaunchURL                           string                `json:"launchURL,omitempty"`
	HasLogo                             bool                  `json:"hasLogo"`
	HasDarkLogo                         bool                  `json:"hasDarkLogo"`
	IsPublic                            bool                  `json:"isPublic"`
	PkceEnabled                         bool                  `json:"pkceEnabled"`
	PkceSupported                       bool                  `json:"pkceSupported"`
	RequiresReauthentication            bool                  `json:"requiresReauthentication"`
	RequiresPushedAuthorizationRequests bool                  `json:"requiresPushedAuthorizationRequests"`
	SkipConsent                         bool                  `json:"skipConsent"`
	IsGroupRestricted                   bool                  `json:"isGroupRestricted"`
	AccessTokenDurationMinutes          int                   `json:"accessTokenDurationMinutes"`
	RefreshTokenDurationMinutes         int                   `json:"refreshTokenDurationMinutes"`
	BackchannelLogoutURL                string                `json:"backchannelLogoutURL,omitempty"`
	Credentials                         OidcClientCredentials `json:"credentials"`
	AllowedUserGroups                   []UserGroupRef        `json:"allowedUserGroups"`
}

// AllowedUserGroupIDs returns the IDs of the user groups allowed on the client.
func (c *OidcClient) AllowedUserGroupIDs() []string {
	ids := make([]string, 0, len(c.AllowedUserGroups))
	for _, g := range c.AllowedUserGroups {
		ids = append(ids, g.ID)
	}
	return ids
}

// OidcClientSecret is a client secret as listed by the API (never the value).
type OidcClientSecret struct {
	ID        string `json:"id"`
	Prefix    string `json:"prefix"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
	IsActive  bool   `json:"isActive"`
}

// OidcClientSecretCreated is the response of a secret creation; Secret is the
// plain-text value and is only returned once.
type OidcClientSecretCreated struct {
	OidcClientSecret
	Secret string `json:"secret"`
}

// OidcClientCreated mirrors OidcClientCreatedDto (the client plus the secret
// Pocket-ID may generate for confidential clients).
type OidcClientCreated struct {
	OidcClient
	CreatedSecret *OidcClientSecretCreated `json:"createdSecret"`
}

func oidcClientPath(id string) string { return "/api/oidc/clients/" + url.PathEscape(id) }

func (r *OidcClientRequest) normalize() {
	if r.CallbackURLs == nil {
		r.CallbackURLs = []string{}
	}
	if r.LogoutCallbackURLs == nil {
		r.LogoutCallbackURLs = []string{}
	}
	if r.Credentials.FederatedIdentities == nil {
		r.Credentials.FederatedIdentities = []FederatedIdentity{}
	}
}

// CreateOidcClient creates an OIDC client (POST /api/oidc/clients).
func (c *Client) CreateOidcClient(ctx context.Context, req OidcClientRequest) (*OidcClientCreated, error) {
	req.normalize()
	var out OidcClientCreated
	if err := c.Do(ctx, http.MethodPost, "/api/oidc/clients", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetOidcClient fetches a client with its allowed user groups.
func (c *Client) GetOidcClient(ctx context.Context, id string) (*OidcClient, error) {
	var out OidcClient
	if err := c.Do(ctx, http.MethodGet, oidcClientPath(id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateOidcClient replaces a client's settings (PUT /api/oidc/clients/{id}).
func (c *Client) UpdateOidcClient(ctx context.Context, id string, req OidcClientRequest) (*OidcClient, error) {
	req.ID = ""
	req.normalize()
	var out OidcClient
	if err := c.Do(ctx, http.MethodPut, oidcClientPath(id), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteOidcClient deletes a client.
func (c *Client) DeleteOidcClient(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, oidcClientPath(id), nil, nil, nil)
}

// SetOidcClientAllowedUserGroups replaces the set of user groups allowed on
// the client (PUT /api/oidc/clients/{id}/allowed-user-groups).
func (c *Client) SetOidcClientAllowedUserGroups(ctx context.Context, id string, groupIDs []string) error {
	if groupIDs == nil {
		groupIDs = []string{}
	}
	body := map[string]any{"userGroupIds": groupIDs}
	return c.Do(ctx, http.MethodPut, oidcClientPath(id)+"/allowed-user-groups", nil, body, nil)
}

// ListOidcClientSecrets lists the secrets of a client (metadata only).
func (c *Client) ListOidcClientSecrets(ctx context.Context, clientID string) ([]OidcClientSecret, error) {
	var out []OidcClientSecret
	if err := c.Do(ctx, http.MethodGet, oidcClientPath(clientID)+"/secrets", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateOidcClientSecret generates a new secret; expiresAt (RFC 3339) is optional.
func (c *Client) CreateOidcClientSecret(ctx context.Context, clientID, expiresAt string) (*OidcClientSecretCreated,
	error) {
	body := map[string]any{}
	if expiresAt != "" {
		body["expiresAt"] = expiresAt
	}
	var out OidcClientSecretCreated
	if err := c.Do(ctx, http.MethodPost, oidcClientPath(clientID)+"/secrets", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteOidcClientSecret revokes one secret of a client.
func (c *Client) DeleteOidcClientSecret(ctx context.Context, clientID, secretID string) error {
	return c.Do(ctx, http.MethodDelete, oidcClientPath(clientID)+"/secrets/"+url.PathEscape(secretID), nil, nil, nil)
}

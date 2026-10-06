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

// ScimServiceProvider mirrors ScimServiceProviderDTO.
type ScimServiceProvider struct {
	ID           string `json:"id"`
	Endpoint     string `json:"endpoint"`
	Token        string `json:"token"`
	CreatedAt    string `json:"createdAt"`
	LastSyncedAt string `json:"lastSyncedAt"`
}

// ScimServiceProviderRequest is the body of POST/PUT /api/scim/service-provider.
type ScimServiceProviderRequest struct {
	OidcClientID string `json:"oidcClientId"`
	Endpoint     string `json:"endpoint"`
	Token        string `json:"token,omitempty"`
}

// CreateScimServiceProvider attaches a SCIM service provider to an OIDC client.
func (c *Client) CreateScimServiceProvider(ctx context.Context,
	req ScimServiceProviderRequest) (*ScimServiceProvider, error) {
	var out ScimServiceProvider
	if err := c.Do(ctx, http.MethodPost, "/api/scim/service-provider", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateScimServiceProvider replaces a service provider's endpoint and token.
func (c *Client) UpdateScimServiceProvider(ctx context.Context, id string,
	req ScimServiceProviderRequest) (*ScimServiceProvider, error) {
	var out ScimServiceProvider
	if err := c.Do(ctx, http.MethodPut, "/api/scim/service-provider/"+url.PathEscape(id), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteScimServiceProvider removes a service provider.
func (c *Client) DeleteScimServiceProvider(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "/api/scim/service-provider/"+url.PathEscape(id), nil, nil, nil)
}

// GetOidcClientScimServiceProvider returns the service provider of a client
// (404 when it has none).
func (c *Client) GetOidcClientScimServiceProvider(ctx context.Context, clientID string) (*ScimServiceProvider, error) {
	var out ScimServiceProvider
	if err := c.Do(ctx, http.MethodGet, oidcClientPath(clientID)+"/scim-service-provider", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

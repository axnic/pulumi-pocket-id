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

// APIPermission is a permission declared by an API (resource server).
type APIPermission struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// AllowedForCimdClients tells whether CIMD-registered clients may request this permission.
	AllowedForCimdClients bool `json:"allowedForCimdClients"`
}

// APIPermissionInput is the writable shape of an API permission.
type APIPermissionInput struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// API is a Pocket-ID API (OAuth resource server).
type API struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Resource    string          `json:"resource"`
	CreatedAt   string          `json:"createdAt"`
	Permissions []APIPermission `json:"permissions"`
	// AllowCimdClients tells whether clients registered through a Client ID Metadata Document may use the API.
	AllowCimdClients bool `json:"allowCimdClients"`
}

// CreateAPI creates an API with the given display name and resource identifier.
func (c *Client) CreateAPI(ctx context.Context, name, resource string) (*API, error) {
	var out API
	body := map[string]string{keyName: name, "resource": resource}
	if err := c.Do(ctx, http.MethodPost, "/api/apis", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAPI fetches an API with its permissions.
func (c *Client) GetAPI(ctx context.Context, id string) (*API, error) {
	var out API
	if err := c.Do(ctx, http.MethodGet, "/api/apis/"+url.PathEscape(id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateAPI renames an API (the resource identifier is immutable).
func (c *Client) UpdateAPI(ctx context.Context, id, name string) (*API, error) {
	var out API
	body := map[string]string{keyName: name}
	if err := c.Do(ctx, http.MethodPut, "/api/apis/"+url.PathEscape(id), nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteAPI deletes an API.
func (c *Client) DeleteAPI(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "/api/apis/"+url.PathEscape(id), nil, nil, nil)
}

// SetAPIPermissions replaces the complete permission set of an API.
func (c *Client) SetAPIPermissions(ctx context.Context, id string, perms []APIPermissionInput) (*API, error) {
	if perms == nil {
		perms = []APIPermissionInput{}
	}
	var out API
	body := map[string]any{"permissions": perms}
	if err := c.Do(ctx, http.MethodPut, "/api/apis/"+url.PathEscape(id)+"/permissions", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetAPICimdAccess replaces the permissions that any client registered through a Client ID Metadata Document
// may request on an API.
func (c *Client) SetAPICimdAccess(ctx context.Context, id string, enabled bool, permissionIDs []string) error {
	if permissionIDs == nil {
		permissionIDs = []string{}
	}
	body := map[string]any{"enabled": enabled, "permissionIds": permissionIDs}
	return c.Do(ctx, http.MethodPut, "/api/apis/"+url.PathEscape(id)+"/cimd-access", nil, body, nil)
}

// APIClientGrant is the access an OIDC client has on an API, expressed in permission IDs.
type APIClientGrant struct {
	ClientAccess               bool     `json:"clientAccess"`
	ClientPermissionIDs        []string `json:"clientPermissionIds"`
	UserDelegatedAccess        bool     `json:"userDelegatedAccess"`
	UserDelegatedPermissionIDs []string `json:"userDelegatedPermissionIds"`
}

// APIClientAccess is one entry of an API's client list.
type APIClientAccess struct {
	APIClientGrant
	Client struct {
		ID string `json:"id"`
	} `json:"client"`
}

// ListAPIClients lists the clients that have access to an API.
func (c *Client) ListAPIClients(ctx context.Context, apiID string) ([]APIClientAccess, error) {
	return listAll[APIClientAccess](ctx, c, "/api/apis/"+url.PathEscape(apiID)+"/clients", nil)
}

// SetAPIClientGrant replaces the access of a client on an API.
func (c *Client) SetAPIClientGrant(ctx context.Context, apiID, clientID string, g APIClientGrant) error {
	if g.ClientPermissionIDs == nil {
		g.ClientPermissionIDs = []string{}
	}
	if g.UserDelegatedPermissionIDs == nil {
		g.UserDelegatedPermissionIDs = []string{}
	}
	return c.Do(ctx, http.MethodPut, apiClientPath(apiID, clientID), nil, g, nil)
}

// DeleteAPIClientGrant removes all access of a client on an API.
func (c *Client) DeleteAPIClientGrant(ctx context.Context, apiID, clientID string) error {
	return c.Do(ctx, http.MethodDelete, apiClientPath(apiID, clientID), nil, nil, nil)
}

func apiClientPath(apiID, clientID string) string {
	return "/api/apis/" + url.PathEscape(apiID) + "/clients/" + url.PathEscape(clientID)
}

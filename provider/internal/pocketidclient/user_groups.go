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

// UserGroupMinimal is the summary form of a group (UserGroupMinimalDto),
// returned by the list endpoint and embedded in users.
type UserGroupMinimal struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	FriendlyName string        `json:"friendlyName"`
	CustomClaims []CustomClaim `json:"customClaims"`
}

// UserGroup is a full user group (UserGroupDto), members included.
type UserGroup struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	FriendlyName string        `json:"friendlyName"`
	LdapID       *string       `json:"ldapId"`
	CustomClaims []CustomClaim `json:"customClaims"`
	Users        []User        `json:"users"`
}

// UserGroupInput is the body of group creation and update.
type UserGroupInput struct {
	Name         string `json:"name"`
	FriendlyName string `json:"friendlyName"`
}

// CreateUserGroup creates a group.
func (c *Client) CreateUserGroup(ctx context.Context, in UserGroupInput) (*UserGroup, error) {
	var out UserGroup
	if err := c.Do(ctx, http.MethodPost, "/api/user-groups", nil, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUserGroup fetches a group (with its members) by ID.
func (c *Client) GetUserGroup(ctx context.Context, id string) (*UserGroup, error) {
	var out UserGroup
	if err := c.Do(ctx, http.MethodGet, "/api/user-groups/"+url.PathEscape(id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateUserGroup updates a group's name and friendly name.
func (c *Client) UpdateUserGroup(ctx context.Context, id string, in UserGroupInput) (*UserGroup, error) {
	var out UserGroup
	if err := c.Do(ctx, http.MethodPut, "/api/user-groups/"+url.PathEscape(id), nil, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteUserGroup deletes a group.
func (c *Client) DeleteUserGroup(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "/api/user-groups/"+url.PathEscape(id), nil, nil, nil)
}

// ListUserGroups returns every group matching the optional search string.
func (c *Client) ListUserGroups(ctx context.Context, search string) ([]UserGroupMinimal, error) {
	q := url.Values{}
	if search != "" {
		q.Set("search", search)
	}
	return listAll[UserGroupMinimal](ctx, c, "/api/user-groups", q)
}

// SetUserGroupMembers replaces the complete member set of a group.
func (c *Client) SetUserGroupMembers(ctx context.Context, id string, userIDs []string) (*UserGroup, error) {
	if userIDs == nil {
		userIDs = []string{}
	}
	var out UserGroup
	body := struct {
		UserIDs []string `json:"userIds"`
	}{userIDs}
	if err := c.Do(ctx, http.MethodPut, "/api/user-groups/"+url.PathEscape(id)+"/users", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

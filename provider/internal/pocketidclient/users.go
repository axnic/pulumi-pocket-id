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

// User is a Pocket-ID user as returned by the API (UserDto).
type User struct {
	ID            string             `json:"id"`
	Username      string             `json:"username"`
	Email         *string            `json:"email"`
	EmailVerified bool               `json:"emailVerified"`
	FirstName     string             `json:"firstName"`
	LastName      *string            `json:"lastName"`
	DisplayName   string             `json:"displayName"`
	IsAdmin       bool               `json:"isAdmin"`
	Locale        *string            `json:"locale"`
	Disabled      bool               `json:"disabled"`
	LdapID        *string            `json:"ldapId"`
	CustomClaims  []CustomClaim      `json:"customClaims"`
	UserGroups    []UserGroupMinimal `json:"userGroups"`
}

// UserInput is the body of user creation and update (UserCreateDto). Group
// membership is deliberately not sent: it is managed from the group side.
type UserInput struct {
	Username      string  `json:"username"`
	Email         *string `json:"email,omitempty"`
	EmailVerified bool    `json:"emailVerified"`
	FirstName     *string `json:"firstName,omitempty"`
	LastName      *string `json:"lastName,omitempty"`
	DisplayName   *string `json:"displayName,omitempty"`
	IsAdmin       bool    `json:"isAdmin"`
	Locale        *string `json:"locale,omitempty"`
	Disabled      bool    `json:"disabled"`
}

// CreateUser creates a user.
func (c *Client) CreateUser(ctx context.Context, in UserInput) (*User, error) {
	var out User
	if err := c.Do(ctx, http.MethodPost, "/api/users", nil, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUser fetches a user by ID.
func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	var out User
	if err := c.Do(ctx, http.MethodGet, "/api/users/"+url.PathEscape(id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateUser replaces a user's attributes.
func (c *Client) UpdateUser(ctx context.Context, id string, in UserInput) (*User, error) {
	var out User
	if err := c.Do(ctx, http.MethodPut, "/api/users/"+url.PathEscape(id), nil, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteUser deletes a user.
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "/api/users/"+url.PathEscape(id), nil, nil, nil)
}

// ListUsers returns every user matching the optional search string.
func (c *Client) ListUsers(ctx context.Context, search string) ([]User, error) {
	q := url.Values{}
	if search != "" {
		q.Set("search", search)
	}
	return listAll[User](ctx, c, "/api/users", q)
}

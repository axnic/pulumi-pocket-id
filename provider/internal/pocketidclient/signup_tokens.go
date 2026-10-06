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
	"time"
)

// SignupToken is a signup token (signupTokenDto).
type SignupToken struct {
	ID         string             `json:"id"`
	Token      string             `json:"token"`
	CreatedAt  time.Time          `json:"createdAt"`
	ExpiresAt  time.Time          `json:"expiresAt"`
	UsageCount int                `json:"usageCount"`
	UsageLimit int                `json:"usageLimit"`
	UserGroups []UserGroupMinimal `json:"userGroups"`
}

// SignupTokenInput is the body of signup token creation. TTLSeconds is the
// token lifetime in seconds.
type SignupTokenInput struct {
	TTLSeconds   int64    `json:"ttl"`
	UsageLimit   int      `json:"usageLimit"`
	UserGroupIDs []string `json:"userGroupIds,omitempty"`
}

// CreateSignupToken creates a signup token.
func (c *Client) CreateSignupToken(ctx context.Context, in SignupTokenInput) (*SignupToken, error) {
	var out SignupToken
	if err := c.Do(ctx, http.MethodPost, "/api/signup-tokens", nil, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListSignupTokens returns every signup token.
func (c *Client) ListSignupTokens(ctx context.Context) ([]SignupToken, error) {
	return listAll[SignupToken](ctx, c, "/api/signup-tokens", nil)
}

// DeleteSignupToken deletes a signup token.
func (c *Client) DeleteSignupToken(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "/api/signup-tokens/"+url.PathEscape(id), nil, nil, nil)
}

// GetSignupToken looks a token up in the list (the API has no get-by-ID
// endpoint). It returns a 404 *APIError when the token does not exist.
func (c *Client) GetSignupToken(ctx context.Context, id string) (*SignupToken, error) {
	tokens, err := c.ListSignupTokens(ctx)
	if err != nil {
		return nil, err
	}
	for i := range tokens {
		if tokens[i].ID == id {
			return &tokens[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: "signup token not found"}
}

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

// CustomClaim is a key/value claim attached to a user or a user group.
type CustomClaim struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SetUserCustomClaims replaces the complete claim list of a user.
func (c *Client) SetUserCustomClaims(ctx context.Context, userID string, claims []CustomClaim) ([]CustomClaim, error) {
	return c.setClaims(ctx, "/api/custom-claims/user/"+url.PathEscape(userID), claims)
}

// SetUserGroupCustomClaims replaces the complete claim list of a user group.
func (c *Client) SetUserGroupCustomClaims(
	ctx context.Context, groupID string, claims []CustomClaim,
) ([]CustomClaim, error) {
	return c.setClaims(ctx, "/api/custom-claims/user-group/"+url.PathEscape(groupID), claims)
}

func (c *Client) setClaims(ctx context.Context, path string, claims []CustomClaim) ([]CustomClaim, error) {
	if claims == nil {
		claims = []CustomClaim{}
	}
	var out []CustomClaim
	if err := c.Do(ctx, http.MethodPut, path, nil, claims, &out); err != nil {
		return nil, err
	}
	return out, nil
}

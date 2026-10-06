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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetCustomClaims(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	claims := []CustomClaim{{Key: "role", Value: "dev"}}

	c, call := identityServer(t, 200, `[{"key":"role","value":"dev"}]`)
	out, err := c.SetUserCustomClaims(ctx, "u1", claims)
	require.NoError(t, err)
	assert.Equal(t, claims, out)
	assert.Equal(t, "PUT /api/custom-claims/user/u1 key", call.method+" "+call.path+" "+call.apiKey)
	assert.JSONEq(t, `[{"key":"role","value":"dev"}]`, call.rawBody)

	c, call = identityServer(t, 200, `[]`)
	_, err = c.SetUserGroupCustomClaims(ctx, "g1", nil)
	require.NoError(t, err)
	assert.Equal(t, "PUT /api/custom-claims/user-group/g1", call.method+" "+call.path)
	assert.JSONEq(t, `[]`, call.rawBody)

	c, _ = identityServer(t, 404, `{"error":"nope"}`)
	_, err = c.SetUserCustomClaims(ctx, "x", claims)
	assert.True(t, IsNotFound(err))
}

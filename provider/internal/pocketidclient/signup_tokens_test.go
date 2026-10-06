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

func TestSignupTokens(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c, call := identityServer(t, 201, `{"id":"s1","token":"secret","usageLimit":3}`)
	tok, err := c.CreateSignupToken(ctx, SignupTokenInput{TTLSeconds: 3600, UsageLimit: 3, UserGroupIDs: []string{"g1"}})
	require.NoError(t, err)
	assert.Equal(t, "secret", tok.Token)
	assert.Equal(t, "POST /api/signup-tokens key", call.method+" "+call.path+" "+call.apiKey)
	assert.JSONEq(t, `{"ttl":3600,"usageLimit":3,"userGroupIds":["g1"]}`, call.rawBody)

	list := `{"data":[{"id":"s1","token":"a"},{"id":"s2","token":"b"}],"pagination":{"currentPage":1,"totalPages":1}}`
	c, call = identityServer(t, 200, list)
	tok, err = c.GetSignupToken(ctx, "s2")
	require.NoError(t, err)
	assert.Equal(t, "b", tok.Token)
	assert.Equal(t, "GET /api/signup-tokens", call.method+" "+call.path)
	_, err = c.GetSignupToken(ctx, "zzz")
	assert.True(t, IsNotFound(err))

	c, call = identityServer(t, 204, ``)
	require.NoError(t, c.DeleteSignupToken(ctx, "s1"))
	assert.Equal(t, "DELETE /api/signup-tokens/s1", call.method+" "+call.path)
	c, _ = identityServer(t, 404, `{"error":"nope"}`)
	assert.True(t, IsNotFound(c.DeleteSignupToken(ctx, "s1")))
}

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

func TestScimServiceProviderLifecycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c, s := recorder(t, 201, `{"id":"sp1","endpoint":"https://scim.example.com"}`)
	got, err := c.CreateScimServiceProvider(ctx, ScimServiceProviderRequest{OidcClientID: valApp,
		Endpoint: "https://scim.example.com", Token: "t"})
	require.NoError(t, err)
	assert.Equal(t, "POST /api/scim/service-provider", s.method+" "+s.path)
	assert.Equal(t, map[string]any{"oidcClientId": valApp, "endpoint": "https://scim.example.com", "token": "t"}, s.body)
	assert.Equal(t, "sp1", got.ID)

	c, s = recorder(t, 200, `{"id":"sp1"}`)
	_, err = c.UpdateScimServiceProvider(ctx, "sp1", ScimServiceProviderRequest{OidcClientID: valApp, Endpoint: "e"})
	require.NoError(t, err)
	assert.Equal(t, "PUT /api/scim/service-provider/sp1", s.method+" "+s.path)
	assert.NotContains(t, s.body, "token")

	c, s = recorder(t, 200, `{"id":"sp1","endpoint":"e"}`)
	got, err = c.GetOidcClientScimServiceProvider(ctx, valApp)
	require.NoError(t, err)
	assert.Equal(t, "GET /api/oidc/clients/app/scim-service-provider", s.method+" "+s.path)
	assert.Equal(t, "e", got.Endpoint)

	c, s = recorder(t, 204, "")
	require.NoError(t, c.DeleteScimServiceProvider(ctx, "sp1"))
	assert.Equal(t, "DELETE /api/scim/service-provider/sp1", s.method+" "+s.path)

	c, _ = recorder(t, 404, `{"error":"none"}`)
	_, err = c.GetOidcClientScimServiceProvider(ctx, valApp)
	assert.True(t, IsNotFound(err))
}

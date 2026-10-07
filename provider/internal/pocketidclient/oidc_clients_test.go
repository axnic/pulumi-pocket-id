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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type seen struct {
	method, path string
	body         map[string]any
}

// recorder serves status/resp for every request and records the last one.
func recorder(t *testing.T, status int, resp string) (*Client, *seen) {
	t.Helper()
	s := &seen{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "k", r.Header.Get("X-API-Key"))
		s.method, s.path = r.Method, r.URL.EscapedPath()
		b, _ := io.ReadAll(r.Body)
		s.body = nil
		if len(b) > 0 {
			require.NoError(t, json.Unmarshal(b, &s.body))
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(resp))
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL, "k", nil), s
}

func TestCreateOidcClient(t *testing.T) {
	t.Parallel()
	c, s := recorder(t, 201,
		`{"id":"app","name":"App","createdSecret":{"id":"s1","secret":"x"},"allowedUserGroups":[{"id":"g1"}]}`)
	got, err := c.CreateOidcClient(context.Background(), OidcClientRequest{ID: valApp, Name: "App"})
	require.NoError(t, err)
	assert.Equal(t, "POST /api/oidc/clients", s.method+" "+s.path)
	assert.Equal(t, valApp, s.body["id"])
	assert.Equal(t, []any{}, s.body["callbackURLs"], "nil slices are sent as empty arrays")
	assert.Equal(t, "s1", got.CreatedSecret.ID)
	assert.Equal(t, []string{"g1"}, got.AllowedUserGroupIDs())
}

func TestUpdateOidcClientOmitsID(t *testing.T) {
	t.Parallel()
	c, s := recorder(t, 200, `{"id":"app"}`)
	_, err := c.UpdateOidcClient(context.Background(), valApp, OidcClientRequest{ID: valApp, Name: "N"})
	require.NoError(t, err)
	assert.Equal(t, "PUT /api/oidc/clients/app", s.method+" "+s.path)
	assert.NotContains(t, s.body, "id")
}

func TestGetAndDeleteOidcClient(t *testing.T) {
	t.Parallel()
	c, s := recorder(t, 200, `{"id":"app","name":"App","pkceSupported":true}`)
	got, err := c.GetOidcClient(context.Background(), valApp)
	require.NoError(t, err)
	assert.True(t, got.PkceSupported)
	assert.Equal(t, "GET /api/oidc/clients/app", s.method+" "+s.path)

	c, s = recorder(t, 204, "")
	require.NoError(t, c.DeleteOidcClient(context.Background(), valApp))
	assert.Equal(t, "DELETE /api/oidc/clients/app", s.method+" "+s.path)
}

func TestOidcClientNotFound(t *testing.T) {
	t.Parallel()
	c, _ := recorder(t, 404, `{"error":"record not found"}`)
	_, err := c.GetOidcClient(context.Background(), "nope")
	assert.True(t, IsNotFound(err))
}

func TestSetOidcClientAllowedUserGroups(t *testing.T) {
	t.Parallel()
	c, s := recorder(t, 200, `{}`)
	require.NoError(t, c.SetOidcClientAllowedUserGroups(context.Background(), valApp, nil))
	assert.Equal(t, "PUT /api/oidc/clients/app/allowed-user-groups", s.method+" "+s.path)
	assert.Equal(t, map[string]any{"userGroupIds": []any{}}, s.body)
}

func TestOidcClientSecrets(t *testing.T) {
	t.Parallel()
	c, s := recorder(t, 201, `{"id":"s1","secret":"plain","prefix":"abc"}`)
	got, err := c.CreateOidcClientSecret(context.Background(), valApp, "2030-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.Equal(t, "POST /api/oidc/clients/app/secrets", s.method+" "+s.path)
	assert.Equal(t, "2030-01-01T00:00:00Z", s.body["expiresAt"])
	assert.Equal(t, "plain", got.Secret)
	assert.Equal(t, "s1", got.ID)

	c, s = recorder(t, 201, `{"id":"s2"}`)
	_, err = c.CreateOidcClientSecret(context.Background(), valApp, "")
	require.NoError(t, err)
	assert.NotContains(t, s.body, "expiresAt")

	c, s = recorder(t, 200, `[{"id":"s1","isActive":true},{"id":"s2"}]`)
	list, err := c.ListOidcClientSecrets(context.Background(), valApp)
	require.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, "GET /api/oidc/clients/app/secrets", s.method+" "+s.path)

	c, s = recorder(t, 204, "")
	require.NoError(t, c.DeleteOidcClientSecret(context.Background(), valApp, "s1"))
	assert.Equal(t, "DELETE /api/oidc/clients/app/secrets/s1", s.method+" "+s.path)
}

// Pocket-ID rejects an optional URL sent as "" with a 400: unset ones must be absent from the request.
func TestOidcClientRequestOmitsEmptyURLs(t *testing.T) {
	t.Parallel()
	c, s := recorder(t, 201, `{"id":"app"}`)
	_, err := c.CreateOidcClient(context.Background(), OidcClientRequest{Name: "App"})
	require.NoError(t, err)
	for _, k := range []string{"launchURL", "backchannelLogoutURL"} {
		assert.NotContains(t, s.body, k)
	}
}

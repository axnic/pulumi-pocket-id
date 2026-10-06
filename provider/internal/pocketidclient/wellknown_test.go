// Copyright 2025, axnic.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOpenIDConfiguration(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/.well-known/openid-configuration", r.URL.Path)
		_, _ = w.Write([]byte(`{"issuer":"https://id.example.com",` +
			`"jwks_uri":"https://id.example.com/.well-known/jwks.json","scopes_supported":["openid","email"]}`))
	}))
	defer srv.Close()

	got, err := New(srv.URL, "k", nil).GetOpenIDConfiguration(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "https://id.example.com", got.Issuer)
	assert.Equal(t, "https://id.example.com/.well-known/jwks.json", got.JwksURI)
	assert.Equal(t, []string{"openid", "email"}, got.ScopesSupported)
}

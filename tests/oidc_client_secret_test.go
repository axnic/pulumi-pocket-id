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

package tests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

func TestOidcClientSecretLifecycle(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	_, err := prov.Create(p.CreateRequest{Urn: urn("OidcClient"), Properties: oidcMap(map[string]property.Value{
		keyClientID: property.New("app"), keyName: property.New("App"),
	})})
	require.NoError(t, err)

	in1 := oidcMap(map[string]property.Value{keyClientID: property.New("app")})
	in2 := in1.Set(keyExpiresAt, property.New("2099-01-01T00:00:00Z"))
	s1, err := prov.Create(p.CreateRequest{Urn: urn("OidcClientSecret"), Properties: in1})
	require.NoError(t, err)
	s2, err := prov.Create(p.CreateRequest{Urn: urn("OidcClientSecret"), Properties: in2})
	require.NoError(t, err)
	assert.NotEqual(t, s1.ID, s2.ID, "secrets coexist")
	assert.Equal(t, 2, oidcCount(fake, "oidcSecrets"))

	secret := s1.Properties.Get("secret")
	assert.Equal(t, "plain-"+s1.Properties.Get("secretId").AsString(), secret.AsString())
	assert.Equal(t, s1.ID, "app/"+s1.Properties.Get("secretId").AsString())
	assert.Equal(t, "2099-01-01T00:00:00Z", s2.Properties.Get(keyExpiresAt).AsString())

	// Read keeps the plain-text secret from the state.
	read, err := prov.Read(p.ReadRequest{ID: s1.ID, Urn: urn("OidcClientSecret"), Properties: s1.Properties, Inputs: in1})
	require.NoError(t, err)
	assert.Equal(t, s1.ID, read.ID)
	assert.Equal(t, secret.AsString(), read.Properties.Get("secret").AsString())

	// Import: no secret value, expiresAt read from the API.
	imp, err := prov.Read(p.ReadRequest{ID: s2.ID, Urn: urn("OidcClientSecret")})
	require.NoError(t, err)
	assert.Equal(t, "app", imp.Inputs.Get(keyClientID).AsString())
	assert.Equal(t, "2099-01-01T00:00:00Z", imp.Inputs.Get(keyExpiresAt).AsString())
	assert.Equal(t, "", imp.Properties.Get("secret").AsString())

	// Deleting one secret leaves the other one.
	require.NoError(t, prov.Delete(p.DeleteRequest{ID: s1.ID, Urn: urn("OidcClientSecret"), Properties: s1.Properties}))
	assert.Equal(t, 1, oidcCount(fake, "oidcSecrets"))
	gone, err := prov.Read(p.ReadRequest{ID: s1.ID, Urn: urn("OidcClientSecret"), Properties: s1.Properties, Inputs: in1})
	require.NoError(t, err)
	assert.Empty(t, gone.ID)
	require.NoError(t, prov.Delete(p.DeleteRequest{ID: s1.ID, Urn: urn("OidcClientSecret"), Properties: s1.Properties}))
}

func TestOidcClientSecretReadClientGone(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	gone, err := prov.Read(p.ReadRequest{ID: "nope/sec-1", Urn: urn("OidcClientSecret")})
	require.NoError(t, err)
	assert.Empty(t, gone.ID)
}

func TestOidcClientSecretPreviewAndReplace(t *testing.T) {
	t.Parallel()
	prov := testServer(t)
	configure(t, prov, "http://127.0.0.1:1", "k")

	in := oidcMap(map[string]property.Value{keyClientID: property.New("app")})
	_, err := prov.Create(p.CreateRequest{Urn: urn("OidcClientSecret"), Properties: in, DryRun: true})
	require.NoError(t, err)

	state := in.Set("secretId", property.New("s")).Set("secret", property.New("x")).
		Set("prefix", property.New("p")).Set(keyCreatedAt, property.New("t"))
	for name, news := range map[string]property.Map{
		keyClientID:  oidcMap(map[string]property.Value{keyClientID: property.New("other")}),
		keyExpiresAt: in.Set(keyExpiresAt, property.New("2099-01-01T00:00:00Z")),
	} {
		resp, err := prov.Diff(p.DiffRequest{ID: "app/s", Urn: urn("OidcClientSecret"), State: state, Inputs: news})
		require.NoError(t, err)
		assert.Contains(t, []p.DiffKind{p.UpdateReplace, p.AddReplace}, resp.DetailedDiff[name].Kind, name)
	}
}

func TestOidcSecretsAreMarkedSecretInSchema(t *testing.T) {
	t.Parallel()
	resp, err := testServer(t).GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)
	schema := resp.Schema
	for _, c := range []struct{ res, in, out string }{
		{"pocket-id:index:OidcClientSecret", "", "secret"},
		{"pocket-id:index:ScimServiceProvider", keyToken, keyToken},
	} {
		spec := gjsonPath(t, schema, c.res)
		if c.in != "" {
			assert.Contains(t, spec, `"`+c.in+`":{`+"")
		}
		assert.Contains(t, spec, `"`+c.out+`"`)
		assert.Contains(t, spec, `"secret":true`)
	}
}

// gjsonPath returns the JSON of one resource of a schema document.
func gjsonPath(t *testing.T, schema, token string) string {
	t.Helper()
	var doc struct {
		Resources map[string]json.RawMessage `json:"resources"`
	}
	require.NoError(t, json.Unmarshal([]byte(schema), &doc))
	raw, ok := doc.Resources[token]
	require.True(t, ok, token)
	return string(raw)
}

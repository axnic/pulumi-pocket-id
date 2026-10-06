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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

func TestScimServiceProviderLifecycle(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	_, err := prov.Create(p.CreateRequest{Urn: urn("OidcClient"), Properties: oidcMap(map[string]property.Value{
		keyClientID: property.New("app"), keyName: property.New("App"),
	})})
	require.NoError(t, err)

	in := oidcMap(map[string]property.Value{
		keyOidcClientID: property.New("app"),
		keyEndpoint:     property.New("https://scim.example.com/v2"),
		keyToken:        property.New("tok"),
	})
	created, err := prov.Create(p.CreateRequest{Urn: urn("ScimServiceProvider"), Properties: in})
	require.NoError(t, err)
	assert.Contains(t, created.ID, "scim-")

	// Read goes through the client and keeps the token from the state.
	read, err := prov.Read(p.ReadRequest{ID: created.ID, Urn: urn("ScimServiceProvider"),
		Properties: created.Properties, Inputs: in})
	require.NoError(t, err)
	assert.Equal(t, created.ID, read.ID)
	assert.Equal(t, "https://scim.example.com/v2", read.Properties.Get(keyEndpoint).AsString())
	assert.Equal(t, "tok", read.Properties.Get(keyToken).AsString())

	updated, err := prov.Update(p.UpdateRequest{
		ID: created.ID, Urn: urn("ScimServiceProvider"), State: created.Properties,
		Inputs: in.Set(keyEndpoint, property.New("https://scim.example.com/v3")),
	})
	require.NoError(t, err)
	assert.Equal(t, "https://scim.example.com/v3", updated.Properties.Get(keyEndpoint).AsString())

	// Import with <oidcClientId>/<serviceProviderId>; the canonical ID is the provider's.
	imp, err := prov.Read(p.ReadRequest{ID: "app/" + created.ID, Urn: urn("ScimServiceProvider")})
	require.NoError(t, err)
	assert.Equal(t, created.ID, imp.ID)
	assert.Equal(t, "app", imp.Inputs.Get(keyOidcClientID).AsString())

	require.NoError(t, prov.Delete(p.DeleteRequest{ID: created.ID, Urn: urn("ScimServiceProvider"),
		Properties: updated.Properties}))
	assert.Equal(t, 0, oidcCount(fake, "scimProviders"))
	require.NoError(t, prov.Delete(p.DeleteRequest{ID: created.ID, Urn: urn("ScimServiceProvider"),
		Properties: updated.Properties}))

	gone, err := prov.Read(p.ReadRequest{ID: created.ID, Urn: urn("ScimServiceProvider"),
		Properties: updated.Properties, Inputs: in})
	require.NoError(t, err)
	assert.Empty(t, gone.ID)
}

func TestScimServiceProviderPreviewAndReplace(t *testing.T) {
	t.Parallel()
	prov := testServer(t)
	configure(t, prov, "http://127.0.0.1:1", "k")

	in := oidcMap(map[string]property.Value{keyOidcClientID: property.New("app"), keyEndpoint: property.New("https://e")})
	created, err := prov.Create(p.CreateRequest{Urn: urn("ScimServiceProvider"), Properties: in, DryRun: true})
	require.NoError(t, err)

	updated, err := prov.Update(p.UpdateRequest{
		ID: "x", Urn: urn("ScimServiceProvider"), DryRun: true, State: created.Properties,
		Inputs: in.Set(keyEndpoint, property.New("https://f")),
	})
	require.NoError(t, err)
	assert.Equal(t, "https://f", updated.Properties.Get(keyEndpoint).AsString())

	state := in.Set(keyCreatedAt, property.New("t")).Set("lastSyncedAt", property.New(""))
	resp, err := prov.Diff(p.DiffRequest{ID: "x", Urn: urn("ScimServiceProvider"), State: state,
		Inputs: in.Set(keyOidcClientID, property.New("other"))})
	require.NoError(t, err)
	assert.Equal(t, p.UpdateReplace, resp.DetailedDiff[keyOidcClientID].Kind)

	resp, err = prov.Diff(p.DiffRequest{ID: "x", Urn: urn("ScimServiceProvider"), State: state,
		Inputs: in.Set(keyEndpoint, property.New("https://g"))})
	require.NoError(t, err)
	assert.Equal(t, p.Update, resp.DetailedDiff[keyEndpoint].Kind)
}

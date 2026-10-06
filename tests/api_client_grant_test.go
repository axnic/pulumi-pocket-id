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

func grantInputs(apiID, clientID string, extra map[string]property.Value) property.Map {
	m := map[string]property.Value{"apiId": property.New(apiID), keyClientID: property.New(clientID)}
	for k, v := range extra {
		m[k] = v
	}
	return property.NewMap(m)
}

func grantStrs(m property.Map, key string) []string {
	var out []string
	for _, v := range m.Get(key).AsArray().AsSlice() {
		out = append(out, v.AsString())
	}
	return out
}

func TestApiClientGrantLifecycle(t *testing.T) {
	t.Parallel()
	fake, prov := apiSetup(t)
	api, err := prov.Create(p.CreateRequest{
		Urn: urn("Api"), Properties: apiInputs("Orders", "https://orders", apiPerm(valRead, "Read", ""), apiPerm("write",
			"Write", "")),
	})
	require.NoError(t, err)
	ids := apiPermIDs(api.Properties)

	in := grantInputs(api.ID, "client-1", map[string]property.Value{
		keyClientAccess:               property.New(true),
		keyClientPermissionKeys:       apiStrs("write", valRead),
		keyUserDelegatedAccess:        property.New(true),
		"userDelegatedPermissionKeys": apiStrs(valRead),
	})
	created, err := prov.Create(p.CreateRequest{Urn: urn("ApiClientGrant"), Properties: in})
	require.NoError(t, err)
	assert.Equal(t, api.ID+"/client-1", created.ID)

	// The API receives permission IDs, not keys.
	g := fake.store("apiGrants")[created.ID]
	assert.ElementsMatch(t, []any{ids[valRead], ids["write"]}, g["clientPermissionIds"])
	assert.Equal(t, []any{ids[valRead]}, g["userDelegatedPermissionIds"])

	// Read converts IDs back to keys and keeps the declared order (no drift).
	read, err := prov.Read(p.ReadRequest{ID: created.ID, Urn: urn("ApiClientGrant"), Properties: created.Properties})
	require.NoError(t, err)
	assert.Equal(t, created.ID, read.ID)
	assert.Equal(t, []string{"write", valRead}, grantStrs(read.Properties, keyClientPermissionKeys))
	assert.True(t, read.Properties.Get(keyClientAccess).AsBool())

	// Update.
	newIn := grantInputs(api.ID, "client-1", map[string]property.Value{
		keyClientAccess:         property.New(true),
		keyClientPermissionKeys: apiStrs(valRead),
	})
	updated, err := prov.Update(p.UpdateRequest{ID: created.ID, Urn: urn("ApiClientGrant"), State: created.Properties,
		Inputs: newIn})
	require.NoError(t, err)
	assert.Equal(t, []string{valRead}, grantStrs(updated.Properties, keyClientPermissionKeys))
	g = fake.store("apiGrants")[created.ID]
	assert.Equal(t, []any{ids[valRead]}, g["clientPermissionIds"])
	assert.Equal(t, false, g[keyUserDelegatedAccess])

	// Delete revokes the access.
	require.NoError(t, prov.Delete(p.DeleteRequest{ID: created.ID, Urn: urn("ApiClientGrant"),
		Properties: updated.Properties}))
	assert.Empty(t, fake.store("apiGrants"))
	require.NoError(t, prov.Delete(p.DeleteRequest{ID: created.ID, Urn: urn("ApiClientGrant"),
		Properties: updated.Properties}))

	// Read now reports the grant as gone.
	read, err = prov.Read(p.ReadRequest{ID: created.ID, Urn: urn("ApiClientGrant"), Properties: created.Properties})
	require.NoError(t, err)
	assert.Empty(t, read.ID)
}

func TestApiClientGrantImport(t *testing.T) {
	t.Parallel()
	_, prov := apiSetup(t)
	api, err := prov.Create(p.CreateRequest{
		Urn: urn("Api"), Properties: apiInputs("Orders", "https://orders", apiPerm(valRead, "Read", "")),
	})
	require.NoError(t, err)
	_, err = prov.Create(p.CreateRequest{Urn: urn("ApiClientGrant"), Properties: grantInputs(api.ID, "c1",
		map[string]property.Value{
			keyUserDelegatedAccess:        property.New(true),
			"userDelegatedPermissionKeys": apiStrs(valRead),
		})})
	require.NoError(t, err)

	imported, err := prov.Read(p.ReadRequest{ID: api.ID + "/c1", Urn: urn("ApiClientGrant")})
	require.NoError(t, err)
	assert.Equal(t, api.ID+"/c1", imported.ID)
	assert.Equal(t, api.ID, imported.Properties.Get("apiId").AsString())
	assert.Equal(t, "c1", imported.Properties.Get(keyClientID).AsString())
	assert.Equal(t, []string{valRead}, grantStrs(imported.Properties, "userDelegatedPermissionKeys"))

	_, err = prov.Read(p.ReadRequest{ID: "malformed", Urn: urn("ApiClientGrant")})
	assert.Error(t, err)

	gone, err := prov.Read(p.ReadRequest{ID: api.ID + "/other", Urn: urn("ApiClientGrant")})
	require.NoError(t, err)
	assert.Empty(t, gone.ID)
	gone, err = prov.Read(p.ReadRequest{ID: "api-404/c1", Urn: urn("ApiClientGrant")})
	require.NoError(t, err)
	assert.Empty(t, gone.ID)
}

func TestApiClientGrantUnknownKeyAndPreview(t *testing.T) {
	t.Parallel()
	fake, prov := apiSetup(t)
	api, err := prov.Create(p.CreateRequest{
		Urn: urn("Api"), Properties: apiInputs("Orders", "https://orders", apiPerm(valRead, "Read", "")),
	})
	require.NoError(t, err)

	bad := grantInputs(api.ID, "c1", map[string]property.Value{keyClientPermissionKeys: apiStrs("nope")})
	_, err = prov.Create(p.CreateRequest{Urn: urn("ApiClientGrant"), Properties: bad})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nope")

	before := apiHits(fake)
	resp, err := prov.Create(p.CreateRequest{Urn: urn("ApiClientGrant"), DryRun: true, Properties: bad})
	require.NoError(t, err)
	assert.Equal(t, api.ID+"/c1", resp.ID)
	assert.Equal(t, before, apiHits(fake))
}

func TestApiClientGrantDiffReplaces(t *testing.T) {
	t.Parallel()
	_, prov := apiSetup(t)
	old := grantInputs("a1", "c1", nil)
	diff, err := prov.Diff(p.DiffRequest{ID: "a1/c1", Urn: urn("ApiClientGrant"), State: old, Inputs: grantInputs("a1",
		"c2", nil)})
	require.NoError(t, err)
	assert.Equal(t, p.UpdateReplace, diff.DetailedDiff[keyClientID].Kind)
	diff, err = prov.Diff(p.DiffRequest{ID: "a1/c1", Urn: urn("ApiClientGrant"), State: old, Inputs: grantInputs("a2",
		"c1", nil)})
	require.NoError(t, err)
	assert.Equal(t, p.UpdateReplace, diff.DetailedDiff["apiId"].Kind)
}

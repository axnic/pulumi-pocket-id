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
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

func apiSetup(t *testing.T) (*fakePocketID, integration.Server) {
	t.Helper()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)
	return fake, prov
}

func apiStrs(s ...string) property.Value {
	vals := make([]property.Value, 0, len(s))
	for _, v := range s {
		vals = append(vals, property.New(v))
	}
	return property.New(property.NewArray(vals))
}

func apiPerm(key, name, desc string) property.Value {
	m := map[string]property.Value{keyKey: property.New(key), keyName: property.New(name)}
	if desc != "" {
		m[keyDescription] = property.New(desc)
	}
	return property.New(property.NewMap(m))
}

func apiInputs(name, resource string, perms ...property.Value) property.Map {
	return property.NewMap(map[string]property.Value{
		keyName:       property.New(name),
		"resource":    property.New(resource),
		"permissions": property.New(property.NewArray(perms)),
	})
}

func apiPermIDs(m property.Map) map[string]string {
	ids := map[string]string{}
	for _, v := range m.Get("permissions").AsArray().AsSlice() {
		pm := v.AsMap()
		ids[pm.Get(keyKey).AsString()] = pm.Get("id").AsString()
	}
	return ids
}

func TestApiLifecycle(t *testing.T) {
	t.Parallel()
	fake, prov := apiSetup(t)

	inputs := apiInputs("Orders", "https://orders", apiPerm(valRead, "Read", "Read orders"), apiPerm("write", "Write", ""))
	created, err := prov.Create(p.CreateRequest{Urn: urn("Api"), Properties: inputs})
	require.NoError(t, err)
	id := created.ID
	assert.Contains(t, id, "api-")
	assert.Equal(t, "Orders", created.Properties.Get(keyName).AsString())
	ids := apiPermIDs(created.Properties)
	assert.Len(t, ids, 2)
	assert.Contains(t, ids[valRead], "perm-")

	read, err := prov.Read(p.ReadRequest{ID: id, Urn: urn("Api"), Properties: created.Properties})
	require.NoError(t, err)
	assert.Equal(t, id, read.ID)
	assert.Equal(t, "https://orders", read.Properties.Get("resource").AsString())
	assert.Equal(t, ids, apiPermIDs(read.Properties))

	// No drift between the read inputs and the state.
	diff, err := prov.Diff(p.DiffRequest{ID: id, Urn: urn("Api"), State: read.Properties, Inputs: inputs})
	require.NoError(t, err)
	assert.False(t, diff.HasChanges)

	// Update: rename, drop "write", add "admin"; valRead keeps its ID.
	newIn := apiInputs("Orders v2", "https://orders", apiPerm(valRead, "Read", "Read orders"), apiPerm("admin", "Admin",
		""))
	diff, err = prov.Diff(p.DiffRequest{ID: id, Urn: urn("Api"), State: created.Properties, Inputs: newIn})
	require.NoError(t, err)
	assert.True(t, diff.HasChanges)
	assert.Equal(t, p.Update, diff.DetailedDiff[keyName].Kind)
	assert.Equal(t, p.Update, diff.DetailedDiff["permissions"].Kind)

	updated, err := prov.Update(p.UpdateRequest{ID: id, Urn: urn("Api"), State: created.Properties, Inputs: newIn})
	require.NoError(t, err)
	assert.Equal(t, "Orders v2", updated.Properties.Get(keyName).AsString())
	newIDs := apiPermIDs(updated.Properties)
	assert.Len(t, newIDs, 2)
	assert.Equal(t, ids[valRead], newIDs[valRead])
	assert.NotContains(t, newIDs, "write")

	// Remote state matches.
	assert.Equal(t, "Orders v2", fake.store("apis")[id][keyName])
	assert.Len(t, fake.store("apis")[id]["permissions"], 2)

	require.NoError(t, prov.Delete(p.DeleteRequest{ID: id, Urn: urn("Api"), Properties: updated.Properties}))
	assert.Empty(t, fake.store("apis"))
	// Delete is idempotent.
	require.NoError(t, prov.Delete(p.DeleteRequest{ID: id, Urn: urn("Api"), Properties: updated.Properties}))
}

func TestApiReadGoneAndImport(t *testing.T) {
	t.Parallel()
	_, prov := apiSetup(t)

	gone, err := prov.Read(p.ReadRequest{ID: "api-404", Urn: urn("Api")})
	require.NoError(t, err)
	assert.Empty(t, gone.ID)

	created, err := prov.Create(p.CreateRequest{
		Urn: urn("Api"), Properties: apiInputs("Orders", "https://orders", apiPerm(valRead, "Read", "")),
	})
	require.NoError(t, err)

	imported, err := prov.Read(p.ReadRequest{ID: created.ID, Urn: urn("Api")})
	require.NoError(t, err)
	assert.Equal(t, created.ID, imported.ID)
	assert.Equal(t, "Orders", imported.Properties.Get(keyName).AsString())
	assert.Equal(t, "https://orders", imported.Properties.Get("resource").AsString())
	assert.Len(t, apiPermIDs(imported.Properties), 1)
}

func TestApiPreviewMakesNoCalls(t *testing.T) {
	t.Parallel()
	fake, prov := apiSetup(t)

	resp, err := prov.Create(p.CreateRequest{
		Urn: urn("Api"), DryRun: true, Properties: apiInputs("Orders", "https://orders", apiPerm(valRead, "Read", "")),
	})
	require.NoError(t, err)
	assert.Equal(t, "Orders", resp.Properties.Get(keyName).AsString())
	assert.Zero(t, apiHits(fake))
}

func TestApiDiffReplacesOnResource(t *testing.T) {
	t.Parallel()
	_, prov := apiSetup(t)

	created, err := prov.Create(p.CreateRequest{Urn: urn("Api"), Properties: apiInputs("Orders", "https://orders")})
	require.NoError(t, err)
	diff, err := prov.Diff(p.DiffRequest{
		ID: created.ID, Urn: urn("Api"), State: created.Properties, Inputs: apiInputs("Orders", "https://other"),
	})
	require.NoError(t, err)
	assert.True(t, diff.HasChanges)
	assert.Equal(t, p.UpdateReplace, diff.DetailedDiff["resource"].Kind)
}

func apiCimd(enabled bool, keys ...string) property.Value {
	return property.New(property.NewMap(map[string]property.Value{
		"enabled": property.New(enabled), "permissionKeys": apiStrs(keys...),
	}))
}

func apiWithCimd(in property.Map, cimd property.Value) property.Map {
	m := in.AsMap()
	m["cimdAccess"] = cimd
	return property.NewMap(m)
}

func cimdAllowedKeys(fake *fakePocketID, id string) []string {
	var keys []string
	for _, v := range fake.store("apis")[id]["permissions"].([]any) {
		pm := v.(map[string]any)
		if pm["allowedForCimdClients"] == true {
			keys = append(keys, pm[keyKey].(string))
		}
	}
	return keys
}

func TestApiCimdAccess(t *testing.T) {
	t.Parallel()
	fake, prov := apiSetup(t)
	base := apiInputs("Orders", "https://orders", apiPerm(valRead, "Read", ""), apiPerm("write", "Write", ""))
	inputs := apiWithCimd(base, apiCimd(true, valRead))

	created, err := prov.Create(p.CreateRequest{Urn: urn("Api"), Properties: inputs})
	require.NoError(t, err)
	id := created.ID
	assert.Equal(t, true, fake.store("apis")[id]["allowCimdClients"])
	assert.Equal(t, []string{valRead}, cimdAllowedKeys(fake, id))
	assert.True(t, created.Properties.Get("cimdAccess").AsMap().Get("enabled").AsBool())

	// Read of a managed API returns the server value; no drift.
	read, err := prov.Read(p.ReadRequest{ID: id, Urn: urn("Api"), Properties: created.Properties, Inputs: inputs})
	require.NoError(t, err)
	diff, err := prov.Diff(p.DiffRequest{ID: id, Urn: urn("Api"), State: read.Properties, Inputs: inputs})
	require.NoError(t, err)
	assert.False(t, diff.HasChanges)

	// Update the CIMD set.
	in2 := apiWithCimd(base, apiCimd(true, valRead, "write"))
	diff, err = prov.Diff(p.DiffRequest{ID: id, Urn: urn("Api"), State: created.Properties, Inputs: in2})
	require.NoError(t, err)
	assert.Equal(t, p.Update, diff.DetailedDiff["cimdAccess"].Kind)
	up, err := prov.Update(p.UpdateRequest{ID: id, Urn: urn("Api"), State: created.Properties, Inputs: in2})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{valRead, "write"}, cimdAllowedKeys(fake, id))

	// Removing a referenced permission requires dropping it from cimdAccess too.
	readOnly := apiInputs("Orders", "https://orders", apiPerm(valRead, "Read", ""))
	in3 := apiWithCimd(readOnly, apiCimd(true, valRead, "write"))
	_, err = prov.Update(p.UpdateRequest{ID: id, Urn: urn("Api"), State: up.Properties, Inputs: in3})
	require.ErrorContains(t, err, `"write" is not declared`)
	in4 := apiWithCimd(readOnly, apiCimd(false, valRead))
	up, err = prov.Update(p.UpdateRequest{ID: id, Urn: urn("Api"), State: up.Properties, Inputs: in4})
	require.NoError(t, err)
	assert.Equal(t, false, fake.store("apis")[id]["allowCimdClients"])
	assert.Equal(t, []string{valRead}, cimdAllowedKeys(fake, id))

	// Absent cimdAccess: server value untouched and not read back.
	in5 := apiInputs("Orders", "https://orders", apiPerm(valRead, "Read", ""))
	noCimd := apiWithCimd(up.Properties, property.New(property.Null))
	diff, err = prov.Diff(p.DiffRequest{ID: id, Urn: urn("Api"), State: noCimd, Inputs: in5})
	require.NoError(t, err)
	assert.False(t, diff.HasChanges)
	refreshed, err := prov.Read(p.ReadRequest{ID: id, Urn: urn("Api"), Properties: noCimd, Inputs: in5})
	require.NoError(t, err)
	assert.False(t, refreshed.Properties.Get("cimdAccess").IsComputed() ||
		refreshed.Properties.Get("cimdAccess").IsMap())
}

func TestApiCimdImport(t *testing.T) {
	t.Parallel()
	_, prov := apiSetup(t)
	created, err := prov.Create(p.CreateRequest{Urn: urn("Api"),
		Properties: apiWithCimd(apiInputs("Orders", "https://orders", apiPerm(valRead, "Read", "")), apiCimd(true, valRead))})
	require.NoError(t, err)
	imported, err := prov.Read(p.ReadRequest{ID: created.ID, Urn: urn("Api")})
	require.NoError(t, err)
	cimd := imported.Properties.Get("cimdAccess").AsMap()
	assert.True(t, cimd.Get("enabled").AsBool())
	assert.Equal(t, valRead, cimd.Get("permissionKeys").AsArray().Get(0).AsString())
}

func TestApiCimdPreviewAndUnknownKey(t *testing.T) {
	t.Parallel()
	fake, prov := apiSetup(t)
	base := apiInputs("Orders", "https://orders", apiPerm(valRead, "Read", ""))

	resp, err := prov.Create(p.CreateRequest{
		Urn: urn("Api"), DryRun: true, Properties: apiWithCimd(base, apiCimd(true, valRead)),
	})
	require.NoError(t, err)
	assert.True(t, resp.Properties.Get("cimdAccess").AsMap().Get("enabled").AsBool())
	assert.Zero(t, apiHits(fake))

	_, err = prov.Create(p.CreateRequest{Urn: urn("Api"), Properties: apiWithCimd(base, apiCimd(true, "nope"))})
	require.ErrorContains(t, err, `"nope" is not declared`)
	assert.Empty(t, fake.store("apis"))
}

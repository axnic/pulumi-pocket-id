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

// claimsOf extracts a claims property as a plain map.
func claimsOf(v property.Value) map[string]string {
	out := map[string]string{}
	for k, e := range v.AsMap().AsMap() {
		out[k] = e.AsString()
	}
	return out
}

// testClaimsResource runs the authoritative-claims cycle for an owner kind.
func testClaimsResource(t *testing.T, resType, ownerType, ownerName, ownerKey string,
	ownerProps map[string]property.Value) {
	t.Helper()
	prov := identityEnv(t)
	owner, _ := idCreate(t, prov, ownerType, ownerProps)

	claims := map[string]string{valTeam: "core", "tier": "1"}
	id, props := idCreate(t, prov, resType, map[string]property.Value{
		ownerKey: property.New(owner), keyClaims: idStringMap(claims),
	})
	assert.Equal(t, owner, id)

	read, err := prov.Read(p.ReadRequest{ID: id, Urn: urn(resType), Properties: props})
	require.NoError(t, err)
	assert.Equal(t, claims, claimsOf(read.Properties.Get(keyClaims)))

	// Update replaces the full set: "tier" disappears.
	newClaims := map[string]string{valTeam: "platform"}
	upd, err := prov.Update(p.UpdateRequest{
		ID: id, Urn: urn(resType), State: props,
		Inputs: property.NewMap(map[string]property.Value{ownerKey: property.New(owner), keyClaims: idStringMap(newClaims)}),
	})
	require.NoError(t, err)
	read, err = prov.Read(p.ReadRequest{ID: id, Urn: urn(resType), Properties: upd.Properties})
	require.NoError(t, err)
	assert.Equal(t, newClaims, claimsOf(read.Properties.Get(keyClaims)))

	// Import from the ID alone.
	imp, err := prov.Read(p.ReadRequest{ID: id, Urn: urn(resType), Properties: property.NewMap(nil)})
	require.NoError(t, err)
	assert.Equal(t, owner, imp.Properties.Get(ownerKey).AsString())
	assert.Equal(t, newClaims, claimsOf(imp.Properties.Get(keyClaims)))

	// Delete clears the claims but keeps the owner.
	require.NoError(t, prov.Delete(p.DeleteRequest{ID: id, Urn: urn(resType), Properties: upd.Properties}))
	read, err = prov.Read(p.ReadRequest{ID: id, Urn: urn(resType), Properties: upd.Properties})
	require.NoError(t, err)
	assert.Equal(t, id, read.ID)
	assert.Empty(t, claimsOf(read.Properties.Get(keyClaims)))

	// Owner gone: Read drops the resource, Delete is idempotent.
	read, err = prov.Read(p.ReadRequest{ID: ownerName + "-404", Urn: urn(resType), Properties: property.NewMap(nil)})
	require.NoError(t, err)
	assert.Empty(t, read.ID)
	require.NoError(t, prov.Delete(p.DeleteRequest{
		ID: ownerName + "-404", Urn: urn(resType),
		Properties: property.NewMap(map[string]property.Value{ownerKey: property.New(ownerName + "-404"),
			keyClaims: idStringMap(nil)}),
	}))

	// Replace on owner change, plain update on claim change.
	state := property.NewMap(map[string]property.Value{ownerKey: property.New("a"), keyClaims: idStringMap(claims)})
	diff, err := prov.Diff(p.DiffRequest{
		ID: "a", Urn: urn(resType), State: state,
		Inputs: property.NewMap(map[string]property.Value{ownerKey: property.New("b"), keyClaims: idStringMap(claims)}),
	})
	require.NoError(t, err)
	assert.Equal(t, p.UpdateReplace, diff.DetailedDiff[ownerKey].Kind)
	diff, err = prov.Diff(p.DiffRequest{
		ID: "a", Urn: urn(resType), State: state,
		Inputs: property.NewMap(map[string]property.Value{ownerKey: property.New("a"), keyClaims: idStringMap(newClaims)}),
	})
	require.NoError(t, err)
	assert.True(t, diff.HasChanges)
	for _, d := range diff.DetailedDiff {
		assert.NotContains(t, string(d.Kind), "replace")
	}

	// Preview makes no API call.
	_, err = identityOffline(t).Create(p.CreateRequest{
		Urn: urn(resType), DryRun: true,
		Properties: property.NewMap(map[string]property.Value{ownerKey: property.New("x"), keyClaims: idStringMap(claims)}),
	})
	require.NoError(t, err)
}

func TestUserCustomClaimsLifecycle(t *testing.T) {
	t.Parallel()
	testClaimsResource(t, "UserCustomClaims", "User", "usr", keyUserID,
		map[string]property.Value{keyUsername: property.New("erin")})
}

func TestUserGroupCustomClaimsLifecycle(t *testing.T) {
	t.Parallel()
	testClaimsResource(t, "UserGroupCustomClaims", "UserGroup", "grp", keyUserGroupID,
		map[string]property.Value{keyName: property.New("g"), keyFriendlyName: property.New("G")})
}

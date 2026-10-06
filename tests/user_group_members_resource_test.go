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

func TestUserGroupMembersLifecycle(t *testing.T) {
	t.Parallel()
	prov := identityEnv(t)
	u1, _ := idCreate(t, prov, "User", map[string]property.Value{keyUsername: property.New("u1")})
	u2, _ := idCreate(t, prov, "User", map[string]property.Value{keyUsername: property.New("u2")})
	g, _ := idCreate(t, prov, "UserGroup", map[string]property.Value{
		keyName: property.New("g"), keyFriendlyName: property.New("G"),
	})

	id, props := idCreate(t, prov, "UserGroupMembers", map[string]property.Value{
		keyGroupID: property.New(g), keyUserIDs: idStrings(u2, u1),
	})
	assert.Equal(t, g, id)

	// Read keeps the order of the state even though the API order may differ.
	read, err := prov.Read(p.ReadRequest{ID: id, Urn: urn("UserGroupMembers"), Properties: props})
	require.NoError(t, err)
	assert.Equal(t, []string{u2, u1}, idStringSlice(read.Properties.Get(keyUserIDs)))

	// The user sees its membership through the group.
	got, err := prov.Invoke(p.InvokeRequest{ //nolint:gosec // not a credential
		Token: "pocket-id:index:getUser", Args: property.NewMap(map[string]property.Value{keyUserID: property.New(u1)}),
	})
	require.NoError(t, err)
	assert.Equal(t, []string{g}, idStringSlice(got.Return.Get(keyUserGroupIDs)))

	upd, err := prov.Update(p.UpdateRequest{
		ID: id, Urn: urn("UserGroupMembers"), State: props,
		Inputs: property.NewMap(map[string]property.Value{keyGroupID: property.New(g), keyUserIDs: idStrings(u1)}),
	})
	require.NoError(t, err)
	assert.Equal(t, []string{u1}, idStringSlice(upd.Properties.Get(keyUserIDs)))
	read, err = prov.Read(p.ReadRequest{ID: id, Urn: urn("UserGroupMembers"), Properties: upd.Properties})
	require.NoError(t, err)
	assert.Equal(t, []string{u1}, idStringSlice(read.Properties.Get(keyUserIDs)))

	// Import from the ID alone.
	imp, err := prov.Read(p.ReadRequest{ID: id, Urn: urn("UserGroupMembers"), Properties: property.NewMap(nil)})
	require.NoError(t, err)
	assert.Equal(t, g, imp.Properties.Get(keyGroupID).AsString())
	assert.Equal(t, []string{u1}, idStringSlice(imp.Properties.Get(keyUserIDs)))

	// Delete empties the group but keeps it.
	require.NoError(t, prov.Delete(p.DeleteRequest{ID: id, Urn: urn("UserGroupMembers"), Properties: upd.Properties}))
	grp, err := prov.Invoke(p.InvokeRequest{ //nolint:gosec // not a credential
		Token: "pocket-id:index:getUserGroup",
		Args:  property.NewMap(map[string]property.Value{keyUserGroupID: property.New(g)}),
	})
	require.NoError(t, err)
	assert.Empty(t, idStringSlice(grp.Return.Get(keyUserIDs)))
}

func TestUserGroupMembersGroupGone(t *testing.T) {
	t.Parallel()
	prov := identityEnv(t)
	read, err := prov.Read(p.ReadRequest{ID: "grp-404", Urn: urn("UserGroupMembers"), Properties: property.NewMap(nil)})
	require.NoError(t, err)
	assert.Empty(t, read.ID)
	require.NoError(t, prov.Delete(p.DeleteRequest{
		ID: "grp-404", Urn: urn("UserGroupMembers"),
		Properties: property.NewMap(map[string]property.Value{keyGroupID: property.New("grp-404"), keyUserIDs: idStrings()}),
	}))
}

func TestUserGroupMembersReplaceOnGroupChange(t *testing.T) {
	t.Parallel()
	prov := identityEnv(t)
	state := property.NewMap(map[string]property.Value{keyGroupID: property.New("g1"), keyUserIDs: idStrings("u1")})
	resp, err := prov.Diff(p.DiffRequest{
		ID: "g1", Urn: urn("UserGroupMembers"), State: state,
		Inputs: property.NewMap(map[string]property.Value{keyGroupID: property.New("g2"), keyUserIDs: idStrings("u1")}),
	})
	require.NoError(t, err)
	assert.True(t, resp.HasChanges)
	assert.Equal(t, p.UpdateReplace, resp.DetailedDiff[keyGroupID].Kind)

	resp, err = prov.Diff(p.DiffRequest{
		ID: "g1", Urn: urn("UserGroupMembers"), State: state,
		Inputs: property.NewMap(map[string]property.Value{keyGroupID: property.New("g1"), keyUserIDs: idStrings("u1", "u2")}),
	})
	require.NoError(t, err)
	assert.True(t, resp.HasChanges)
	for _, d := range resp.DetailedDiff {
		assert.NotContains(t, string(d.Kind), "replace")
	}
}

func TestUserGroupMembersPreviewMakesNoAPICall(t *testing.T) {
	t.Parallel()
	prov := identityOffline(t)
	_, err := prov.Create(p.CreateRequest{
		Urn: urn("UserGroupMembers"), DryRun: true,
		Properties: property.NewMap(map[string]property.Value{keyGroupID: property.New("g"), keyUserIDs: idStrings("u")}),
	})
	require.NoError(t, err)
}

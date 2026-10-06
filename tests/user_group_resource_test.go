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

func TestUserGroupResourceLifecycle(t *testing.T) {
	t.Parallel()
	prov := identityEnv(t)

	inputs := map[string]property.Value{keyName: property.New("admins"), keyFriendlyName: property.New("Admins")}
	id, props := idCreate(t, prov, "UserGroup", inputs)
	assert.Contains(t, id, "grp-")

	read, err := prov.Read(p.ReadRequest{ID: id, Urn: urn("UserGroup"), Properties: props})
	require.NoError(t, err)
	assert.Equal(t, "Admins", read.Properties.Get(keyFriendlyName).AsString())

	inputs[keyFriendlyName] = property.New("Administrators")
	upd, err := prov.Update(p.UpdateRequest{ID: id, Urn: urn("UserGroup"), State: props, Inputs: property.NewMap(inputs)})
	require.NoError(t, err)
	assert.Equal(t, "Administrators", upd.Properties.Get(keyFriendlyName).AsString())

	// Import: Read from the ID alone.
	imp, err := prov.Read(p.ReadRequest{ID: id, Urn: urn("UserGroup"), Properties: property.NewMap(nil)})
	require.NoError(t, err)
	assert.Equal(t, "admins", imp.Properties.Get(keyName).AsString())
	assert.Equal(t, "Administrators", imp.Inputs.Get(keyFriendlyName).AsString())

	require.NoError(t, prov.Delete(p.DeleteRequest{ID: id, Urn: urn("UserGroup"), Properties: upd.Properties}))
	require.NoError(t, prov.Delete(p.DeleteRequest{ID: id, Urn: urn("UserGroup"), Properties: upd.Properties}))
	read, err = prov.Read(p.ReadRequest{ID: id, Urn: urn("UserGroup"), Properties: upd.Properties})
	require.NoError(t, err)
	assert.Empty(t, read.ID)
}

func TestUserGroupResourcePreviewMakesNoAPICall(t *testing.T) {
	t.Parallel()
	prov := identityOffline(t)
	_, err := prov.Create(p.CreateRequest{
		Urn: urn("UserGroup"), DryRun: true,
		Properties: property.NewMap(map[string]property.Value{
			keyName: property.New("g"), keyFriendlyName: property.New("G"),
		}),
	})
	require.NoError(t, err)
}

func TestGetUserGroupFunction(t *testing.T) {
	t.Parallel()
	prov := identityEnv(t)
	grpID, _ := idCreate(t, prov, "UserGroup", map[string]property.Value{
		keyName: property.New("ops"), keyFriendlyName: property.New("Operations"),
	})
	idCreate(t, prov, "UserGroup", map[string]property.Value{
		keyName: property.New("ops-extra"), keyFriendlyName: property.New("Ops extra"),
	})
	uid, _ := idCreate(t, prov, "User", map[string]property.Value{keyUsername: property.New("dave")})
	idCreate(t, prov, "UserGroupMembers", map[string]property.Value{
		keyGroupID: property.New(grpID), keyUserIDs: idStrings(uid),
	})

	invoke := func(args map[string]property.Value) (p.InvokeResponse, error) {
		return prov.Invoke(p.InvokeRequest{ //nolint:gosec // not a credential
			Token: "pocket-id:index:getUserGroup",
			Args:  property.NewMap(args),
		})
	}
	byName, err := invoke(map[string]property.Value{keyName: property.New("ops")})
	require.NoError(t, err)
	assert.Equal(t, grpID, byName.Return.Get(keyUserGroupID).AsString())
	assert.Equal(t, "Operations", byName.Return.Get(keyFriendlyName).AsString())
	assert.Equal(t, []string{uid}, idStringSlice(byName.Return.Get(keyUserIDs)))

	byID, err := invoke(map[string]property.Value{keyUserGroupID: property.New(grpID)})
	require.NoError(t, err)
	assert.Equal(t, "ops", byID.Return.Get(keyName).AsString())

	_, err = invoke(map[string]property.Value{keyName: property.New("missing")})
	assert.ErrorContains(t, err, "found 0")
	_, err = invoke(map[string]property.Value{keyUserGroupID: property.New("a"), keyName: property.New("b")})
	assert.ErrorContains(t, err, "exactly one of userGroupId or name")
}

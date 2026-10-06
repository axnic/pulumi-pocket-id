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

func TestUserResourceLifecycle(t *testing.T) {
	t.Parallel()
	prov := identityEnv(t)

	inputs := map[string]property.Value{
		keyUsername: property.New("alice"), keyEmail: property.New("alice@example.com"),
		keyFirstName: property.New("Alice"), keyDisplayName: property.New("Alice Liddell"),
		keyIsAdmin: property.New(true),
	}
	id, props := idCreate(t, prov, "User", inputs)
	assert.Contains(t, id, "usr-")
	assert.Equal(t, "alice", props.Get(keyUsername).AsString())
	assert.True(t, props.Get(keyIsAdmin).AsBool())

	read, err := prov.Read(p.ReadRequest{ID: id, Urn: urn("User"), Properties: props})
	require.NoError(t, err)
	assert.Equal(t, id, read.ID)
	assert.Equal(t, "Alice Liddell", read.Properties.Get(keyDisplayName).AsString())
	assert.True(t, read.Properties.Get(keyIsAdmin).AsBool())

	inputs[keyDisplayName] = property.New("Alice in Wonderland")
	inputs[keyDisabled] = property.New(true)
	upd, err := prov.Update(p.UpdateRequest{ID: id, Urn: urn("User"), State: props, Inputs: property.NewMap(inputs)})
	require.NoError(t, err)
	assert.Equal(t, "Alice in Wonderland", upd.Properties.Get(keyDisplayName).AsString())

	read, err = prov.Read(p.ReadRequest{ID: id, Urn: urn("User"), Properties: upd.Properties})
	require.NoError(t, err)
	assert.True(t, read.Properties.Get(keyDisabled).AsBool())
	assert.Equal(t, "Alice in Wonderland", read.Properties.Get(keyDisplayName).AsString())

	require.NoError(t, prov.Delete(p.DeleteRequest{ID: id, Urn: urn("User"), Properties: upd.Properties}))
	// Idempotent: deleting again tolerates the 404.
	require.NoError(t, prov.Delete(p.DeleteRequest{ID: id, Urn: urn("User"), Properties: upd.Properties}))
	read, err = prov.Read(p.ReadRequest{ID: id, Urn: urn("User"), Properties: upd.Properties})
	require.NoError(t, err)
	assert.Empty(t, read.ID, "a deleted user must drop out of state")
}

func TestUserResourceImport(t *testing.T) {
	t.Parallel()
	prov := identityEnv(t)
	id, _ := idCreate(t, prov, "User", map[string]property.Value{
		keyUsername: property.New("bob"), keyFirstName: property.New("Bob"), keyLastName: property.New("Builder"),
		keyLocale: property.New("fr"),
	})

	read, err := prov.Read(p.ReadRequest{ID: id, Urn: urn("User"), Properties: property.NewMap(nil)})
	require.NoError(t, err)
	assert.Equal(t, id, read.ID)
	assert.Equal(t, "bob", read.Properties.Get(keyUsername).AsString())
	assert.Equal(t, "Builder", read.Properties.Get(keyLastName).AsString())
	assert.Equal(t, "fr", read.Inputs.Get(keyLocale).AsString())
}

func TestUserResourcePreviewMakesNoAPICall(t *testing.T) {
	t.Parallel()
	prov := identityOffline(t)
	resp, err := prov.Create(p.CreateRequest{
		Urn: urn("User"), DryRun: true,
		Properties: property.NewMap(map[string]property.Value{keyUsername: property.New("alice")}),
	})
	require.NoError(t, err)
	assert.Equal(t, "alice", resp.Properties.Get(keyUsername).AsString())
}

func TestGetUserFunction(t *testing.T) {
	t.Parallel()
	prov := identityEnv(t)
	id, _ := idCreate(t, prov, "User", map[string]property.Value{
		keyUsername: property.New("carol"), keyEmail: property.New("carol@example.com"),
	})
	idCreate(t, prov, "User", map[string]property.Value{keyUsername: property.New("carol2")})
	grpID, _ := idCreate(t, prov, "UserGroup", map[string]property.Value{
		keyName: property.New("devs"), keyFriendlyName: property.New("Devs"),
	})
	idCreate(t, prov, "UserGroupMembers", map[string]property.Value{
		keyGroupID: property.New(grpID), keyUserIDs: idStrings(id),
	})
	idCreate(t, prov, "UserCustomClaims", map[string]property.Value{
		keyUserID: property.New(id), keyClaims: idStringMap(map[string]string{valTeam: "core"}),
	})

	invoke := func(args map[string]property.Value) (p.InvokeResponse, error) {
		return prov.Invoke(p.InvokeRequest{ //nolint:gosec // not a credential
			Token: "pocket-id:index:getUser",
			Args:  property.NewMap(args),
		})
	}

	// "carol" also matches "carol2" in the API search; only the exact name counts.
	byName, err := invoke(map[string]property.Value{keyUsername: property.New("carol")})
	require.NoError(t, err)
	assert.Equal(t, id, byName.Return.Get(keyUserID).AsString())
	assert.Equal(t, "carol@example.com", byName.Return.Get(keyEmail).AsString())
	assert.Equal(t, map[string]property.Value{valTeam: property.New("core")},
		byName.Return.Get(keyCustomClaims).AsMap().AsMap())
	assert.Equal(t, []string{grpID}, idStringSlice(byName.Return.Get(keyUserGroupIDs)))

	byID, err := invoke(map[string]property.Value{keyUserID: property.New(id)})
	require.NoError(t, err)
	assert.Equal(t, "carol", byID.Return.Get(keyUsername).AsString())

	_, err = invoke(map[string]property.Value{keyUsername: property.New("nobody")})
	assert.ErrorContains(t, err, "found 0")
	_, err = invoke(map[string]property.Value{})
	assert.ErrorContains(t, err, "exactly one of userId or username")
}

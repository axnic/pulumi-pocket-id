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

func TestSignupTokenLifecycle(t *testing.T) {
	t.Parallel()
	prov := identityEnv(t)
	g, _ := idCreate(t, prov, "UserGroup", map[string]property.Value{
		keyName: property.New("newcomers"), keyFriendlyName: property.New("Newcomers"),
	})

	id, props := idCreate(t, prov, "SignupToken", map[string]property.Value{
		keyTTL: property.New(3600.0), keyUsageLimit: property.New(5.0), keyUserGroupIDs: idStrings(g),
	})
	assert.Contains(t, id, "sgn-")
	secret := props.Get(keyToken).AsString()
	assert.Contains(t, secret, "secret-")
	assert.NotEmpty(t, props.Get(keyExpiresAt).AsString())

	// Read refreshes from the list and keeps the secret from state.
	read, err := prov.Read(p.ReadRequest{ID: id, Urn: urn("SignupToken"), Properties: props})
	require.NoError(t, err)
	assert.Equal(t, id, read.ID)
	assert.Equal(t, secret, read.Properties.Get(keyToken).AsString())
	assert.Equal(t, 5.0, read.Properties.Get(keyUsageLimit).AsNumber())
	assert.Equal(t, []string{g}, idStringSlice(read.Properties.Get(keyUserGroupIDs)))

	// Import: no secret is available, other inputs are derived.
	imp, err := prov.Read(p.ReadRequest{ID: id, Urn: urn("SignupToken"), Properties: property.NewMap(nil)})
	require.NoError(t, err)
	assert.Equal(t, 3600.0, imp.Inputs.Get(keyTTL).AsNumber())
	assert.Equal(t, []string{g}, idStringSlice(imp.Inputs.Get(keyUserGroupIDs)))
	assert.Empty(t, imp.Properties.Get(keyToken).AsString())

	// Every input change forces a replacement.
	state := property.NewMap(map[string]property.Value{
		keyTTL: property.New(3600.0), keyUsageLimit: property.New(5.0), keyUserGroupIDs: idStrings(g),
	})
	for key, val := range map[string]property.Value{
		keyTTL: property.New(60.0), keyUsageLimit: property.New(1.0), keyUserGroupIDs: idStrings(),
	} {
		inputs := state.AsMap()
		inputs[key] = val
		diff, err := prov.Diff(p.DiffRequest{ID: id, Urn: urn("SignupToken"), State: state, Inputs: property.NewMap(inputs)})
		require.NoError(t, err)
		require.NotEmpty(t, diff.DetailedDiff, key)
		for _, d := range diff.DetailedDiff { // list diffs are reported per element
			assert.Contains(t, string(d.Kind), "replace", key)
		}
	}

	require.NoError(t, prov.Delete(p.DeleteRequest{ID: id, Urn: urn("SignupToken"), Properties: props}))
	require.NoError(t, prov.Delete(p.DeleteRequest{ID: id, Urn: urn("SignupToken"), Properties: props}))
	read, err = prov.Read(p.ReadRequest{ID: id, Urn: urn("SignupToken"), Properties: props})
	require.NoError(t, err)
	assert.Empty(t, read.ID)
}

func TestSignupTokenPreviewMakesNoAPICall(t *testing.T) {
	t.Parallel()
	_, err := identityOffline(t).Create(p.CreateRequest{
		Urn: urn("SignupToken"), DryRun: true,
		Properties: property.NewMap(map[string]property.Value{keyTTL: property.New(60.0), keyUsageLimit: property.New(1.0)}),
	})
	require.NoError(t, err)
}

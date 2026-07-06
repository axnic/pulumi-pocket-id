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

func TestCustomClaimsLifecycle(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	// Create a user-group to own the claims.
	grp, err := prov.Create(p.CreateRequest{
		Urn: urn("UserGroup"),
		Properties: property.NewMap(map[string]property.Value{
			"friendlyName": property.New("Engineers"),
			"name":         property.New("engineers"),
		}),
	})
	require.NoError(t, err)

	// Create claims on the group.
	claims := property.NewArray([]property.Value{
		property.New(map[string]property.Value{
			"key":   property.New("department"),
			"value": property.New("engineering"),
		}),
	})
	createResp, err := prov.Create(p.CreateRequest{
		Urn: urn("CustomClaims"),
		Properties: property.NewMap(map[string]property.Value{
			"ownerType": property.New("user-group"),
			"ownerId":   property.New(grp.ID),
			"claims":    property.New(claims),
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, "user-group/"+grp.ID, createResp.ID)

	// Read reflects the stored claims.
	readResp, err := prov.Read(p.ReadRequest{
		ID:  createResp.ID,
		Urn: urn("CustomClaims"),
		Properties: property.NewMap(map[string]property.Value{
			"ownerType": property.New("user-group"),
			"ownerId":   property.New(grp.ID),
		}),
	})
	require.NoError(t, err)
	arr := readResp.Properties.Get("claims").AsArray()
	require.Equal(t, 1, arr.Len())
	first := arr.Get(0).AsMap()
	assert.Equal(t, "department", first.Get("key").AsString())
	assert.Equal(t, "engineering", first.Get("value").AsString())

	// Update replaces the full set.
	claims2 := property.NewArray([]property.Value{
		property.New(map[string]property.Value{
			"key":   property.New("department"),
			"value": property.New("platform"),
		}),
		property.New(map[string]property.Value{
			"key":   property.New("team"),
			"value": property.New("infra"),
		}),
	})
	updateResp, err := prov.Update(p.UpdateRequest{
		ID:  createResp.ID,
		Urn: urn("CustomClaims"),
		State: property.NewMap(map[string]property.Value{
			"ownerType": property.New("user-group"),
			"ownerId":   property.New(grp.ID),
			"claims":    property.New(claims),
		}),
		Inputs: property.NewMap(map[string]property.Value{
			"ownerType": property.New("user-group"),
			"ownerId":   property.New(grp.ID),
			"claims":    property.New(claims2),
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, 2, updateResp.Properties.Get("claims").AsArray().Len())

	// Delete clears the claims.
	err = prov.Delete(p.DeleteRequest{
		ID:  createResp.ID,
		Urn: urn("CustomClaims"),
		Properties: property.NewMap(map[string]property.Value{
			"ownerType": property.New("user-group"),
			"ownerId":   property.New(grp.ID),
		}),
	})
	require.NoError(t, err)
}

func TestCustomClaimsDiffReplaceOnOwnerChange(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	// Changing the ownerId forces a replacement (DeleteBeforeReplace).
	diffResp, err := prov.Diff(p.DiffRequest{
		ID:  "user-group/grp-1",
		Urn: urn("CustomClaims"),
		State: property.NewMap(map[string]property.Value{
			"ownerType": property.New("user-group"),
			"ownerId":   property.New("grp-1"),
		}),
		Inputs: property.NewMap(map[string]property.Value{
			"ownerType": property.New("user-group"),
			"ownerId":   property.New("grp-2"),
		}),
	})
	require.NoError(t, err)
	assert.True(t, diffResp.HasChanges)
	assert.True(t, diffResp.DeleteBeforeReplace)
	assert.Equal(t, p.UpdateReplace, diffResp.DetailedDiff["ownerId"].Kind)
}

func TestCustomClaimsDiffNoChange(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	claims := property.NewArray([]property.Value{
		property.New(map[string]property.Value{
			"key":   property.New("k"),
			"value": property.New("v"),
		}),
	})
	diffResp, err := prov.Diff(p.DiffRequest{
		ID:  "user-group/grp-1",
		Urn: urn("CustomClaims"),
		State: property.NewMap(map[string]property.Value{
			"ownerType": property.New("user-group"),
			"ownerId":   property.New("grp-1"),
			"claims":    property.New(claims),
		}),
		Inputs: property.NewMap(map[string]property.Value{
			"ownerType": property.New("user-group"),
			"ownerId":   property.New("grp-1"),
			"claims":    property.New(claims),
		}),
	})
	require.NoError(t, err)
	assert.False(t, diffResp.HasChanges)
}

func TestCustomClaimsInvalidOwnerType(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	_, err := prov.Create(p.CreateRequest{
		Urn: urn("CustomClaims"),
		Properties: property.NewMap(map[string]property.Value{
			"ownerType": property.New("application"),
			"ownerId":   property.New("x"),
		}),
	})
	require.Error(t, err)
}

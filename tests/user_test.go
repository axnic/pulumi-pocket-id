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

func TestUserLifecycle(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	// Create
	createResp, err := prov.Create(p.CreateRequest{
		Urn: urn("User"),
		Properties: property.NewMap(map[string]property.Value{
			"username":    property.New("alice"),
			"firstName":   property.New("Alice"),
			"displayName": property.New("Alice Liddell"),
		}),
	})
	require.NoError(t, err)
	id := createResp.ID
	assert.Contains(t, id, "usr-")
	assert.Equal(t, "alice", createResp.Properties.Get("username").AsString())
	assert.Equal(t, "Alice", createResp.Properties.Get("firstName").AsString())

	// Read
	readResp, err := prov.Read(p.ReadRequest{
		ID:  id,
		Urn: urn("User"),
		Properties: property.NewMap(map[string]property.Value{
			"username": property.New("alice"),
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, "alice", readResp.Properties.Get("username").AsString())

	// Update
	updateResp, err := prov.Update(p.UpdateRequest{
		ID:  id,
		Urn: urn("User"),
		State: property.NewMap(map[string]property.Value{
			"username":    property.New("alice"),
			"firstName":   property.New("Alice"),
			"displayName": property.New("Alice Liddell"),
		}),
		Inputs: property.NewMap(map[string]property.Value{
			"username":    property.New("alice"),
			"firstName":   property.New("Alice"),
			"displayName": property.New("Alice in Wonderland"),
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, "Alice in Wonderland", updateResp.Properties.Get("displayName").AsString())

	// Delete
	err = prov.Delete(p.DeleteRequest{
		ID:         id,
		Urn:        urn("User"),
		Properties: createResp.Properties,
	})
	require.NoError(t, err)
}

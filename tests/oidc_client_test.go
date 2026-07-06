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

func TestOidcClientLifecycle(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	// Create with an explicit client id
	createResp, err := prov.Create(p.CreateRequest{
		Urn: urn("OidcClient"),
		Properties: property.NewMap(map[string]property.Value{
			"clientId": property.New("my-app"),
			"name":     property.New("My App"),
			"callbackUrls": property.New(property.NewArray([]property.Value{
				property.New("https://app.example.com/callback"),
			})),
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, "my-app", createResp.ID)
	assert.Equal(t, "My App", createResp.Properties.Get("name").AsString())
	assert.True(t, createResp.Properties.Get("pkceSupported").AsBool())

	// Read
	readResp, err := prov.Read(p.ReadRequest{
		ID:  "my-app",
		Urn: urn("OidcClient"),
		Properties: property.NewMap(map[string]property.Value{
			"name": property.New("My App"),
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, "My App", readResp.Properties.Get("name").AsString())

	// Update
	updateResp, err := prov.Update(p.UpdateRequest{
		ID:  "my-app",
		Urn: urn("OidcClient"),
		State: property.NewMap(map[string]property.Value{
			"clientId": property.New("my-app"),
			"name":     property.New("My App"),
		}),
		Inputs: property.NewMap(map[string]property.Value{
			"clientId": property.New("my-app"),
			"name":     property.New("My Application"),
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, "My Application", updateResp.Properties.Get("name").AsString())

	// Delete
	err = prov.Delete(p.DeleteRequest{
		ID:         "my-app",
		Urn:        urn("OidcClient"),
		Properties: createResp.Properties,
	})
	require.NoError(t, err)
}

func TestOidcClientGeneratedID(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	// Create without a client id; the server generates one.
	createResp, err := prov.Create(p.CreateRequest{
		Urn: urn("OidcClient"),
		Properties: property.NewMap(map[string]property.Value{
			"name": property.New("Generated"),
		}),
	})
	require.NoError(t, err)
	assert.Contains(t, createResp.ID, "cli-")
}

func TestOidcClientDryRun(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	// A dry-run create must not contact the server.
	createResp, err := prov.Create(p.CreateRequest{
		Urn:    urn("OidcClient"),
		Properties: property.NewMap(map[string]property.Value{"name": property.New("Preview")}),
		DryRun: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, createResp.ID)
}

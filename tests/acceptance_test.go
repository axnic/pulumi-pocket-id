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
	"fmt"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// Acceptance tests run against a real Pocket-ID instance. They are skipped unless both
// POCKET_ID_BASE_URL and POCKET_ID_API_KEY are set. Start an instance with:
//
//	docker compose -f docker-compose.test.yml up -d
//
// then run:
//
//	POCKET_ID_BASE_URL=http://localhost:1411 \
//	POCKET_ID_API_KEY=pulumi-test-api-key-at-least-16-chars \
//	go test ./tests/... -run TestAcceptance -count=1 -v

func acceptanceServer(t *testing.T) integration.Server {
	t.Helper()
	baseURL := os.Getenv("POCKET_ID_BASE_URL")
	apiKey := os.Getenv("POCKET_ID_API_KEY")
	if baseURL == "" || apiKey == "" {
		t.Skipf("POCKET_ID_BASE_URL or POCKET_ID_API_KEY not set; skipping acceptance test")
	}
	prov := testServer(t)
	configure(t, prov, baseURL, apiKey)
	return prov
}

var nameCounter uint64

func unique(prefix, typ string) string {
	n := atomic.AddUint64(&nameCounter, 1)
	return fmt.Sprintf("%s-%s-%d", prefix, typ, n)
}

func TestAcceptanceUserGroupLifecycle(t *testing.T) {
	prov := acceptanceServer(t)
	name := unique("group", "acc")

	createResp, err := prov.Create(p.CreateRequest{
		Urn: urn("UserGroup"),
		Properties: property.NewMap(map[string]property.Value{
			"friendlyName": property.New("Acceptance " + name),
			"name":         property.New(name),
		}),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = prov.Delete(p.DeleteRequest{ID: createResp.ID, Urn: urn("UserGroup"), Properties: createResp.Properties})
	})
	assert.NotEmpty(t, createResp.ID)
	assert.Equal(t, "Acceptance "+name, createResp.Properties.Get("friendlyName").AsString())

	// Read
	readResp, err := prov.Read(p.ReadRequest{
		ID:         createResp.ID,
		Urn:        urn("UserGroup"),
		Properties: createResp.Properties,
	})
	require.NoError(t, err)
	assert.Equal(t, name, readResp.Properties.Get("name").AsString())

	// Update
	updated := "Acceptance Updated " + name
	updateResp, err := prov.Update(p.UpdateRequest{
		ID:         createResp.ID,
		Urn:        urn("UserGroup"),
		State:      createResp.Properties,
		Inputs: property.NewMap(map[string]property.Value{
			"friendlyName": property.New(updated),
			"name":         property.New(name),
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, updated, updateResp.Properties.Get("friendlyName").AsString())

	// Delete
	err = prov.Delete(p.DeleteRequest{
		ID:         createResp.ID,
		Urn:        urn("UserGroup"),
		Properties: updateResp.Properties,
	})
	require.NoError(t, err)
}

func TestAcceptanceUserLifecycle(t *testing.T) {
	prov := acceptanceServer(t)
	username := unique("user", "acc")

	createResp, err := prov.Create(p.CreateRequest{
		Urn: urn("User"),
		Properties: property.NewMap(map[string]property.Value{
			"username":  property.New(username),
			"firstName": property.New("Acceptance"),
		}),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = prov.Delete(p.DeleteRequest{ID: createResp.ID, Urn: urn("User"), Properties: createResp.Properties})
	})
	assert.NotEmpty(t, createResp.ID)

	readResp, err := prov.Read(p.ReadRequest{
		ID:         createResp.ID,
		Urn:        urn("User"),
		Properties: createResp.Properties,
	})
	require.NoError(t, err)
	assert.Equal(t, username, readResp.Properties.Get("username").AsString())

	err = prov.Delete(p.DeleteRequest{
		ID:         createResp.ID,
		Urn:        urn("User"),
		Properties: createResp.Properties,
	})
	require.NoError(t, err)
}

func TestAcceptanceOidcClientLifecycle(t *testing.T) {
	prov := acceptanceServer(t)
	clientID := unique("client", "acc")

	createResp, err := prov.Create(p.CreateRequest{
		Urn: urn("OidcClient"),
		Properties: property.NewMap(map[string]property.Value{
			"clientId": property.New(clientID),
			"name":     property.New("Acceptance Client " + clientID),
		}),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = prov.Delete(p.DeleteRequest{ID: clientID, Urn: urn("OidcClient"), Properties: createResp.Properties})
	})
	assert.Equal(t, clientID, createResp.ID)

	readResp, err := prov.Read(p.ReadRequest{
		ID:         clientID,
		Urn:        urn("OidcClient"),
		Properties: createResp.Properties,
	})
	require.NoError(t, err)
	assert.Equal(t, "Acceptance Client "+clientID, readResp.Properties.Get("name").AsString())

	err = prov.Delete(p.DeleteRequest{
		ID:         clientID,
		Urn:        urn("OidcClient"),
		Properties: createResp.Properties,
	})
	require.NoError(t, err)
}

func TestAcceptanceCustomClaimsLifecycle(t *testing.T) {
	prov := acceptanceServer(t)

	// Create a user-group to own the claims.
	groupName := unique("claims-group", "acc")
	grp, err := prov.Create(p.CreateRequest{
		Urn: urn("UserGroup"),
		Properties: property.NewMap(map[string]property.Value{
			"friendlyName": property.New("Claims Group " + groupName),
			"name":         property.New(groupName),
		}),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = prov.Delete(p.DeleteRequest{ID: grp.ID, Urn: urn("UserGroup"), Properties: grp.Properties})
	})

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

	readResp, err := prov.Read(p.ReadRequest{
		ID:         createResp.ID,
		Urn:        urn("CustomClaims"),
		Properties: createResp.Properties,
	})
	require.NoError(t, err)
	arr := readResp.Properties.Get("claims").AsArray()
	require.Equal(t, 1, arr.Len())
	assert.Equal(t, "department", arr.Get(0).AsMap().Get("key").AsString())

	err = prov.Delete(p.DeleteRequest{
		ID:         createResp.ID,
		Urn:        urn("CustomClaims"),
		Properties: createResp.Properties,
	})
	require.NoError(t, err)
}

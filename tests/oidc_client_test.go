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
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

func oidcStrs(s ...string) property.Value {
	vals := make([]property.Value, 0, len(s))
	for _, v := range s {
		vals = append(vals, property.New(v))
	}
	return property.New(property.NewArray(vals))
}

func oidcMap(kv map[string]property.Value) property.Map { return property.NewMap(kv) }

func oidcCount(fake *fakePocketID, store string) int {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return len(fake.store(store))
}

func TestOidcClientLifecycle(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	inputs := oidcMap(map[string]property.Value{
		keyClientID:     property.New("my-app"),
		keyName:         property.New("My App"),
		keyCallbackURLs: oidcStrs("https://app.example.com/callback"),
		"launchUrl":     property.New("https://app.example.com"),
		"pkceEnabled":   property.New(true),
		"credentials": property.New(oidcMap(map[string]property.Value{
			"federatedIdentities": property.New(property.NewArray([]property.Value{
				property.New(oidcMap(map[string]property.Value{
					"issuer": property.New("https://issuer.example.com"), "subject": property.New("sub"),
				})),
			})),
		})),
		keyAllowedUserGroupIDs: oidcStrs("g1", "g2"),
	})

	created, err := prov.Create(p.CreateRequest{Urn: urn("OidcClient"), Properties: inputs})
	require.NoError(t, err)
	assert.Equal(t, "my-app", created.ID)
	assert.Equal(t, "My App", created.Properties.Get(keyName).AsString())
	assert.True(t, created.Properties.Get("pkceSupported").AsBool())
	assert.True(t, created.Properties.Get("isGroupRestricted").AsBool())
	assert.Equal(t, "confidential", created.Properties.Get("clientType").AsString())
	assert.Equal(t, 2, created.Properties.Get(keyAllowedUserGroupIDs).AsArray().Len())
	// The auto-generated secret was removed: the client owns no secret.
	assert.Equal(t, 0, oidcCount(fake, "oidcSecrets"))

	read, err := prov.Read(p.ReadRequest{ID: created.ID, Urn: urn("OidcClient"), Properties: created.Properties,
		Inputs: inputs})
	require.NoError(t, err)
	assert.Equal(t, "my-app", read.ID)
	assert.Equal(t, "https://app.example.com", read.Properties.Get("launchUrl").AsString())
	fi := read.Properties.Get("credentials").AsMap().Get("federatedIdentities").AsArray().Get(0).AsMap()
	assert.Equal(t, "https://issuer.example.com", fi.Get("issuer").AsString())
	assert.Equal(t, "sub", fi.Get("subject").AsString())

	// Update: new name, groups shrink to one, then the group list is emptied.
	updated, err := prov.Update(p.UpdateRequest{
		ID: created.ID, Urn: urn("OidcClient"), State: created.Properties,
		Inputs: inputs.Set(keyName, property.New("My Application")).Set(keyAllowedUserGroupIDs, oidcStrs("g2")),
	})
	require.NoError(t, err)
	assert.Equal(t, "My Application", updated.Properties.Get(keyName).AsString())
	assert.Equal(t, "g2", updated.Properties.Get(keyAllowedUserGroupIDs).AsArray().Get(0).AsString())

	updated, err = prov.Update(p.UpdateRequest{
		ID: created.ID, Urn: urn("OidcClient"), State: updated.Properties,
		Inputs: inputs.Delete(keyAllowedUserGroupIDs).Delete("credentials"),
	})
	require.NoError(t, err)
	assert.False(t, updated.Properties.Get("isGroupRestricted").AsBool())
	groupIDs := updated.Properties.Get(keyAllowedUserGroupIDs)
	assert.False(t, groupIDs.IsArray() && groupIDs.AsArray().Len() > 0)

	require.NoError(t, prov.Delete(p.DeleteRequest{ID: created.ID, Urn: urn("OidcClient"),
		Properties: updated.Properties}))
	assert.Equal(t, 0, oidcCount(fake, "oidcClients"))
	// Delete is idempotent.
	require.NoError(t, prov.Delete(p.DeleteRequest{ID: created.ID, Urn: urn("OidcClient"),
		Properties: updated.Properties}))
}

func TestOidcClientGeneratedIDAndPublicClient(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	created, err := prov.Create(p.CreateRequest{Urn: urn("OidcClient"), Properties: oidcMap(map[string]property.Value{
		keyName: property.New("Gen"), keyIsPublic: property.New(true),
	})})
	require.NoError(t, err)
	assert.Contains(t, created.ID, "oidc-")
	assert.Equal(t, "public", created.Properties.Get("clientType").AsString())
	assert.False(t, created.Properties.Get(keyClientID).IsString(), "clientId stays unset when not provided")
}

func TestOidcClientReadGoneAndImport(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	gone, err := prov.Read(p.ReadRequest{ID: "missing", Urn: urn("OidcClient"), Properties: oidcMap(nil)})
	require.NoError(t, err)
	assert.Empty(t, gone.ID)

	created, err := prov.Create(p.CreateRequest{Urn: urn("OidcClient"), Properties: oidcMap(map[string]property.Value{
		keyClientID: property.New("imp"), keyName: property.New("Imported"), keyAllowedUserGroupIDs: oidcStrs("g1"),
	})})
	require.NoError(t, err)

	// Import: Read from the ID alone.
	imp, err := prov.Read(p.ReadRequest{ID: created.ID, Urn: urn("OidcClient")})
	require.NoError(t, err)
	assert.Equal(t, "imp", imp.ID)
	assert.Equal(t, "Imported", imp.Properties.Get(keyName).AsString())
	assert.Equal(t, "imp", imp.Inputs.Get(keyClientID).AsString())
	assert.Equal(t, "g1", imp.Inputs.Get(keyAllowedUserGroupIDs).AsArray().Get(0).AsString())
}

func TestOidcClientPreviewMakesNoAPICall(t *testing.T) {
	t.Parallel()
	prov := testServer(t)
	configure(t, prov, "http://127.0.0.1:1", "k") // unreachable: any API call would fail

	created, err := prov.Create(p.CreateRequest{
		Urn: urn("OidcClient"), DryRun: true,
		Properties: oidcMap(map[string]property.Value{keyClientID: property.New("pv"), keyName: property.New("Preview")}),
	})
	require.NoError(t, err)
	assert.Equal(t, "pv", created.ID)

	updated, err := prov.Update(p.UpdateRequest{
		ID: "pv", Urn: urn("OidcClient"), DryRun: true, State: created.Properties,
		Inputs: oidcMap(map[string]property.Value{keyClientID: property.New("pv"), keyName: property.New("Renamed")}),
	})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", updated.Properties.Get(keyName).AsString())
}

func TestOidcClientDiffReplacesOnClientID(t *testing.T) {
	t.Parallel()
	prov := testServer(t)
	state := oidcMap(map[string]property.Value{
		keyClientID: property.New("a"), keyName: property.New("N"),
		"clientType": property.New("confidential"), "pkceSupported": property.New(true),
		"hasLogo": property.New(false), "hasDarkLogo": property.New(false), "isGroupRestricted": property.New(false),
	})

	resp, err := prov.Diff(p.DiffRequest{ID: "a", Urn: urn("OidcClient"), State: state,
		Inputs: oidcMap(map[string]property.Value{keyClientID: property.New("b"), keyName: property.New("N")})})
	require.NoError(t, err)
	assert.Equal(t, p.UpdateReplace, resp.DetailedDiff[keyClientID].Kind)

	resp, err = prov.Diff(p.DiffRequest{ID: "a", Urn: urn("OidcClient"), State: state,
		Inputs: oidcMap(map[string]property.Value{keyClientID: property.New("a"), keyName: property.New("M")})})
	require.NoError(t, err)
	assert.Equal(t, p.Update, resp.DetailedDiff[keyName].Kind)
}

func TestGetOidcClientFunction(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	_, err := prov.Create(p.CreateRequest{Urn: urn("OidcClient"), Properties: oidcMap(map[string]property.Value{
		keyClientID: property.New("fn"), keyName: property.New("Fn"), keyCallbackURLs: oidcStrs("https://x/cb"),
		keyAllowedUserGroupIDs: oidcStrs("g1"),
	})})
	require.NoError(t, err)

	resp, err := prov.Invoke(p.InvokeRequest{
		Token: tokens.Type("pocket-id:index:getOidcClient"),
		Args:  oidcMap(map[string]property.Value{keyClientID: property.New("fn")}),
	})
	require.NoError(t, err)
	assert.Equal(t, "Fn", resp.Return.Get(keyName).AsString())
	assert.Equal(t, "https://x/cb", resp.Return.Get(keyCallbackURLs).AsArray().Get(0).AsString())
	assert.Equal(t, "g1", resp.Return.Get(keyAllowedUserGroupIDs).AsArray().Get(0).AsString())
	_, hasSecret := resp.Return.GetOk("secret")
	assert.False(t, hasSecret)

	_, err = prov.Invoke(p.InvokeRequest{
		Token: tokens.Type("pocket-id:index:getOidcClient"),
		Args:  oidcMap(map[string]property.Value{keyClientID: property.New("nope")}),
	})
	require.Error(t, err)
}

// Pocket-ID answers 400 to an optional URL sent as "": unset URLs must be omitted from the request, and a
// refresh must not turn the server's null/"" defaults into inputs.
func TestOidcClientOptionalURLsAreOmittedAndStayDriftFree(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	inputs := oidcMap(map[string]property.Value{
		keyName: property.New("Bare"), keyCallbackURLs: oidcStrs("https://a.example.com/cb"),
	})
	created, err := prov.Create(p.CreateRequest{Urn: urn("OidcClient"), Properties: inputs})
	require.NoError(t, err)
	assert.True(t, created.Properties.Get("launchUrl").IsNull())
	assert.True(t, created.Properties.Get("backchannelLogoutUrl").IsNull())

	for range 2 { // a second refresh changes nothing either
		read, err := prov.Read(p.ReadRequest{
			ID: created.ID, Urn: urn("OidcClient"), Properties: created.Properties, Inputs: inputs,
		})
		require.NoError(t, err)
		assert.True(t, read.Inputs.Get("launchUrl").IsNull())
		assert.True(t, read.Inputs.Get("backchannelLogoutUrl").IsNull())
		assert.True(t, read.Inputs.Get(keyLogoURL).IsNull())
		created.Properties = read.Properties
	}
}

// Pocket-ID keeps the logo when an update omits the logo URL: dropping logoUrl from the program deletes it.
func TestOidcClientDroppedLogoURLDeletesLogo(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	inputs := oidcMap(map[string]property.Value{
		keyName: property.New("Logo"), keyCallbackURLs: oidcStrs("https://a.example.com/cb"),
		keyLogoURL: property.New("https://img.example.com/logo.png"),
	})
	created, err := prov.Create(p.CreateRequest{Urn: urn("OidcClient"), Properties: inputs})
	require.NoError(t, err)
	assert.True(t, created.Properties.Get("hasLogo").AsBool())
	assert.NotNil(t, fake.fakeImage("oidc/"+created.ID+"/light"))

	// Changing the URL keeps a logo; dropping it removes the logo.
	withDark := inputs.Set("darkLogoUrl", property.New("https://img.example.com/dark.png"))
	updated, err := prov.Update(p.UpdateRequest{
		ID: created.ID, Urn: urn("OidcClient"), State: created.Properties, Inputs: withDark,
	})
	require.NoError(t, err)
	assert.True(t, updated.Properties.Get("hasDarkLogo").AsBool())

	bare := inputs.Delete(keyLogoURL)
	updated, err = prov.Update(p.UpdateRequest{
		ID: created.ID, Urn: urn("OidcClient"), State: updated.Properties, Inputs: bare,
	})
	require.NoError(t, err)
	assert.False(t, updated.Properties.Get("hasLogo").AsBool())
	assert.False(t, updated.Properties.Get("hasDarkLogo").AsBool())
	assert.Nil(t, fake.fakeImage("oidc/"+created.ID+"/light"))
	assert.Nil(t, fake.fakeImage("oidc/"+created.ID+"/dark"))
}

func TestOidcClientBackchannelLogoutRequiresRecentServer(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	fake.legacyOIDC = true
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	// Without the field, a legacy server is fine.
	_, err := prov.Create(p.CreateRequest{Urn: urn("OidcClient"), Properties: oidcMap(map[string]property.Value{
		keyClientID: property.New("plain"), keyName: property.New("Plain"),
		keyCallbackURLs: oidcStrs("https://app.example.com/callback"),
	})})
	require.NoError(t, err)

	// With it, the silent no-op becomes an explicit error and no half-configured client is left behind.
	_, err = prov.Create(p.CreateRequest{Urn: urn("OidcClient"), Properties: oidcMap(map[string]property.Value{
		keyClientID: property.New("legacy"), keyName: property.New("Legacy"),
		keyCallbackURLs:        oidcStrs("https://app.example.com/callback"),
		"backchannelLogoutUrl": property.New("https://app.example.com/bc"),
	})})
	require.ErrorContains(t, err, "requires v2.17.0")
	assert.Equal(t, 1, oidcCount(fake, "oidcClients"))
}

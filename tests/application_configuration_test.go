// Copyright 2025, axnic.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

func appConfigInputs(kv map[string]property.Value) property.Map { return property.NewMap(kv) }

func TestApplicationConfigurationLifecycle(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)
	cfg := func() map[string]any { return fake.store("appconfig")["all"] }

	inputs := appConfigInputs(map[string]property.Value{
		keyAppName:           property.New("Acme ID"),
		keySessionDuration:   property.New(120.0),
		keyDisableAnimations: property.New(true),
		"smtpHost":           property.New("smtp.example.com"),
		"smtpPort":           property.New(465.0),
		"smtpPassword":       property.New("s3cret").WithSecret(true),
	})

	// Create: values go out as strings, untouched keys are preserved.
	created, err := prov.Create(p.CreateRequest{Urn: urn("ApplicationConfiguration"), Properties: inputs})
	require.NoError(t, err)
	assert.Equal(t, "app-config", created.ID)
	assert.Equal(t, "Acme ID", cfg()[keyAppName])
	assert.Equal(t, "120", cfg()[keySessionDuration])
	assert.Equal(t, keyTrue, cfg()[keyDisableAnimations])
	assert.Equal(t, "465", cfg()["smtpPort"])
	assert.Equal(t, "s3cret", cfg()["smtpPassword"])
	assert.Equal(t, keyDisabled, cfg()[keyAllowUserSignups])
	assert.Equal(t, "fake-instance", cfg()["instanceId"])

	// Read: only managed fields come back; the secret stays from state.
	read, err := prov.Read(p.ReadRequest{
		ID: created.ID, Urn: urn("ApplicationConfiguration"),
		Inputs: inputs, Properties: created.Properties,
	})
	require.NoError(t, err)
	assert.Equal(t, "Acme ID", read.Properties.Get(keyAppName).AsString())
	assert.Equal(t, 120.0, read.Properties.Get(keySessionDuration).AsNumber())
	assert.True(t, read.Properties.Get(keyDisableAnimations).AsBool())
	assert.Equal(t, "s3cret", read.Properties.Get("smtpPassword").AsString())
	assert.True(t, read.Properties.Get("homePageUrl").IsNull(), "unmanaged fields are not read back")

	// Drift on the server shows up in the next read.
	cfg()[keyAppName] = "Changed outside"
	read, err = prov.Read(p.ReadRequest{ID: created.ID, Urn: urn("ApplicationConfiguration"), Inputs: inputs,
		Properties: created.Properties})
	require.NoError(t, err)
	assert.Equal(t, "Changed outside", read.Properties.Get(keyAppName).AsString())

	// Update.
	updatedInputs := appConfigInputs(map[string]property.Value{
		keyAppName:          property.New("Acme ID v2"),
		keyAllowUserSignups: property.New("withToken"),
	})
	updated, err := prov.Update(p.UpdateRequest{
		ID: created.ID, Urn: urn("ApplicationConfiguration"), State: created.Properties, Inputs: updatedInputs,
	})
	require.NoError(t, err)
	assert.Equal(t, "Acme ID v2", updated.Properties.Get(keyAppName).AsString())
	assert.Equal(t, "Acme ID v2", cfg()[keyAppName])
	assert.Equal(t, "withToken", cfg()[keyAllowUserSignups])
	assert.Equal(t, "465", cfg()["smtpPort"], "fields dropped from the program keep their server value")

	// Delete is a no-op.
	require.NoError(t, prov.Delete(p.DeleteRequest{ID: created.ID, Urn: urn("ApplicationConfiguration"),
		Properties: updated.Properties}))
	assert.Equal(t, "Acme ID v2", cfg()[keyAppName])
}

func TestApplicationConfigurationImport(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	read, err := prov.Read(p.ReadRequest{ID: "app-config", Urn: urn("ApplicationConfiguration"),
		Properties: property.NewMap(nil)})
	require.NoError(t, err)
	assert.Equal(t, "app-config", read.ID)
	assert.Equal(t, "Pocket ID", read.Properties.Get(keyAppName).AsString())
	assert.Equal(t, 60.0, read.Properties.Get(keySessionDuration).AsNumber())
	assert.True(t, read.Properties.Get("allowOwnAccountEdit").AsBool())
	assert.True(t, read.Properties.Get("smtpPassword").IsNull(), "secrets are not read back")
}

func TestApplicationConfigurationPreviewMakesNoCall(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	resp, err := prov.Create(p.CreateRequest{
		Urn: urn("ApplicationConfiguration"), DryRun: true,
		Properties: appConfigInputs(map[string]property.Value{keyAppName: property.New("Preview")}),
	})
	require.NoError(t, err)
	assert.Equal(t, "app-config", resp.ID)
	assert.Equal(t, "Pocket ID", fake.store("appconfig")["all"][keyAppName])
}

func TestGetOpenIdConfigurationFunction(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	resp, err := prov.Invoke(p.InvokeRequest{ //nolint:gosec // not a credential
		Token: "pocket-id:index:getOpenIdConfiguration", Args: property.NewMap(nil),
	})
	require.NoError(t, err)
	assert.Equal(t, "https://id.example.com", resp.Return.Get("issuer").AsString())
	assert.Equal(t, "https://id.example.com/.well-known/jwks.json", resp.Return.Get("jwksUri").AsString())
	assert.Equal(t, "https://id.example.com/authorize", resp.Return.Get("authorizationEndpoint").AsString())
	assert.Equal(t, 3, resp.Return.Get("scopesSupported").AsArray().Len())
	assert.True(t, resp.Return.Get("claimsSupported").IsArray(), "missing lists are empty, not null")
}

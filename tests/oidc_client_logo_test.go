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
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/archive"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

const (
	keyLogo        = "logo"
	keyDarkLogo    = "darkLogo"
	keyLogoSHA     = "logoSha256"
	keyDarkSHA     = "darkLogoSha256"
	keyDarkLogoURL = "darkLogoUrl"
)

// asAsset turns the wire form of an asset (what the in-process harness hands back) into an asset property.
func asAsset(v property.Value) property.Value {
	if !v.IsMap() {
		return v
	}
	a, ok, err := resource.DeserializeAsset(resource.ToResourcePropertyValue(v).ObjectValue().Mappable())
	if err != nil || !ok {
		return v
	}
	return property.New(a)
}

// retypeLogos re-types the logo assets of a returned OidcClient state, like the real engine does.
func retypeLogos(m property.Map) property.Map {
	for _, k := range []string{keyLogo, keyDarkLogo} {
		if v, ok := m.GetOk(k); ok {
			m = m.Set(k, asAsset(v))
		}
	}
	return m
}

type logoEnv struct {
	imgRes
	fake *fakePocketID
}

func newLogoEnv(t *testing.T) logoEnv {
	t.Helper()
	fake, prov := imgEnv(t)
	return logoEnv{imgRes: imgRes{t: t, prov: prov, urn: urn("OidcClient")}, fake: fake}
}

func (e logoEnv) dropFakeImage(key string) {
	e.fake.mu.Lock()
	defer e.fake.mu.Unlock()
	delete(e.fake.store("images"), key)
}

// checked runs the inputs through Check, as the engine does before every other call: it fills the defaults
// of the optional booleans, which would otherwise show up as a diff against the state.
func (e logoEnv) checked(in property.Map) property.Map {
	e.t.Helper()
	resp, err := e.prov.Check(p.CheckRequest{Urn: e.urn, Inputs: in})
	require.NoError(e.t, err)
	require.Empty(e.t, resp.Failures)
	return retypeLogos(resp.Inputs)
}

func logoClientInputs(extra map[string]property.Value) property.Map {
	in := map[string]property.Value{
		keyClientID: property.New("app"), keyName: property.New("App"),
		keyCallbackURLs: oidcStrs("https://app.example.com/cb"),
	}
	for k, v := range extra {
		in[k] = v
	}
	return property.NewMap(in)
}

func TestOidcClientUploadedLogoLifecycle(t *testing.T) {
	t.Parallel()
	e := newLogoEnv(t)
	light, dark := imgAsset(t, "logo.png", imgPNG), imgAsset(t, "dark.svg", imgSVG)
	in := e.checked(logoClientInputs(map[string]property.Value{keyLogo: light, keyDarkLogo: dark}))

	created := e.create(in, false)
	created.Properties = retypeLogos(created.Properties)
	assert.Equal(t, sum(imgPNG), created.Properties.Get(keyLogoSHA).AsString())
	assert.Equal(t, sum(imgSVG), created.Properties.Get(keyDarkSHA).AsString())
	assert.True(t, created.Properties.Get("hasLogo").AsBool())
	assert.True(t, created.Properties.Get("hasDarkLogo").AsBool())
	stored := e.fake.fakeImage("oidc/app/light")
	require.NotNil(t, stored)
	assert.Equal(t, imgPNG, stored[imgData])
	assert.Equal(t, "logo.png", stored[imgFilename])
	assert.Equal(t, "image/png", stored[imgContentType])
	assert.Equal(t, imgSVG, e.fake.fakeImage("oidc/app/dark")[imgData])

	// Idempotence: a refresh and a diff on unchanged inputs report nothing, twice.
	state := created.Properties
	for range 2 {
		read := e.read(created.ID, state, in)
		state = retypeLogos(read.Properties)
		assert.Equal(t, sum(imgPNG), state.Get(keyLogoSHA).AsString())
		assert.Equal(t, sum(imgSVG), state.Get(keyDarkSHA).AsString())
		assert.Empty(t, e.diff(created.ID, state, in).DetailedDiff)
		assert.False(t, e.diff(created.ID, retypeLogos(read.Inputs), in).HasChanges)
	}

	// A new file is an update that uploads it; the dark logo is not touched.
	light2 := imgAsset(t, "new.png", imgPNG2)
	in2 := in.Set(keyLogo, light2)
	d := e.diff(created.ID, state, in2)
	assert.Equal(t, p.Update, d.DetailedDiff[keyLogo].Kind)
	assert.NotContains(t, d.DetailedDiff, keyDarkLogo)
	upd := e.update(created.ID, state, in2)
	upd.Properties = retypeLogos(upd.Properties)
	assert.Equal(t, sum(imgPNG2), upd.Properties.Get(keyLogoSHA).AsString())
	assert.Equal(t, imgPNG2, e.fake.fakeImage("oidc/app/light")[imgData])
	assert.Equal(t, "new.png", e.fake.fakeImage("oidc/app/light")[imgFilename])
	assert.False(t, e.diff(created.ID, upd.Properties, in2).HasChanges)

	// An update that does not concern the logos does not upload them again.
	e.fake.setFakeImage("oidc/app/dark", imgPNG)
	in3 := in2.Set(keyName, property.New("Renamed"))
	renamed := e.update(created.ID, upd.Properties, in3)
	assert.Equal(t, imgPNG, e.fake.fakeImage("oidc/app/dark")[imgData], "dark logo left alone")
	assert.Equal(t, "Renamed", renamed.Properties.Get(keyName).AsString())
	assert.True(t, renamed.Properties.Get("hasDarkLogo").AsBool(), "a plain update keeps the logos")
	e.fake.setFakeImage("oidc/app/dark", imgSVG)

	// Removing logo from the program deletes the light logo only.
	in4 := in3.Delete(keyLogo)
	assert.Equal(t, p.Delete, e.diff(created.ID, renamed.Properties, in4).DetailedDiff[keyLogo].Kind)
	removed := e.update(created.ID, retypeLogos(renamed.Properties), in4)
	assert.Nil(t, e.fake.fakeImage("oidc/app/light"))
	assert.NotNil(t, e.fake.fakeImage("oidc/app/dark"))
	assert.False(t, removed.Properties.Get("hasLogo").AsBool())
	assert.True(t, removed.Properties.Get("hasDarkLogo").AsBool())
	assert.True(t, removed.Properties.Get(keyLogoSHA).IsNull())
	assert.True(t, removed.Properties.Get(keyLogo).IsNull())
	assert.Equal(t, sum(imgSVG), removed.Properties.Get(keyDarkSHA).AsString())

	// ... and the dark one when it goes too; removing again is harmless.
	in5 := in4.Delete(keyDarkLogo)
	gone := e.update(created.ID, retypeLogos(removed.Properties), in5)
	assert.Nil(t, e.fake.fakeImage("oidc/app/dark"))
	assert.False(t, gone.Properties.Get("hasDarkLogo").AsBool())
	assert.False(t, e.diff(created.ID, gone.Properties, in5).HasChanges)
}

func TestOidcClientUploadedLogoDriftIsDetectedAndRepaired(t *testing.T) {
	t.Parallel()
	e := newLogoEnv(t)
	img := imgAsset(t, "logo.png", imgPNG)
	in := e.checked(logoClientInputs(map[string]property.Value{keyLogo: img}))
	created := e.create(in, false)

	// The served bytes change out of band: the refresh records their digest and the diff re-uploads.
	e.fake.setFakeImage("oidc/app/light", imgPNG2)
	read := e.read(created.ID, retypeLogos(created.Properties), in)
	state := retypeLogos(read.Properties)
	assert.Equal(t, sum(imgPNG2), state.Get(keyLogoSHA).AsString())
	assert.Equal(t, p.Update, e.diff(created.ID, state, in).DetailedDiff[keyLogo].Kind)
	upd := e.update(created.ID, state, in)
	assert.Equal(t, imgPNG, e.fake.fakeImage("oidc/app/light")[imgData])
	assert.Equal(t, sum(imgPNG), upd.Properties.Get(keyLogoSHA).AsString())

	// Once repaired, nothing is reported any more.
	read = e.read(created.ID, retypeLogos(upd.Properties), in)
	assert.False(t, e.diff(created.ID, retypeLogos(read.Properties), in).HasChanges)
}

func TestOidcClientUploadedLogoMissingOnServerIsReuploaded(t *testing.T) {
	t.Parallel()
	e := newLogoEnv(t)
	img := imgAsset(t, "logo.png", imgPNG)
	in := e.checked(logoClientInputs(map[string]property.Value{keyLogo: img}))
	created := e.create(in, false)

	e.dropFakeImage("oidc/app/light") // GET answers 404
	read := e.read(created.ID, retypeLogos(created.Properties), in)
	assert.Equal(t, created.ID, read.ID, "the client itself is still there")
	state := retypeLogos(read.Properties)
	assert.True(t, state.Get(keyLogo).IsNull())
	assert.True(t, state.Get(keyLogoSHA).IsNull())
	assert.False(t, state.Get("hasLogo").AsBool())
	assert.Equal(t, p.Add, e.diff(created.ID, state, in).DetailedDiff[keyLogo].Kind)

	upd := e.update(created.ID, state, in)
	assert.Equal(t, imgPNG, e.fake.fakeImage("oidc/app/light")[imgData])
	assert.Equal(t, sum(imgPNG), upd.Properties.Get(keyLogoSHA).AsString())

	// A logo that was never managed as a file is not looked at.
	bare := e.create(logoClientInputs(map[string]property.Value{keyClientID: property.New("bare")}), false)
	e.fake.setFakeImage("oidc/bare/light", imgPNG2)
	read = e.read(bare.ID, bare.Properties, logoClientInputs(map[string]property.Value{keyClientID: property.New("bare")}))
	assert.True(t, read.Properties.Get(keyLogoSHA).IsNull())
}

func TestOidcClientUploadedLogoAndLogoURLAreExclusive(t *testing.T) {
	t.Parallel()
	e := newLogoEnv(t)
	img := imgAsset(t, "logo.png", imgPNG)
	existing := e.create(logoClientInputs(nil), false).Properties
	for _, c := range []struct{ file, url string }{{keyLogo, keyLogoURL}, {keyDarkLogo, keyDarkLogoURL}} {
		in := logoClientInputs(map[string]property.Value{c.file: img, c.url: property.New("https://img.example.com/x.png")})

		checked, err := e.prov.Check(p.CheckRequest{Urn: e.urn, Inputs: in})
		require.NoError(t, err)
		require.Len(t, checked.Failures, 1, c.file)
		assert.Equal(t, c.file, checked.Failures[0].Property)
		assert.Contains(t, checked.Failures[0].Reason, "mutually exclusive")

		_, err = e.prov.Create(p.CreateRequest{Urn: e.urn, Properties: in})
		require.ErrorContains(t, err, "mutually exclusive")
		_, err = e.prov.Create(p.CreateRequest{Urn: e.urn, Properties: in, DryRun: true})
		require.ErrorContains(t, err, "mutually exclusive")
		_, err = e.prov.Update(p.UpdateRequest{ID: "app", Urn: e.urn, State: existing, Inputs: in})
		require.ErrorContains(t, err, "mutually exclusive")
	}
	assert.Equal(t, 1, oidcCount(e.fake, "oidcClients"), "only the client created above exists")

	// Each one alone is fine.
	checked, err := e.prov.Check(p.CheckRequest{Urn: e.urn, Inputs: logoClientInputs(map[string]property.Value{
		keyLogo: img, keyDarkLogoURL: property.New("https://img.example.com/x.png"),
	})})
	require.NoError(t, err)
	assert.Empty(t, checked.Failures)
}

func TestOidcClientUploadedLogoSwitchesBetweenURLAndFile(t *testing.T) {
	t.Parallel()
	e := newLogoEnv(t)
	byURL := logoClientInputs(map[string]property.Value{keyLogoURL: property.New("https://img.example.com/logo.png")})
	created := e.create(byURL, false)
	assert.True(t, created.Properties.Get(keyLogoSHA).IsNull())

	// URL -> file: the file replaces the downloaded logo.
	byFile := logoClientInputs(map[string]property.Value{keyLogo: imgAsset(t, "logo.png", imgPNG)})
	toFile := e.update(created.ID, created.Properties, byFile)
	assert.Equal(t, imgPNG, e.fake.fakeImage("oidc/app/light")[imgData])
	assert.Equal(t, sum(imgPNG), toFile.Properties.Get(keyLogoSHA).AsString())

	// File -> URL: the logo downloaded from the new URL must survive.
	toURL := e.update(created.ID, retypeLogos(toFile.Properties), byURL)
	assert.Equal(t, []byte("downloaded:https://img.example.com/logo.png"), e.fake.fakeImage("oidc/app/light")[imgData])
	assert.True(t, toURL.Properties.Get("hasLogo").AsBool())
	assert.True(t, toURL.Properties.Get(keyLogoSHA).IsNull())
}

func TestOidcClientUploadedLogoPreviewMakesNoAPICall(t *testing.T) {
	t.Parallel()
	prov := testServer(t)
	configure(t, prov, "http://127.0.0.1:1", "k") // unreachable: any API call would fail
	r := imgRes{t: t, prov: prov, urn: urn("OidcClient")}
	in := logoClientInputs(map[string]property.Value{keyLogo: imgAsset(t, "logo.png", imgPNG)})

	created := r.create(in, true)
	assert.Equal(t, "app", created.ID)
	assert.Equal(t, "App", created.Properties.Get(keyName).AsString())

	in2 := in.Set(keyLogo, imgAsset(t, "logo2.png", imgPNG2))
	resp, err := prov.Update(p.UpdateRequest{
		ID: "app", Urn: r.urn, DryRun: true, State: retypeLogos(created.Properties), Inputs: in2,
	})
	require.NoError(t, err)
	assert.Equal(t, "App", resp.Properties.Get(keyName).AsString())
}

func TestOidcClientUploadedLogoImportLeavesAssetsEmpty(t *testing.T) {
	t.Parallel()
	e := newLogoEnv(t)
	in := e.checked(logoClientInputs(map[string]property.Value{
		keyLogo: imgAsset(t, "logo.png", imgPNG), keyDarkLogo: imgAsset(t, "dark.svg", imgSVG),
	}))
	created := e.create(in, false)

	imp := e.read(created.ID, property.NewMap(nil), property.NewMap(nil))
	assert.Equal(t, "app", imp.ID)
	for _, k := range []string{keyLogo, keyDarkLogo, keyLogoSHA, keyDarkSHA} {
		assert.True(t, imp.Properties.Get(k).IsNull(), k)
		assert.True(t, imp.Inputs.Get(k).IsNull(), k)
	}
	assert.True(t, imp.Properties.Get("hasLogo").AsBool())
	assert.True(t, imp.Properties.Get("hasDarkLogo").AsBool())

	// Setting the assets in the program afterwards shows a diff that re-uploads the files (idempotent).
	d := e.diff(created.ID, imp.Properties, in)
	assert.Equal(t, p.Add, d.DetailedDiff[keyLogo].Kind)
	upd := e.update(created.ID, imp.Properties, in)
	assert.Equal(t, sum(imgPNG), upd.Properties.Get(keyLogoSHA).AsString())
}

func TestOidcClientUploadedLogoRejectsArchives(t *testing.T) {
	t.Parallel()
	e := newLogoEnv(t)
	arch, err := archive.FromAssets(map[string]any{"a.txt": &resource.Asset{Text: "x"}})
	require.NoError(t, err)
	in := logoClientInputs(map[string]property.Value{keyLogo: property.New(arch)})
	_, err = e.prov.Create(p.CreateRequest{Urn: e.urn, Properties: in})
	require.Error(t, err)
	assert.Equal(t, 0, oidcCount(e.fake, "oidcClients"), "an invalid logo fails before the client is created")
}

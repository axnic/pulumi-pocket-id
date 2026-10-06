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
//go:build e2e

package tests

// End-to-end checks on idempotence (a refresh right after an apply must not
// change the inputs), import and the units of the API (TTL, durations).

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// plain turns a property value into a comparable Go value (secrets are transparent, nulls dropped).
func plain(v property.Value) any {
	switch {
	case v.IsNull():
		return nil
	case v.IsString():
		return v.AsString()
	case v.IsBool():
		if !v.AsBool() { // false is the zero value of the plain bool inputs: same as unset
			return nil
		}
		return true
	case v.IsNumber():
		return v.AsNumber()
	case v.IsAsset():
		return v.AsAsset().Hash
	case v.IsArray():
		out := []any{}
		for _, item := range v.AsArray().All {
			out = append(out, plain(item))
		}
		return out
	case v.IsMap():
		out := map[string]any{}
		if h, ok := v.AsMap().GetOk("hash"); ok && v.AsMap().Len() == 5 { // an asset serialized as a map
			return plain(h)
		}
		for k, item := range v.AsMap().All {
			if pv := plain(item); pv != nil {
				out[k] = pv
			}
		}
		return out
	}
	return fmt.Sprintf("%v", v)
}

// noDrift refreshes r with `in` as the inputs and requires the inputs returned by Read to be exactly `in`
// (a Read that rewrites an input makes the next `pulumi up` show a diff).
func (e *e2eEnv) noDrift(r *e2eRes, in e2eProps) {
	e.t.Helper()
	resp, err := e.prov.Read(p.ReadRequest{ID: r.id, Urn: e2eURN(r.typ), Inputs: property.NewMap(in), Properties: r.props})
	require.NoError(e.t, err, "read %s", r.typ)
	require.Equal(e.t, r.id, resp.ID, "%s disappeared on refresh", r.typ)
	assert.Equal(e.t, plain(property.New(in)), plain(property.New(resp.Inputs)), "%s drifts on refresh", r.typ)
}

// rawDo issues an authenticated request without a body and returns the status.
func (e *e2eEnv) rawDo(method, path string) int {
	e.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, e.baseURL+path, nil)
	require.NoError(e.t, err)
	req.Header.Set("X-API-Key", e.apiKey)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(e.t, err)
	_ = resp.Body.Close()
	return resp.StatusCode
}

func TestE2EOidcClientMinimalIdempotentAndImport(t *testing.T) {
	e := newE2E(t)
	in := e2eProps{keyName: property.New(e2eName("min")), keyCallbackURLs: e2eStrs("https://app.example.com/cb")}
	client := e.create("OidcClient", in)
	e.noDrift(client, in)
	e.noDrift(client, in) // a second refresh stays quiet too

	imp := e.importRead("OidcClient", client.id)
	assert.Equal(t, in[keyName].AsString(), imp.Get(keyName).AsString())
	assert.False(t, imp.Get("hasLogo").AsBool())
	assert.True(t, imp.Get("launchUrl").IsNull(), "an unset launch URL is not turned into an empty string")
	assert.True(t, imp.Get("backchannelLogoutUrl").IsNull())
}

func TestE2EOidcClientFullIdempotentAndUpdate(t *testing.T) {
	e := newE2E(t)
	group := e.group("full")
	in := e2eProps{
		keyName:                       property.New(e2eName("full")),
		keyDescription:                property.New("e2e client"),
		keyCallbackURLs:               e2eStrs("https://app.example.com/cb", "https://app.example.com/cb2"),
		"logoutCallbackUrls":          e2eStrs("https://app.example.com/logout"),
		"launchUrl":                   property.New("https://app.example.com/launch"),
		"backchannelLogoutUrl":        property.New("https://app.example.com/backchannel"),
		"pkceEnabled":                 property.New(true),
		"skipConsent":                 property.New(true),
		"accessTokenDurationMinutes":  property.New(30.0),
		"refreshTokenDurationMinutes": property.New(600.0),
		keyAllowedUserGroupIDs:        e2eStrs(group.id),
	}
	if !e.atLeast(2, 17) {
		// Pocket-ID < v2.17.0 ignores backchannelLogoutURL (the provider rejects it, see the unit tests).
		delete(in, "backchannelLogoutUrl")
	}
	client := e.create("OidcClient", in)
	e.noDrift(client, in)

	var api struct {
		LaunchURL                   string `json:"launchURL"`
		AccessTokenDurationMinutes  int    `json:"accessTokenDurationMinutes"`
		RefreshTokenDurationMinutes int    `json:"refreshTokenDurationMinutes"`
		SkipConsent                 bool   `json:"skipConsent"`
		IsGroupRestricted           bool   `json:"isGroupRestricted"`
	}
	require.Equal(t, http.StatusOK, e.rawGet("/api/oidc/clients/"+client.id, &api))
	assert.Equal(t, "https://app.example.com/launch", api.LaunchURL)
	assert.Equal(t, 30, api.AccessTokenDurationMinutes)
	assert.Equal(t, 600, api.RefreshTokenDurationMinutes)
	assert.True(t, api.SkipConsent)
	assert.True(t, api.IsGroupRestricted)

	// Removing optional inputs clears them on the server and stays drift-free.
	in2 := e2eProps{keyName: in[keyName], keyCallbackURLs: in[keyCallbackURLs]}
	e.update(client, in2)
	e.noDrift(client, in2)
	api.LaunchURL = ""
	require.Equal(t, http.StatusOK, e.rawGet("/api/oidc/clients/"+client.id, &api))
	assert.Empty(t, api.LaunchURL)
	assert.False(t, api.IsGroupRestricted)
	assert.False(t, api.SkipConsent)

	// Import of a fully set client returns the same inputs as the program.
	e.update(client, in)
	imp := e.importRead("OidcClient", client.id)
	assert.Equal(t, "https://app.example.com/launch", imp.Get("launchUrl").AsString())
	assert.Equal(t, 30.0, imp.Get("accessTokenDurationMinutes").AsNumber())
	assert.Equal(t, []any{group.id}, plain(imp.Get(keyAllowedUserGroupIDs)))
}

func TestE2EOidcClientLogoURL(t *testing.T) {
	e := newE2E(t)
	// Pocket-ID downloads the logo itself and refuses private addresses, so this needs a public URL.
	in := e2eProps{
		keyName: property.New(e2eName("logourl")), keyCallbackURLs: e2eStrs("https://app.example.com/cb"),
		keyLogoURL: property.New("https://www.google.com/favicon.ico"),
	}
	client, err := e.tryCreate("OidcClient", in)
	if err != nil && strings.Contains(err.Error(), "Logo could not be downloaded") {
		t.Skipf("the instance has no Internet access: %v", err)
	}
	require.NoError(t, err)
	assert.True(t, client.props.Get("hasLogo").AsBool())
	e.noDrift(client, in)
	status, _ := e.rawBytes("/api/oidc/clients/" + client.id + "/logo?light=true")
	assert.Equal(t, http.StatusOK, status)

	// Dropping the URL from the program removes the logo.
	in2 := e2eProps{keyName: in[keyName], keyCallbackURLs: in[keyCallbackURLs]}
	e.update(client, in2)
	e.noDrift(client, in2)
	status, _ = e.rawBytes("/api/oidc/clients/" + client.id + "/logo?light=true")
	assert.Equal(t, http.StatusNotFound, status)
}

func TestE2ESignupTokenTTLIdempotentAndImport(t *testing.T) {
	e := newE2E(t)
	group := e.group("signup")
	in := e2eProps{keyTTL: property.New(3600.0), keyUsageLimit: property.New(3.0), keyUserGroupIDs: e2eStrs(group.id)}
	token := e.createOrSkip("SignupToken", in)

	created, err := time.Parse(time.RFC3339, token.props.Get(keyCreatedAt).AsString())
	require.NoError(t, err)
	expires, err := time.Parse(time.RFC3339, token.props.Get(keyExpiresAt).AsString())
	require.NoError(t, err)
	assert.Equal(t, time.Hour, expires.Sub(created), "ttl is expressed in seconds")

	e.noDrift(token, in)
	e.noDrift(token, in)

	imp := e.importRead("SignupToken", token.id)
	assert.Equal(t, 3600.0, imp.Get(keyTTL).AsNumber(), "the TTL is derived from the timestamps on import")
	assert.Equal(t, 3.0, imp.Get(keyUsageLimit).AsNumber())
	assert.Equal(t, []any{group.id}, plain(imp.Get(keyUserGroupIDs)))

	// A signup token without groups.
	in2 := e2eProps{keyTTL: property.New(60.0), keyUsageLimit: property.New(1.0)}
	bare := e.create("SignupToken", in2)
	e.noDrift(bare, in2)

	e.destroy(token)
	resp, err := e.prov.Read(p.ReadRequest{ID: token.id, Urn: e2eURN("SignupToken"), Properties: token.props})
	require.NoError(t, err)
	assert.Empty(t, resp.ID)
}

func TestE2EScimIdempotentAndImport(t *testing.T) {
	e := newE2E(t)
	client := e.create("OidcClient", e2eProps{
		keyName: property.New(e2eName("scim2")), keyCallbackURLs: e2eStrs("https://app.example.com/cb"),
	})
	in := e2eProps{
		keyOidcClientID: property.New(client.id),
		keyEndpoint:     property.New("https://scim.example.com/v2"),
		keyToken:        property.New("scim-token"),
	}
	scim := e.createOrSkip("ScimServiceProvider", in)
	e.noDrift(scim, in)

	// The import ID is <oidcClientId>/<serviceProviderId>; the resource ID afterwards is the provider's own.
	resp, err := e.prov.Read(p.ReadRequest{
		ID: client.id + "/" + scim.id, Urn: e2eURN("ScimServiceProvider"), Properties: property.NewMap(nil),
	})
	require.NoError(t, err)
	require.Equal(t, scim.id, resp.ID)
	imp := resp.Properties
	assert.Equal(t, "https://scim.example.com/v2", imp.Get(keyEndpoint).AsString())
	assert.Equal(t, client.id, imp.Get(keyOidcClientID).AsString())
}

func TestE2EIdentityIdempotentAndImport(t *testing.T) {
	e := newE2E(t)
	name := e2eName("idem")
	uin := e2eProps{
		keyUsername: property.New(name), keyEmail: property.New(name + "@example.com"),
		keyFirstName: property.New("Idem"), keyDisplayName: property.New("Idem User"),
	}
	user := e.create("User", uin)
	e.noDrift(user, uin)
	imp := e.importRead("User", user.id)
	assert.Equal(t, name, imp.Get(keyUsername).AsString())

	gin := e2eProps{keyName: property.New(e2eName("idemg")), keyFriendlyName: property.New("Idem group")}
	group := e.create("UserGroup", gin)
	e.noDrift(group, gin)
	assert.Equal(t, "Idem group", e.importRead("UserGroup", group.id).Get(keyFriendlyName).AsString())

	membersIn := e2eProps{keyGroupID: property.New(group.id), keyUserIDs: e2eStrs(user.id)}
	members := e.create("UserGroupMembers", membersIn)
	e.noDrift(members, membersIn)
	assert.Equal(t, []any{user.id}, plain(e.importRead("UserGroupMembers", group.id).Get(keyUserIDs)))

	ucin := e2eProps{keyUserID: property.New(user.id), keyClaims: property.New(e2eProps{"department": property.New("it")})}
	uc := e.create("UserCustomClaims", ucin)
	e.noDrift(uc, ucin)
	assert.Equal(t, "it", e.importRead("UserCustomClaims", user.id).Get(keyClaims).AsMap().Get("department").AsString())

	gcin := e2eProps{keyUserGroupID: property.New(group.id), keyClaims: property.New(e2eProps{"tier": property.New("gold")})}
	gc := e.create("UserGroupCustomClaims", gcin)
	e.noDrift(gc, gcin)
	assert.Equal(t, "gold", e.importRead("UserGroupCustomClaims", group.id).Get(keyClaims).AsMap().Get("tier").AsString())
}

func TestE2EApiIdempotentAndImport(t *testing.T) {
	e := newE2E(t)
	client := e.create("OidcClient", e2eProps{
		keyName: property.New(e2eName("apic")), keyCallbackURLs: e2eStrs("https://app.example.com/cb"),
	})
	in := e2eProps{
		keyName:    property.New(e2eName("api")),
		"resource": property.New("https://" + e2eName("api") + ".example.com"),
		"permissions": property.New([]property.Value{property.New(e2eProps{
			keyKey: property.New(valRead), keyName: property.New("Read"), keyDescription: property.New("Read access"),
		})}),
	}
	api := e.createOrSkip("Api", in)
	e.noDrift(api, in)
	assert.Equal(t, in[keyName].AsString(), e.importRead("Api", api.id).Get(keyName).AsString())

	gin := e2eProps{
		"apiId": property.New(api.id), keyClientID: property.New(client.id),
		"clientAccess": property.New(true), "clientPermissionKeys": e2eStrs(valRead),
	}
	grant := e.create("ApiClientGrant", gin)
	e.noDrift(grant, gin)
	imp := e.importRead("ApiClientGrant", grant.id)
	assert.True(t, imp.Get("clientAccess").AsBool())
}

func TestE2EOidcClientSecretIdempotentAndImport(t *testing.T) {
	e := newE2E(t)
	client := e.create("OidcClient", e2eProps{
		keyName: property.New(e2eName("sec")), keyCallbackURLs: e2eStrs("https://app.example.com/cb"),
	})
	in := e2eProps{keyClientID: property.New(client.id)}
	secret := e.create("OidcClientSecret", in)
	e.noDrift(secret, in)
	imp := e.importRead("OidcClientSecret", secret.id)
	assert.Equal(t, secret.props.Get("secretId").AsString(), imp.Get("secretId").AsString())

	dated := e2eProps{keyClientID: property.New(client.id), keyExpiresAt: property.New("2099-01-01T00:00:00Z")}
	withExp := e.create("OidcClientSecret", dated)
	e.noDrift(withExp, dated)
}

func TestE2EApplicationConfigurationIdempotent(t *testing.T) {
	e := newE2E(t)
	var current []struct{ Key, Value string }
	require.Equal(t, http.StatusOK, e.rawGet("/api/application-configuration/all", &current))
	original := ""
	for _, v := range current {
		if v.Key == keyAppName {
			original = v.Value
		}
	}
	in := e2eProps{keyAppName: property.New(original)}
	cfg := e.create("ApplicationConfiguration", in)
	e.noDrift(cfg, in)
	e.noDrift(cfg, in)
}

// e2eLogo drives the logo uploads of an OidcClient on the real instance.
type e2eLogo struct {
	*e2eEnv
	client *e2eRes
}

// check runs inputs through Check as the engine does (it fills the defaults of the optional inputs).
func (l e2eLogo) check(in e2eProps) e2eProps {
	l.t.Helper()
	resp, err := l.prov.Check(p.CheckRequest{Urn: e2eURN("OidcClient"), Inputs: property.NewMap(in)})
	require.NoError(l.t, err)
	require.Empty(l.t, resp.Failures)
	return e2eProps(retypeLogos(resp.Inputs).AsMap())
}

// diff returns what a `pulumi up` with these inputs would change.
func (l e2eLogo) diff(in e2eProps) map[string]p.PropertyDiff {
	l.t.Helper()
	resp, err := l.prov.Diff(p.DiffRequest{
		ID: l.client.id, Urn: e2eURN("OidcClient"), State: l.client.props, Inputs: property.NewMap(l.check(in)),
	})
	require.NoError(l.t, err)
	return resp.DetailedDiff
}

func (l e2eLogo) up(in e2eProps) {
	l.t.Helper()
	resp, err := l.prov.Update(p.UpdateRequest{
		ID: l.client.id, Urn: e2eURN("OidcClient"), State: l.client.props, Inputs: property.NewMap(l.check(in)),
	})
	require.NoError(l.t, err)
	l.client.props = retypeLogos(resp.Properties)
}

// refresh re-reads the client and keeps the result as its state, as `pulumi refresh` does.
func (l e2eLogo) refresh(in e2eProps) property.Map {
	l.t.Helper()
	resp, err := l.prov.Read(p.ReadRequest{
		ID: l.client.id, Urn: e2eURN("OidcClient"), Inputs: property.NewMap(l.check(in)), Properties: l.client.props,
	})
	require.NoError(l.t, err)
	require.Equal(l.t, l.client.id, resp.ID)
	l.client.props = retypeLogos(resp.Properties)
	return l.client.props
}

func (l e2eLogo) served(light bool) (int, []byte) {
	return l.rawBytes(fmt.Sprintf("/api/oidc/clients/%s/logo?light=%t", l.client.id, light))
}

// upload writes a logo behind the provider's back.
func (l e2eLogo) upload(light bool, data []byte) {
	l.t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "oob.png")
	require.NoError(l.t, err)
	_, err = part.Write(data)
	require.NoError(l.t, err)
	require.NoError(l.t, w.Close())
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		fmt.Sprintf("%s/api/oidc/clients/%s/logo?light=%t", l.baseURL, l.client.id, light), &body)
	require.NoError(l.t, err)
	req.Header.Set("X-API-Key", l.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	require.NoError(l.t, err)
	_ = resp.Body.Close()
	require.Equal(l.t, http.StatusNoContent, resp.StatusCode)
}

func TestE2EOidcClientUploadedLogos(t *testing.T) {
	e := newE2E(t)
	red, blue := e2ePNG(t, 8, e2eRed), e2ePNG(t, 8, color.RGBA{B: 255, A: 255})
	green := e2ePNG(t, 8, color.RGBA{G: 255, A: 255})
	base := e2eProps{keyName: property.New(e2eName("logos")), keyCallbackURLs: e2eStrs("https://app.example.com/cb")}
	with := func(kv e2eProps) e2eProps {
		in := e2eProps{}
		for k, v := range base {
			in[k] = v
		}
		for k, v := range kv {
			in[k] = v
		}
		return in
	}

	in := with(e2eProps{keyLogo: imgAsset(t, "logo.png", red), keyDarkLogo: imgAsset(t, "dark.png", blue)})
	client := e.create("OidcClient", in)
	client.props = retypeLogos(client.props)
	l := e2eLogo{e, client}
	assert.True(t, client.props.Get("hasLogo").AsBool())
	assert.True(t, client.props.Get("hasDarkLogo").AsBool())
	assert.Equal(t, sum(red), client.props.Get(keyLogoSHA).AsString())
	assert.Equal(t, sum(blue), client.props.Get(keyDarkSHA).AsString())
	status, body := l.served(true)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, red, body)
	status, body = l.served(false)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, blue, body)

	// Nothing changed: an up and a refresh are both silent, however many times.
	for range 2 {
		assert.Empty(t, l.diff(in))
		state := l.refresh(in)
		assert.Equal(t, sum(red), state.Get(keyLogoSHA).AsString())
		assert.Equal(t, sum(blue), state.Get(keyDarkSHA).AsString())
		assert.Empty(t, l.diff(in), "a refresh does not create a diff")
	}

	// A new file for the light logo only.
	in2 := with(e2eProps{keyLogo: imgAsset(t, "logo2.png", green), keyDarkLogo: in[keyDarkLogo]})
	assert.Equal(t, p.Update, l.diff(in2)[keyLogo].Kind)
	assert.NotContains(t, l.diff(in2), keyDarkLogo)
	l.up(in2)
	assert.Equal(t, sum(green), client.props.Get(keyLogoSHA).AsString())
	_, body = l.served(true)
	assert.Equal(t, green, body)
	_, body = l.served(false)
	assert.Equal(t, blue, body)
	assert.Empty(t, l.diff(in2))

	// An update of another field keeps both logos.
	in3 := with(e2eProps{keyName: property.New(e2eName("logos2")), keyLogo: in2[keyLogo], keyDarkLogo: in2[keyDarkLogo]})
	l.up(in3)
	assert.True(t, client.props.Get("hasLogo").AsBool())
	assert.True(t, client.props.Get("hasDarkLogo").AsBool())
	assert.Empty(t, l.diff(in3))

	// The served bytes changed out of band: detected, then repaired.
	l.upload(true, red)
	state := l.refresh(in3)
	assert.Equal(t, sum(red), state.Get(keyLogoSHA).AsString())
	assert.Equal(t, p.Update, l.diff(in3)[keyLogo].Kind)
	l.up(in3)
	_, body = l.served(true)
	assert.Equal(t, green, body)
	assert.Empty(t, l.diff(in3))

	// Deleted out of band: the refresh sees no dark logo (404) and the diff proposes to upload it again.
	require.Equal(t, http.StatusNoContent, l.rawDo(http.MethodDelete, "/api/oidc/clients/"+client.id+"/logo?light=false"))
	state = l.refresh(in3)
	assert.False(t, state.Get("hasDarkLogo").AsBool())
	assert.True(t, state.Get(keyDarkSHA).IsNull())
	assert.Equal(t, p.Add, l.diff(in3)[keyDarkLogo].Kind)
	l.up(in3)
	_, body = l.served(false)
	assert.Equal(t, blue, body)
	assert.Empty(t, l.diff(in3))

	// An import knows nothing about the files.
	imp := e.importRead("OidcClient", client.id)
	assert.True(t, imp.Get("hasLogo").AsBool())
	assert.True(t, imp.Get(keyLogo).IsNull())
	assert.True(t, imp.Get(keyLogoSHA).IsNull())

	// Dropping logo from the program deletes the light logo only, then darkLogo goes too.
	in4 := with(e2eProps{keyName: in3[keyName], keyDarkLogo: in3[keyDarkLogo]})
	assert.Equal(t, p.Delete, l.diff(in4)[keyLogo].Kind)
	l.up(in4)
	// With a dark logo left, the instance still serves one on light=true (a fallback): hasLogo is the truth.
	var flags struct{ HasLogo, HasDarkLogo bool }
	require.Equal(t, http.StatusOK, l.rawGet("/api/oidc/clients/"+client.id, &flags))
	assert.False(t, flags.HasLogo)
	assert.True(t, flags.HasDarkLogo)
	assert.False(t, client.props.Get("hasLogo").AsBool())
	assert.Empty(t, l.diff(in4))
	assert.Empty(t, l.diff(in4), "up without change")

	in5 := with(e2eProps{keyName: in3[keyName]})
	l.up(in5)
	status, _ = l.served(true)
	assert.Equal(t, http.StatusNotFound, status)
	status, _ = l.served(false)
	assert.Equal(t, http.StatusNotFound, status)
	assert.False(t, client.props.Get("hasDarkLogo").AsBool())
	state = l.refresh(in5)
	assert.True(t, state.Get(keyDarkSHA).IsNull())
	assert.Empty(t, l.diff(in5))
}

func TestE2EOidcClientUploadedLogoExclusiveWithURL(t *testing.T) {
	e := newE2E(t)
	in := e2eProps{
		keyName: property.New(e2eName("excl")), keyCallbackURLs: e2eStrs("https://app.example.com/cb"),
		keyLogo: imgAsset(t, "logo.png", e2ePNG(t, 8, e2eRed)), keyLogoURL: property.New("https://example.com/logo.png"),
	}
	resp, err := e.prov.Check(p.CheckRequest{Urn: e2eURN("OidcClient"), Inputs: property.NewMap(in)})
	require.NoError(t, err)
	require.Len(t, resp.Failures, 1)
	assert.Contains(t, resp.Failures[0].Reason, "mutually exclusive")
	_, err = e.tryCreate("OidcClient", in)
	require.ErrorContains(t, err, "mutually exclusive")
}

var e2eRed = color.RGBA{R: 255, A: 255}

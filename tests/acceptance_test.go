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

// End-to-end tests: every resource of the provider, driven through the real
// provider against a real Pocket-ID instance. Run through mise:
//
//	E2E_VERSION=v2.3.0 mise run ci:e2e
//
// or by hand against a running instance (docker-compose.test.yml):
//
//	POCKET_ID_BASE_URL=http://localhost:1411 POCKET_ID_API_KEY=pulumi-test-api-key-0123456789 \
//	  go test -tags e2e ./tests/... -count=1 -v
//
// Resources are driven with property maps (no provider types), so the suite
// only depends on the property names of the schema. Everything is created
// with unique names and removed on cleanup, so the suite can be re-run against
// the same instance.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"

	"github.com/axnic/pulumi-pocket-id/provider"
)

type e2eProps = map[string]property.Value

var e2eCounter atomic.Uint64

// e2eName returns a unique, lowercase, API-safe name.
func e2eName(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().Unix()%1_000_000, e2eCounter.Add(1))
}

func e2eStrs(v ...string) property.Value {
	out := make([]property.Value, len(v))
	for i, s := range v {
		out[i] = property.New(s)
	}
	return property.New(out)
}

// e2eEnv is a provider server configured against the live instance.
type e2eEnv struct {
	t       *testing.T
	prov    integration.Server
	baseURL string
	apiKey  string
	version string
}

func newE2E(t *testing.T) *e2eEnv {
	t.Helper()
	baseURL, apiKey := os.Getenv("POCKET_ID_BASE_URL"), os.Getenv("POCKET_ID_API_KEY")
	require.NotEmpty(t, baseURL, "POCKET_ID_BASE_URL must be set for e2e tests")
	require.NotEmpty(t, apiKey, "POCKET_ID_API_KEY must be set for e2e tests")

	prov, err := integration.NewServer(context.Background(), provider.Name, semver.MustParse("1.0.0"),
		integration.WithProvider(provider.Provider()))
	require.NoError(t, err)
	require.NoError(t, prov.Configure(p.ConfigureRequest{Args: property.NewMap(e2eProps{
		"baseUrl": property.New(baseURL), keyAPIKey: property.New(apiKey),
	})}))

	e := &e2eEnv{t: t, prov: prov, baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey}
	var ver map[string]any
	if e.rawGet("/api/version/current", &ver) == http.StatusOK {
		for _, v := range ver {
			if s, ok := v.(string); ok {
				e.version = s
			}
		}
	}
	t.Logf("Pocket-ID version: %q", e.version)
	return e
}

func e2eURN(typ string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type("pocket-id:index:"+typ), "e2e")
}

// rawGet issues an authenticated GET and decodes a JSON body into out (when non-nil).
func (e *e2eEnv) rawGet(path string, out any) int {
	e.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, e.baseURL+path, nil)
	require.NoError(e.t, err)
	req.Header.Set("X-API-Key", e.apiKey)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(e.t, err)
	defer func() { _ = resp.Body.Close() }()
	if out != nil && resp.StatusCode == http.StatusOK {
		require.NoError(e.t, json.NewDecoder(resp.Body).Decode(out))
	}
	return resp.StatusCode
}

// e2eRes is a created resource, kept to update, read and delete it.
type e2eRes struct {
	typ   string
	id    string
	props property.Map // current state
}

// tryCreate creates a resource and registers its deletion on cleanup.
func (e *e2eEnv) tryCreate(typ string, in e2eProps) (*e2eRes, error) {
	e.t.Helper()
	resp, err := e.prov.Create(p.CreateRequest{Urn: e2eURN(typ), Properties: property.NewMap(in)})
	if err != nil {
		return nil, err
	}
	r := &e2eRes{typ: typ, id: resp.ID, props: resp.Properties}
	e.t.Cleanup(func() {
		_ = e.prov.Delete(p.DeleteRequest{ID: r.id, Urn: e2eURN(r.typ), Properties: r.props})
	})
	return r, nil
}

func (e *e2eEnv) create(typ string, in e2eProps) *e2eRes {
	e.t.Helper()
	r, err := e.tryCreate(typ, in)
	require.NoError(e.t, err, "create %s", typ)
	require.NotEmpty(e.t, r.id)
	return r
}

// createOrSkip skips the test when the instance does not serve the endpoint
// (the resource's API does not exist in this Pocket-ID version).
func (e *e2eEnv) createOrSkip(typ string, in e2eProps) *e2eRes {
	e.t.Helper()
	r, err := e.tryCreate(typ, in)
	if err != nil && (strings.Contains(err.Error(), ": 404:") || strings.Contains(err.Error(), ": 405:")) {
		e.t.Skipf("%s is not available on Pocket-ID %q: %v", typ, e.version, err)
	}
	require.NoError(e.t, err, "create %s", typ)
	return r
}

func (e *e2eEnv) update(r *e2eRes, in e2eProps) {
	e.t.Helper()
	resp, err := e.prov.Update(p.UpdateRequest{ID: r.id, Urn: e2eURN(r.typ), State: r.props, Inputs: property.NewMap(in)})
	require.NoError(e.t, err, "update %s", r.typ)
	r.props = resp.Properties
}

// read refreshes r from the server using its own state as inputs and returns the result.
func (e *e2eEnv) read(r *e2eRes, in e2eProps) property.Map {
	e.t.Helper()
	resp, err := e.prov.Read(p.ReadRequest{ID: r.id, Urn: e2eURN(r.typ), Inputs: property.NewMap(in), Properties: r.props})
	require.NoError(e.t, err, "read %s", r.typ)
	return resp.Properties
}

// importRead reads a resource knowing only its ID, as `pulumi import` does.
func (e *e2eEnv) importRead(typ, id string) property.Map {
	e.t.Helper()
	resp, err := e.prov.Read(p.ReadRequest{ID: id, Urn: e2eURN(typ), Properties: property.NewMap(nil)})
	require.NoError(e.t, err, "import %s", typ)
	require.Equal(e.t, id, resp.ID)
	return resp.Properties
}

func (e *e2eEnv) destroy(r *e2eRes) {
	e.t.Helper()
	require.NoError(e.t, e.prov.Delete(p.DeleteRequest{ID: r.id, Urn: e2eURN(r.typ), Properties: r.props}))
}

func (e *e2eEnv) user(prefix string) *e2eRes {
	name := e2eName(prefix)
	return e.create("User", e2eProps{
		keyUsername: property.New(name), keyEmail: property.New(name + "@example.com"),
		keyFirstName: property.New("E2E"), keyDisplayName: property.New("E2E " + name),
	})
}

func (e *e2eEnv) group(prefix string) *e2eRes {
	name := e2eName(prefix)
	return e.create("UserGroup", e2eProps{keyName: property.New(name), keyFriendlyName: property.New("E2E " + name)})
}

func TestE2EUserAndGroup(t *testing.T) {
	e := newE2E(t)

	user := e.user("user")
	var apiUser struct {
		Username string `json:"username"`
	}
	require.Equal(t, http.StatusOK, e.rawGet("/api/users/"+user.id, &apiUser))
	assert.Equal(t, user.props.Get(keyUsername).AsString(), apiUser.Username)

	// Update, then drift-free read.
	in := e2eProps{
		keyUsername: user.props.Get(keyUsername), keyEmail: user.props.Get(keyEmail),
		keyFirstName: property.New("Renamed"), keyDisplayName: property.New("Renamed E2E"),
	}
	e.update(user, in)
	assert.Equal(t, "Renamed", e.read(user, in).Get(keyFirstName).AsString())

	// Import from the ID alone.
	assert.Equal(t, user.props.Get(keyUsername).AsString(), e.importRead("User", user.id).Get(keyUsername).AsString())

	group := e.group("group")
	gin := e2eProps{keyName: group.props.Get(keyName), keyFriendlyName: property.New("Renamed group")}
	e.update(group, gin)
	assert.Equal(t, "Renamed group", e.read(group, gin).Get(keyFriendlyName).AsString())

	// Gone resources read back with an empty ID.
	e.destroy(group)
	resp, err := e.prov.Read(p.ReadRequest{ID: group.id, Urn: e2eURN("UserGroup"), Properties: group.props})
	require.NoError(t, err)
	assert.Empty(t, resp.ID)
}

func TestE2EUserGroupMembers(t *testing.T) {
	e := newE2E(t)
	group, alice, bob := e.group("members"), e.user("alice"), e.user("bob")

	members := e.create("UserGroupMembers", e2eProps{keyGroupID: property.New(group.id), keyUserIDs: e2eStrs(alice.id)})
	var apiGroup struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	require.Equal(t, http.StatusOK, e.rawGet("/api/user-groups/"+group.id, &apiGroup))
	require.Len(t, apiGroup.Users, 1)
	assert.Equal(t, alice.id, apiGroup.Users[0].ID)

	in := e2eProps{keyGroupID: property.New(group.id), keyUserIDs: e2eStrs(alice.id, bob.id)}
	e.update(members, in)
	apiGroup.Users = nil
	require.Equal(t, http.StatusOK, e.rawGet("/api/user-groups/"+group.id, &apiGroup))
	assert.Len(t, apiGroup.Users, 2)

	// Deleting the association empties the group but keeps the users.
	e.destroy(members)
	apiGroup.Users = nil
	require.Equal(t, http.StatusOK, e.rawGet("/api/user-groups/"+group.id, &apiGroup))
	assert.Empty(t, apiGroup.Users)
	assert.Equal(t, http.StatusOK, e.rawGet("/api/users/"+alice.id, nil))
}

func TestE2ECustomClaims(t *testing.T) {
	e := newE2E(t)
	user, group := e.user(keyClaims), e.group(keyClaims)

	// Pocket-ID has no dedicated GET for custom claims: they are a field of the user / group.
	type claimsDTO struct {
		CustomClaims []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"customClaims"`
	}
	userClaims := func() claimsDTO {
		var dto claimsDTO
		require.Equal(t, http.StatusOK, e.rawGet("/api/users/"+user.id, &dto))
		return dto
	}
	groupClaims := func() claimsDTO {
		var dto claimsDTO
		require.Equal(t, http.StatusOK, e.rawGet("/api/user-groups/"+group.id, &dto))
		return dto
	}

	uin := e2eProps{
		keyUserID: property.New(user.id),
		keyClaims: property.New(e2eProps{"department": property.New("engineering")}),
	}
	uc := e.create("UserCustomClaims", uin)
	dto := userClaims()
	require.Len(t, dto.CustomClaims, 1)
	assert.Equal(t, "department", dto.CustomClaims[0].Key)
	assert.Equal(t, "engineering", dto.CustomClaims[0].Value)

	uin = e2eProps{keyUserID: property.New(user.id), keyClaims: property.New(e2eProps{
		"department": property.New("sales"), valTeam: property.New("blue"),
	})}
	e.update(uc, uin)
	assert.Len(t, userClaims().CustomClaims, 2)

	gin := e2eProps{
		keyUserGroupID: property.New(group.id),
		keyClaims:      property.New(e2eProps{"tier": property.New("gold")}),
	}
	gc := e.create("UserGroupCustomClaims", gin)
	dto = groupClaims()
	require.Len(t, dto.CustomClaims, 1)
	assert.Equal(t, "tier", dto.CustomClaims[0].Key)

	e.destroy(gc)
	assert.Empty(t, groupClaims().CustomClaims)
}

func TestE2EOidcClientAndSecret(t *testing.T) {
	e := newE2E(t)
	group := e.group("oidc")

	in := e2eProps{
		keyName:                property.New(e2eName("client")),
		keyCallbackURLs:        e2eStrs("https://app.example.com/callback"),
		keyAllowedUserGroupIDs: e2eStrs(group.id),
	}
	client := e.create("OidcClient", in)
	// The server-generated client ID is the resource ID (the clientId input is only for a chosen ID).
	require.NotEmpty(t, client.id)

	var apiClient struct {
		Name         string   `json:"name"`
		CallbackURLs []string `json:"callbackURLs"`
	}
	require.Equal(t, http.StatusOK, e.rawGet("/api/oidc/clients/"+client.id, &apiClient))
	assert.Equal(t, in[keyName].AsString(), apiClient.Name)
	assert.Equal(t, []string{"https://app.example.com/callback"}, apiClient.CallbackURLs)

	in[keyCallbackURLs] = e2eStrs("https://app.example.com/callback", "https://app.example.com/other")
	e.update(client, in)
	assert.Equal(t, 2, e.read(client, in).Get(keyCallbackURLs).AsArray().Len())

	// Import from the ID alone.
	assert.Equal(t, in[keyName].AsString(), e.importRead("OidcClient", client.id).Get(keyName).AsString())

	secret := e.create("OidcClientSecret", e2eProps{keyClientID: property.New(client.id)})
	assert.NotEmpty(t, secret.props.Get("secret").AsString(), "the secret is returned at creation")
	assert.NotEmpty(t, secret.props.Get("secretId").AsString())
	e.destroy(secret)
}

func TestE2EApiAndGrant(t *testing.T) {
	e := newE2E(t)
	client := e.create("OidcClient", e2eProps{
		keyName: property.New(e2eName("grant-client")), keyCallbackURLs: e2eStrs("https://app.example.com/callback"),
	})

	api := e.createOrSkip("Api", e2eProps{
		keyName:    property.New(e2eName("api")),
		"resource": property.New("https://" + e2eName("api") + ".example.com"),
		"permissions": property.New([]property.Value{property.New(e2eProps{
			keyKey: property.New(valRead), keyName: property.New("Read"), keyDescription: property.New("Read access"),
		})}),
	})
	assert.NotEmpty(t, api.props.Get("permissions").AsArray().Len())

	grant := e.create("ApiClientGrant", e2eProps{
		"apiId":                 property.New(api.id),
		keyClientID:             property.New(client.id),
		keyClientAccess:         property.New(true),
		keyClientPermissionKeys: e2eStrs(valRead),
	})
	gin := e2eProps{
		"apiId": property.New(api.id), keyClientID: property.New(client.id),
		keyClientAccess: property.New(false), keyUserDelegatedAccess: property.New(true),
		"userDelegatedPermissionKeys": e2eStrs(valRead),
	}
	e.update(grant, gin)
	assert.True(t, e.read(grant, gin).Get(keyUserDelegatedAccess).AsBool())

	// Deleting the grant must not delete the API.
	e.destroy(grant)
	assert.Equal(t, http.StatusOK, e.rawGet("/api/apis/"+api.id, nil))
}

func TestE2EScimServiceProvider(t *testing.T) {
	e := newE2E(t)
	client := e.create("OidcClient", e2eProps{
		keyName: property.New(e2eName("scim-client")), keyCallbackURLs: e2eStrs("https://app.example.com/callback"),
	})
	scim := e.createOrSkip("ScimServiceProvider", e2eProps{
		keyOidcClientID: property.New(client.id),
		keyEndpoint:     property.New("https://scim.example.com/v2"),
		keyToken:        property.New("scim-token"),
	})
	in := e2eProps{
		keyOidcClientID: property.New(client.id),
		keyEndpoint:     property.New("https://scim.example.com/v2/updated"),
		keyToken:        property.New("scim-token"),
	}
	e.update(scim, in)
	assert.Equal(t, "https://scim.example.com/v2/updated", e.read(scim, in).Get(keyEndpoint).AsString())
}

func TestE2ESignupToken(t *testing.T) {
	e := newE2E(t)
	group := e.group("signup")
	token := e.createOrSkip("SignupToken", e2eProps{
		keyTTL: property.New(3600.0), keyUsageLimit: property.New(3.0), keyUserGroupIDs: e2eStrs(group.id),
	})
	assert.NotEmpty(t, token.props.Get(keyToken).AsString(), "the token is returned at creation")
	e.destroy(token)
}

func TestE2EApplicationConfiguration(t *testing.T) {
	e := newE2E(t)

	var current []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	require.Equal(t, http.StatusOK, e.rawGet("/api/application-configuration/all", &current))
	original := ""
	for _, v := range current {
		if v.Key == keyAppName {
			original = v.Value
		}
	}
	require.NotEmpty(t, original)

	cfg := e.create("ApplicationConfiguration", e2eProps{keyAppName: property.New(original)})
	assert.Equal(t, "app-config", cfg.id)
	// The suite leaves the instance as it found it; Delete itself is a no-op.
	t.Cleanup(func() { e.update(cfg, e2eProps{keyAppName: property.New(original)}) })

	in := e2eProps{keyAppName: property.New("E2E Pocket ID"), keySessionDuration: property.New(90.0)}
	e.update(cfg, in)
	read := e.read(cfg, in)
	assert.Equal(t, "E2E Pocket ID", read.Get(keyAppName).AsString())
	assert.Equal(t, 90.0, read.Get(keySessionDuration).AsNumber())

	// A full import reads every non-secret field.
	imported := e.importRead("ApplicationConfiguration", "app-config")
	assert.Equal(t, "E2E Pocket ID", imported.Get(keyAppName).AsString())
	assert.True(t, imported.Get("smtpPassword").IsNull())
}

func TestE2EGetOpenIdConfiguration(t *testing.T) {
	e := newE2E(t)
	resp, err := e.prov.Invoke(p.InvokeRequest{Token: "pocket-id:index:getOpenIdConfiguration", Args: property.NewMap(nil)})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Return.Get("issuer").AsString())
	assert.True(t, strings.HasSuffix(resp.Return.Get("jwksUri").AsString(), "/.well-known/jwks.json"))
	assert.NotEmpty(t, resp.Return.Get("tokenEndpoint").AsString())
}

// e2ePNG returns a small valid PNG filled with c.
func e2ePNG(t *testing.T, size int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for x := range size {
		for y := range size {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// rawBytes GETs an API path and returns the status and the raw body.
func (e *e2eEnv) rawBytes(path string) (int, []byte) {
	e.t.Helper()
	req, err := http.NewRequest(http.MethodGet, e.baseURL+path, nil)
	require.NoError(e.t, err)
	req.Header.Set("X-API-Key", e.apiKey)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(e.t, err)
	defer func() { _ = resp.Body.Close() }()
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(e.t, err)
	return resp.StatusCode, buf.Bytes()
}

func TestE2EApplicationImage(t *testing.T) {
	e := newE2E(t)
	red, blue := e2ePNG(t, 8, color.RGBA{R: 255, A: 255}), e2ePNG(t, 8, color.RGBA{B: 255, A: 255})

	// The background can be deleted, so the instance is left as it was found.
	img := imgAsset(t, "bg.png", red)
	in := e2eProps{keyType: property.New("background"), keyImage: img}
	bg := e.createOrSkip("ApplicationImage", in)
	assert.Equal(t, "background", bg.id)
	bg.props = asState(bg.props, img)

	status, body := e.rawBytes("/api/application-images/background")
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, red, body)
	assert.Equal(t, sum(red), e.read(bg, in).Get(keySHA256).AsString())

	img = imgAsset(t, "bg2.png", blue)
	in[keyImage] = img
	e.update(bg, in)
	bg.props = asState(bg.props, img)
	assert.Equal(t, sum(blue), e.read(bg, in).Get(keySHA256).AsString())
	assert.Equal(t, "background", e.importRead("ApplicationImage", "background").Get(keyType).AsString())
	e.destroy(bg)
}

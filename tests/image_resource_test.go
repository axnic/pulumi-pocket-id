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
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/asset"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

const (
	keyImage  = "image"
	keyType   = "type"
	keyDark   = "dark"
	keySHA256 = "sha256"

	imgData        = "data"
	imgFilename    = "filename"
	imgContentType = "contentType"
	imgFavicon     = "favicon"
	imgBackground  = "background"
	imgEmail       = "email"
)

var (
	imgPNG  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06")
	imgPNG2 = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x02\x00\x00\x00\x02\x08\x06\x00\x00\x00rest-of-file")
	imgSVG  = []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"/>")
)

// imgAsset writes data to a temp file and returns it as a FileAsset property.
func imgAsset(t *testing.T, name string, data []byte) property.Value {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, data, 0o600))
	a, err := asset.FromPath(path)
	require.NoError(t, err)
	return property.New(a)
}

func imgEnv(t *testing.T) (*fakePocketID, integration.Server) {
	t.Helper()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)
	return fake, prov
}

// asState re-types the image of a returned state as an asset. The in-process
// test harness hands back the wire form of the asset (an object carrying the
// asset signature), which the real engine turns back into an asset.
func asState(m property.Map, img property.Value) property.Map { return m.Set(keyImage, img) }

// imgRes drives one resource type through the provider with short calls.
type imgRes struct {
	t    *testing.T
	prov integration.Server
	urn  resource.URN
}

func (r imgRes) create(in property.Map, dry bool) p.CreateResponse {
	r.t.Helper()
	resp, err := r.prov.Create(p.CreateRequest{Urn: r.urn, Properties: in, DryRun: dry})
	require.NoError(r.t, err)
	return resp
}

func (r imgRes) read(id string, state, in property.Map) p.ReadResponse {
	r.t.Helper()
	resp, err := r.prov.Read(p.ReadRequest{ID: id, Urn: r.urn, Properties: state, Inputs: in})
	require.NoError(r.t, err)
	return resp
}

func (r imgRes) diff(id string, state, in property.Map) p.DiffResponse {
	r.t.Helper()
	resp, err := r.prov.Diff(p.DiffRequest{ID: id, Urn: r.urn, State: state, Inputs: in})
	require.NoError(r.t, err)
	return resp
}

func (r imgRes) update(id string, state, in property.Map) p.UpdateResponse {
	r.t.Helper()
	resp, err := r.prov.Update(p.UpdateRequest{ID: id, Urn: r.urn, State: state, Inputs: in})
	require.NoError(r.t, err)
	return resp
}

func (r imgRes) del(id string, state property.Map) {
	r.t.Helper()
	require.NoError(r.t, r.prov.Delete(p.DeleteRequest{ID: id, Urn: r.urn, Properties: state}))
}

func sum(data []byte) string { return sha256Hex(data) }

func TestApplicationImageLifecycle(t *testing.T) {
	t.Parallel()
	fake, prov := imgEnv(t)
	r := imgRes{t: t, prov: prov, urn: urn("ApplicationImage")}

	for typ, key := range map[string]string{
		"logo": "app/logo/light", "darkLogo": "app/logo/dark", imgFavicon: "app/favicon",
		imgBackground: "app/background", imgEmail: "app/email",
		"defaultProfilePicture": "app/default-profile-picture",
	} {
		img := imgAsset(t, "i.png", imgPNG)
		in := property.NewMap(map[string]property.Value{keyType: property.New(typ), keyImage: img})
		created := r.create(in, false)
		created.Properties = asState(created.Properties, img)
		assert.Equal(t, typ, created.ID)
		require.NotNil(t, fake.fakeImage(key), typ)
		assert.Equal(t, imgPNG, fake.fakeImage(key)[imgData], typ)

		// Drift detection, update, import.
		fake.setFakeImage(key, imgPNG2)
		read := r.read(typ, created.Properties, in)
		assert.Equal(t, sum(imgPNG2), read.Properties.Get(keySHA256).AsString(), typ)
		diff := r.diff(typ, asState(read.Properties, img), in)
		assert.Equal(t, p.Update, diff.DetailedDiff[keyImage].Kind, typ)
		upd := r.update(typ, asState(read.Properties, img), in)
		upd.Properties = asState(upd.Properties, img)
		assert.Equal(t, sum(imgPNG), upd.Properties.Get(keySHA256).AsString(), typ)
		assert.Equal(t, imgPNG, fake.fakeImage(key)[imgData], typ)

		imp := r.read(typ, property.NewMap(nil), property.NewMap(nil))
		assert.Equal(t, typ, imp.Inputs.Get(keyType).AsString())

		// A type change replaces.
		other := imgBackground
		if typ == other {
			other = imgFavicon
		}
		diff = r.diff(typ, upd.Properties, in.Set(keyType, property.New(other)))
		assert.Equal(t, p.UpdateReplace, diff.DetailedDiff[keyType].Kind, typ)

		// Delete: real delete when the API allows it, documented no-op for favicon and email.
		r.del(typ, upd.Properties)
		if typ == imgFavicon || typ == imgEmail {
			assert.NotNil(t, fake.fakeImage(key), typ)
			continue
		}
		assert.Nil(t, fake.fakeImage(key), typ)
		r.del(typ, upd.Properties)
		assert.Empty(t, r.read(typ, upd.Properties, in).ID, typ)
	}
}

func TestApplicationImagePreviewMakesNoAPICall(t *testing.T) {
	t.Parallel()
	prov := testServer(t)
	configure(t, prov, "http://127.0.0.1:1", "k")
	r := imgRes{t: t, prov: prov, urn: urn("ApplicationImage")}
	in := property.NewMap(map[string]property.Value{
		keyType: property.New(imgFavicon), keyImage: imgAsset(t, "f.svg", imgSVG),
	})
	assert.Equal(t, imgFavicon, r.create(in, true).Properties.Get(keyType).AsString())
}

func sha256Hex(data []byte) string {
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:])
}

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

package pocketidclient

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06")
	svgBytes = []byte("<?xml version=\"1.0\"?><svg xmlns=\"http://www.w3.org/2000/svg\"/>")
)

type upload struct {
	method, path, query, field, filename, contentType string
	data                                              []byte
}

// uploadRecorder serves status and records the multipart upload it receives.
func uploadRecorder(t *testing.T, status int) (*Client, *upload) {
	t.Helper()
	u := &upload{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "k", r.Header.Get("X-API-Key"))
		u.method, u.path, u.query = r.Method, r.URL.EscapedPath(), r.URL.RawQuery
		if mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err == nil && mt == "multipart/form-data" {
			part, err := multipart.NewReader(r.Body, params["boundary"]).NextPart()
			require.NoError(t, err)
			u.field, u.filename, u.contentType = part.FormName(), part.FileName(), part.Header.Get("Content-Type")
			u.data, _ = io.ReadAll(part)
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL, "k", nil), u
}

func TestDetectContentType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "image/png", DetectContentType(pngBytes))
	assert.Equal(t, "image/svg+xml", DetectContentType(svgBytes))
	assert.Equal(t, "text/plain; charset=utf-8", DetectContentType([]byte("hello")))
}

func TestImageFilename(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "logo.png", ImageFilename("/a/b/logo.png", pngBytes))
	assert.Equal(t, "x.jpeg", ImageFilename("x.jpeg", pngBytes))
	assert.Equal(t, "image.png", ImageFilename("blob", pngBytes))
	assert.Equal(t, "image.svg", ImageFilename("", svgBytes))
	assert.Equal(t, "image", ImageFilename("", []byte("nope")))
}

func TestSetOidcClientLogo(t *testing.T) {
	t.Parallel()
	c, u := uploadRecorder(t, 204)
	require.NoError(t, c.SetOidcClientLogo(context.Background(), "app", false, "l.png", pngBytes))
	assert.Equal(t, "POST /api/oidc/clients/app/logo light=false", u.method+" "+u.path+" "+u.query)
	assert.Equal(t, "file", u.field)
	assert.Equal(t, "l.png", u.filename)
	assert.Equal(t, "image/png", u.contentType)
	assert.Equal(t, pngBytes, u.data)
}

func TestSetApplicationImage(t *testing.T) {
	t.Parallel()
	for kind, want := range map[string]string{
		ImageLogo:                  "/api/application-images/logo light=true",
		ImageDarkLogo:              "/api/application-images/logo light=false",
		ImageFavicon:               "/api/application-images/favicon ",
		ImageBackground:            "/api/application-images/background ",
		ImageEmail:                 "/api/application-images/email ",
		ImageDefaultProfilePicture: "/api/application-images/default-profile-picture ",
	} {
		c, u := uploadRecorder(t, 204)
		require.NoError(t, c.SetApplicationImage(context.Background(), kind, "i.svg", svgBytes), kind)
		assert.Equal(t, "PUT "+want, u.method+" "+u.path+" "+u.query, kind)
		assert.Equal(t, "image/svg+xml", u.contentType)
	}
	c, _ := uploadRecorder(t, 204)
	require.Error(t, c.SetApplicationImage(context.Background(), "bogus", "i.png", pngBytes))
}

func TestGetBytes(t *testing.T) {
	t.Parallel()
	c, s := recorder(t, 200, "RAW")
	got, err := c.GetOidcClientLogo(context.Background(), "app", true)
	require.NoError(t, err)
	assert.Equal(t, []byte("RAW"), got)
	assert.Equal(t, "GET /api/oidc/clients/app/logo", s.method+" "+s.path)

	got, err = c.GetApplicationImage(context.Background(), ImageFavicon)
	require.NoError(t, err)
	assert.Equal(t, []byte("RAW"), got)
	assert.Equal(t, "/api/application-images/favicon", s.path)

	c, _ = recorder(t, 404, `{"error":"nope"}`)
	_, err = c.GetApplicationImage(context.Background(), ImageLogo)
	assert.True(t, IsNotFound(err))
}

func TestDeleteImages(t *testing.T) {
	t.Parallel()
	c, s := recorder(t, 204, "")
	require.NoError(t, c.DeleteOidcClientLogo(context.Background(), "app", true))
	assert.Equal(t, "DELETE /api/oidc/clients/app/logo", s.method+" "+s.path)
	require.NoError(t, c.DeleteApplicationImage(context.Background(), ImageBackground))
	assert.Equal(t, "DELETE /api/application-images/background", s.method+" "+s.path)

	assert.True(t, CanDeleteApplicationImage(ImageLogo))
	assert.False(t, CanDeleteApplicationImage(ImageFavicon))
	assert.False(t, CanDeleteApplicationImage(ImageEmail))
	require.Error(t, c.DeleteApplicationImage(context.Background(), ImageFavicon))
}

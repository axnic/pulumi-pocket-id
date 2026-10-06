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
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// Image domain: the "images" store maps a key ("oidc/<client>/light",
// "app/logo/dark", "app/favicon"...) to {imgData: []byte, imgFilename: string,
// imgContentType: string}. Endpoints mirror the swagger (POST/PUT multipart
// "file", GET octet-stream, DELETE 204).
func init() {
	fakeRegistrars = append(fakeRegistrars, func(f *fakePocketID) {
		images := f.store("images")

		// light reads the ?light=bool query (default true).
		light := func(q url.Values) string {
			if v, err := strconv.ParseBool(q.Get("light")); err == nil && !v {
				return "dark"
			}
			return "light"
		}
		store := func(w http.ResponseWriter, r *http.Request, key string) {
			file, hdr, err := r.FormFile("file")
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{keyError: "file is required"})
				return
			}
			defer func() { _ = file.Close() }()
			data, _ := io.ReadAll(file)
			images[key] = map[string]any{
				imgData: data, imgFilename: hdr.Filename, imgContentType: hdr.Header.Get("Content-Type"),
			}
			w.WriteHeader(http.StatusNoContent)
		}
		get := func(w http.ResponseWriter, key string) {
			img, ok := images[key]
			if !ok {
				notFound(w)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(img[imgData].([]byte))
		}
		del := func(w http.ResponseWriter, key string) {
			if _, ok := images[key]; !ok {
				notFound(w)
				return
			}
			delete(images, key)
			w.WriteHeader(http.StatusNoContent)
		}

		clientKey := func(r *http.Request) string {
			return "oidc/" + r.PathValue("id") + "/" + light(r.URL.Query())
		}
		f.handle("POST /api/oidc/clients/{id}/logo", func(w http.ResponseWriter, r *http.Request) {
			store(w, r, clientKey(r))
		})
		f.handle("GET /api/oidc/clients/{id}/logo", func(w http.ResponseWriter, r *http.Request) { get(w, clientKey(r)) })
		f.handle("DELETE /api/oidc/clients/{id}/logo", func(w http.ResponseWriter, r *http.Request) { del(w, clientKey(r)) })

		logoKey := func(r *http.Request) string { return "app/logo/" + light(r.URL.Query()) }
		f.handle("PUT /api/application-images/logo", func(w http.ResponseWriter, r *http.Request) { store(w, r, logoKey(r)) })
		f.handle("GET /api/application-images/logo", func(w http.ResponseWriter, r *http.Request) { get(w, logoKey(r)) })
		f.handle("DELETE /api/application-images/logo", func(w http.ResponseWriter, r *http.Request) { del(w, logoKey(r)) })

		// favicon and email: no DELETE in the API.
		for _, name := range []string{imgFavicon, imgEmail} {
			key := "app/" + name
			f.handle("PUT /api/application-images/"+name, func(w http.ResponseWriter, r *http.Request) { store(w, r, key) })
			f.handle("GET /api/application-images/"+name, func(w http.ResponseWriter, _ *http.Request) { get(w, key) })
		}
		for _, name := range []string{imgBackground, "default-profile-picture"} {
			key := "app/" + name
			f.handle("PUT /api/application-images/"+name, func(w http.ResponseWriter, r *http.Request) { store(w, r, key) })
			f.handle("GET /api/application-images/"+name, func(w http.ResponseWriter, _ *http.Request) { get(w, key) })
			f.handle("DELETE /api/application-images/"+name, func(w http.ResponseWriter, _ *http.Request) { del(w, key) })
		}
	})
}

// fakeImage returns the stored image under key (nil when absent).
func (f *fakePocketID) fakeImage(key string) map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.store("images")[key]
}

// setFakeImage overwrites the bytes stored under key, as an out-of-band change would.
func (f *fakePocketID) setFakeImage(key string, data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store("images")[key] = map[string]any{imgData: data, imgFilename: "x.png", imgContentType: "image/png"}
}

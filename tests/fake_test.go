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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/blang/semver"
	"github.com/stretchr/testify/require"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"

	"github.com/axnic/pulumi-pocket-id/provider"
)

// fakePocketID is an in-memory stand-in for the Pocket-ID REST API. This file
// holds only the core (routing, auth, locking, helpers); each API domain adds
// its endpoints from its own fake_<domain>_test.go by appending to
// fakeRegistrars in an init() function, so domains never edit this file.
type fakePocketID struct {
	mu     sync.Mutex
	apiKey string
	nextID int
	mux    *http.ServeMux

	// Per-domain in-memory stores, keyed by domain name (e.g. "users").
	// Handlers run under f.mu, so they may read/write them freely.
	stores map[string]map[string]map[string]any
}

// fakeRegistrars is appended to by each fake_<domain>_test.go.
var fakeRegistrars []func(f *fakePocketID)

func newFake(apiKey string) *fakePocketID {
	f := &fakePocketID{apiKey: apiKey, mux: http.NewServeMux(), stores: map[string]map[string]map[string]any{}}
	for _, register := range fakeRegistrars {
		register(f)
	}
	return f
}

// store returns (creating it if needed) the named in-memory collection.
func (f *fakePocketID) store(name string) map[string]map[string]any {
	if f.stores[name] == nil {
		f.stores[name] = map[string]map[string]any{}
	}
	return f.stores[name]
}

// handle registers a Go 1.22 mux pattern (e.g. "GET /api/users/{id}"). The
// handler runs with the API key checked and f.mu held.
func (f *fakePocketID) handle(pattern string, h func(w http.ResponseWriter, r *http.Request)) {
	f.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != f.apiKey {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		h(w, r)
	})
}

func (f *fakePocketID) handler() http.Handler { return f.mux }

// genID returns a unique ID with the given prefix, e.g. "usr-1".
func (f *fakePocketID) genID(prefix string) string {
	f.nextID++
	return prefix + strconv.Itoa(f.nextID)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func notFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]string{keyError: "record not found"})
}

// readBody decodes the request's JSON object body.
func readBody(r *http.Request) map[string]any {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body == nil {
		body = map[string]any{}
	}
	return body
}

// paginated writes items in Pocket-ID's {"data": [...], "pagination": {...}} envelope (single page).
func paginated(w http.ResponseWriter, items []any) {
	if items == nil {
		items = []any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"pagination": map[string]any{
			"currentPage": 1, "itemsPerPage": 100, "totalItems": len(items), "totalPages": 1,
		},
	})
}

// --- integration helpers ---

func urn(typ string) resource.URN {
	return resource.NewURN("stack", "proj", "",
		tokens.Type("pocket-id:index:"+typ), keyName)
}

// testServer builds an integration server running the real provider.
func testServer(t *testing.T) integration.Server {
	t.Helper()
	s, err := integration.NewServer(
		context.Background(),
		provider.Name,
		semver.MustParse("1.0.0"),
		integration.WithProvider(provider.Provider()),
	)
	require.NoError(t, err)
	return s
}

// configure seeds the provider config so resource methods can reach the API.
func configure(t *testing.T, srv integration.Server, baseURL, apiKey string) {
	t.Helper()
	err := srv.Configure(p.ConfigureRequest{
		Args: property.NewMap(map[string]property.Value{
			"baseUrl": property.New(baseURL),
			keyAPIKey: property.New(apiKey),
		}),
	})
	require.NoError(t, err)
}

// newFakeServer returns a running httptest server backed by a fakePocketID.
func newFakeServer(t *testing.T, apiKey string) (*fakePocketID, *httptest.Server) {
	t.Helper()
	fake := newFake(apiKey)
	srv := httptest.NewServer(fake.handler())
	t.Cleanup(srv.Close)
	return fake, srv
}

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
	"strings"
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

// fakePocketID is an in-memory stand-in for the Pocket-ID REST API. It supports
// the subset of endpoints exercised by the provider: user-groups, users, OIDC
// clients, and custom claims.
type fakePocketID struct {
	mu          sync.Mutex
	apiKey      string
	nextID      int
	userGroups  map[string]map[string]any
	users       map[string]map[string]any
	clients     map[string]map[string]any
	clientOrder []string // preserves insertion order for clients
}

func newFake(apiKey string) *fakePocketID {
	return &fakePocketID{
		apiKey:     apiKey,
		userGroups: map[string]map[string]any{},
		users:      map[string]map[string]any{},
		clients:    map[string]map[string]any{},
	}
}

func (f *fakePocketID) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/user-groups", f.serveUserGroups)
	mux.HandleFunc("/api/user-groups/", f.serveUserGroup)
	mux.HandleFunc("/api/users", f.serveUsers)
	mux.HandleFunc("/api/users/", f.serveUser)
	mux.HandleFunc("/api/oidc/clients", f.serveClients)
	mux.HandleFunc("/api/oidc/clients/", f.serveClient)
	mux.HandleFunc("/api/custom-claims/user-group/", f.serveUserGroupClaims)
	mux.HandleFunc("/api/custom-claims/user/", f.serveUserClaims)
	return mux
}

func (f *fakePocketID) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != f.apiKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		next(w, r)
	}
}

func (f *fakePocketID) genID(prefix string) string {
	f.nextID++
	return prefix + itoa(f.nextID)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// --- user groups ---

func (f *fakePocketID) serveUserGroups(w http.ResponseWriter, r *http.Request) {
	f.auth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		id := f.genID("grp-")
		body["id"] = id
		body["customClaims"] = []any{}
		f.userGroups[id] = body
		writeJSON(w, http.StatusOK, body)
	})(w, r)
}

func (f *fakePocketID) serveUserGroup(w http.ResponseWriter, r *http.Request) {
	f.auth(func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/user-groups/")
		id := rest
		if idx := strings.Index(rest, "/"); idx >= 0 {
			id = rest[:idx]
		}
		body, ok := f.userGroups[id]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, body)
		case http.MethodPut:
			var in map[string]any
			_ = json.NewDecoder(r.Body).Decode(&in)
			in["id"] = id
			in["customClaims"] = body["customClaims"]
			f.userGroups[id] = in
			writeJSON(w, http.StatusOK, in)
		case http.MethodDelete:
			delete(f.userGroups, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})(w, r)
}

// --- users ---

func (f *fakePocketID) serveUsers(w http.ResponseWriter, r *http.Request) {
	f.auth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		id := f.genID("usr-")
		body["id"] = id
		if body["userGroups"] == nil {
			body["userGroups"] = []any{}
		}
		body["customClaims"] = []any{}
		f.users[id] = body
		writeJSON(w, http.StatusOK, body)
	})(w, r)
}

func (f *fakePocketID) serveUser(w http.ResponseWriter, r *http.Request) {
	f.auth(func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/users/")
		id := rest
		if idx := strings.Index(rest, "/"); idx >= 0 {
			id = rest[:idx]
		}
		body, ok := f.users[id]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, body)
		case http.MethodPut:
			var in map[string]any
			_ = json.NewDecoder(r.Body).Decode(&in)
			in["id"] = id
			in["userGroups"] = body["userGroups"]
			in["customClaims"] = body["customClaims"]
			f.users[id] = in
			writeJSON(w, http.StatusOK, in)
		case http.MethodDelete:
			delete(f.users, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})(w, r)
}

// --- oidc clients ---

func (f *fakePocketID) serveClients(w http.ResponseWriter, r *http.Request) {
	f.auth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		id, _ := body["id"].(string)
		if id == "" {
			id = f.genID("cli-")
		}
		body["id"] = id
		body["pkceSupported"] = true
		f.clients[id] = body
		f.clientOrder = append(f.clientOrder, id)
		writeJSON(w, http.StatusOK, body)
	})(w, r)
}

func (f *fakePocketID) serveClient(w http.ResponseWriter, r *http.Request) {
	f.auth(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/oidc/clients/")
		body, ok := f.clients[id]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, body)
		case http.MethodPut:
			var in map[string]any
			_ = json.NewDecoder(r.Body).Decode(&in)
			in["id"] = id
			in["pkceSupported"] = true
			f.clients[id] = in
			writeJSON(w, http.StatusOK, in)
		case http.MethodDelete:
			delete(f.clients, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})(w, r)
}

// --- custom claims ---

func (f *fakePocketID) serveUserGroupClaims(w http.ResponseWriter, r *http.Request) {
	f.auth(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/custom-claims/user-group/")
		body, ok := f.userGroups[id]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var claims []any
		_ = json.NewDecoder(r.Body).Decode(&claims)
		body["customClaims"] = claims
		w.WriteHeader(http.StatusNoContent)
	})(w, r)
}

func (f *fakePocketID) serveUserClaims(w http.ResponseWriter, r *http.Request) {
	f.auth(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/custom-claims/user/")
		body, ok := f.users[id]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var claims []any
		_ = json.NewDecoder(r.Body).Decode(&claims)
		body["customClaims"] = claims
		w.WriteHeader(http.StatusNoContent)
	})(w, r)
}

// --- integration helpers ---

func urn(typ string) resource.URN {
	return resource.NewURN("stack", "proj", "",
		tokens.Type("pocketid:index:"+typ), "name")
}

// testServer builds an integration server wired against a fake Pocket-ID instance.
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

// configure seeds the provider config so resource methods can read it via infer.GetConfig.
func configure(t *testing.T, srv integration.Server, baseURL, apiKey string) {
	t.Helper()
	err := srv.Configure(p.ConfigureRequest{
		Args: property.NewMap(map[string]property.Value{
			"baseUrl": property.New(baseURL),
			"apiKey":  property.New(apiKey),
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

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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoSendsAPIKeyAndDecodes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "k", r.Header.Get("X-API-Key"))
		assert.Equal(t, "/api/x", r.URL.Path)
		_, _ = w.Write([]byte(`{"a":"b"}`))
	}))
	defer srv.Close()

	var out map[string]string
	require.NoError(t, New(srv.URL+"/", "k", nil).Do(context.Background(), "GET", "/api/x", nil, nil, &out))
	assert.Equal(t, "b", out["a"])
}

func TestDoMapsErrors(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	}))
	defer srv.Close()

	err := New(srv.URL, "k", nil).Do(context.Background(), "GET", "/api/x", nil, nil, nil)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
	assert.Contains(t, err.Error(), "nope")
}

func TestListAllPaginates(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "100", r.URL.Query().Get("pagination[limit]"))
		assert.Equal(t, "x", r.URL.Query().Get("search"))
		if r.URL.Query().Get("pagination[page]") == "1" {
			_, _ = w.Write([]byte(`{"data":[1,2],"pagination":{"currentPage":1,"totalPages":2}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[3],"pagination":{"currentPage":2,"totalPages":2}}`))
	}))
	defer srv.Close()

	got, err := listAll[int](context.Background(), New(srv.URL, "k", nil), "/api/x", map[string][]string{"search": {"x"}})
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, got)
}

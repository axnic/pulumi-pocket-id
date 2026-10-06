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
package pocketidclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAppConfig(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/application-configuration/all", r.URL.Path)
		assert.Equal(t, "k", r.Header.Get("X-API-Key"))
		_, _ = w.Write([]byte(`[{"key":"appName","type":"string","value":"Pocket ` +
			`ID","isPublic":true},{"key":"smtpPort","value":"587"}]`))
	}))
	defer srv.Close()

	got, err := New(srv.URL, "k", nil).GetAppConfig(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]string{keyAppName: "Pocket ID", "smtpPort": "587"}, got)
}

func TestUpdateAppConfig(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/api/application-configuration", r.URL.Path)
		assert.Equal(t, "k", r.Header.Get("X-API-Key"))
		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]string{keyAppName: "Acme"}, body)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	require.NoError(t, New(srv.URL, "k", nil).UpdateAppConfig(context.Background(), map[string]string{keyAppName: "Acme"}))
}

func TestAppConfigError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := New(srv.URL, "k", nil).GetAppConfig(context.Background())
	assert.True(t, IsNotFound(err))
}

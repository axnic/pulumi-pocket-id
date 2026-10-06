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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPICRUDAndPermissions(t *testing.T) {
	t.Parallel()
	var calls []string
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "k", r.Header.Get("X-API-Key"))
		calls = append(calls, r.Method+" "+r.URL.Path)
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		bodies = append(bodies, b)
		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"a1","name":"n","resource":"https://r"}`))
		default:
			_, _ = w.Write([]byte(`{"id":"a1","name":"n","resource":"https://r",` +
				`"permissions":[{"id":"p1","key":"read","name":"Read"}]}`))
		}
	}))
	defer srv.Close()
	c := New(srv.URL, "k", nil)
	ctx := context.Background()

	a, err := c.CreateAPI(ctx, "n", "https://r")
	require.NoError(t, err)
	assert.Equal(t, "a1", a.ID)
	_, err = c.GetAPI(ctx, "a1")
	require.NoError(t, err)
	_, err = c.UpdateAPI(ctx, "a1", "n2")
	require.NoError(t, err)
	a, err = c.SetAPIPermissions(ctx, "a1", []APIPermissionInput{{Key: "read", Name: "Read"}})
	require.NoError(t, err)
	assert.Equal(t, "p1", a.Permissions[0].ID)
	require.NoError(t, c.SetAPICimdAccess(ctx, "a1", true, nil))
	require.NoError(t, c.DeleteAPI(ctx, "a1"))

	assert.Equal(t, []string{
		"POST /api/apis", "GET /api/apis/a1", "PUT /api/apis/a1", "PUT /api/apis/a1/permissions",
		"PUT /api/apis/a1/cimd-access", "DELETE /api/apis/a1",
	}, calls)
	assert.Equal(t, map[string]any{"enabled": true, "permissionIds": []any{}}, bodies[4])
	assert.Equal(t, map[string]any{keyName: "n", "resource": "https://r"}, bodies[0])
	assert.Equal(t, map[string]any{keyName: "n2"}, bodies[2])
	assert.Equal(t, map[string]any{"permissions": []any{map[string]any{"key": "read", keyName: "Read"}}}, bodies[3])
}

func TestAPIClientGrant(t *testing.T) {
	t.Parallel()
	var calls []string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[{"client":{"id":"c1"},"clientAccess":true,"clientPermissionIds":["p1"],` +
				`"userDelegatedAccess":false}],"pagination":{"currentPage":1,"totalPages":1}}`))
		case http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&body)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	c := New(srv.URL, "k", nil)
	ctx := context.Background()

	list, err := c.ListAPIClients(ctx, "a1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "c1", list[0].Client.ID)
	assert.True(t, list[0].ClientAccess)
	assert.Equal(t, []string{"p1"}, list[0].ClientPermissionIDs)

	require.NoError(t, c.SetAPIClientGrant(ctx, "a1", "c1", APIClientGrant{ClientAccess: true}))
	assert.Equal(t, map[string]any{
		"clientAccess": true, "clientPermissionIds": []any{},
		"userDelegatedAccess": false, "userDelegatedPermissionIds": []any{},
	}, body)

	err = c.DeleteAPIClientGrant(ctx, "a1", "c1")
	assert.True(t, IsNotFound(err))
	assert.Equal(t, []string{"GET /api/apis/a1/clients", "PUT /api/apis/a1/clients/c1",
		"DELETE /api/apis/a1/clients/c1"}, calls)
}

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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// identityCall records what the client sent for one request.
type identityCall struct {
	method, path, apiKey string
	body                 map[string]any
	rawBody              string
}

// identityServer serves a canned status/body and records the request.
func identityServer(t *testing.T, status int, resp string) (*Client, *identityCall) {
	t.Helper()
	call := &identityCall{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call.method, call.path, call.apiKey = r.Method, r.URL.Path, r.Header.Get("X-API-Key")
		b, _ := io.ReadAll(r.Body)
		call.rawBody = string(b)
		_ = json.Unmarshal(b, &call.body)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(resp))
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL, "key", nil), call
}

func TestUserCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	email := "a@example.com"

	c, call := identityServer(t, 201, `{"id":"u1","username":"alice","email":"a@example.com","firstName":"Alice"}`)
	u, err := c.CreateUser(ctx, UserInput{Username: "alice", Email: &email, IsAdmin: true})
	require.NoError(t, err)
	assert.Equal(t, "u1", u.ID)
	assert.Equal(t, "POST /api/users key", call.method+" "+call.path+" "+call.apiKey)
	assert.Equal(t, "alice", call.body["username"])
	assert.Equal(t, true, call.body["isAdmin"])
	assert.Equal(t, false, call.body["disabled"])
	assert.NotContains(t, call.body, "firstName")
	assert.NotContains(t, call.body, "userGroupIds")

	c, call = identityServer(t, 200, `{"id":"u1","username":"alice2"}`)
	u, err = c.UpdateUser(ctx, "u1", UserInput{Username: "alice2"})
	require.NoError(t, err)
	assert.Equal(t, "alice2", u.Username)
	assert.Equal(t, "PUT /api/users/u1", call.method+" "+call.path)

	c, call = identityServer(t, 200,
		`{"id":"u1","username":"alice","customClaims":[{"key":"k","value":"v"}],"userGroups":[{"id":"g1"}]}`)
	u, err = c.GetUser(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, "GET /api/users/u1", call.method+" "+call.path)
	assert.Equal(t, []CustomClaim{{"k", "v"}}, u.CustomClaims)
	assert.Equal(t, "g1", u.UserGroups[0].ID)

	c, call = identityServer(t, 204, ``)
	require.NoError(t, c.DeleteUser(ctx, "u1"))
	assert.Equal(t, "DELETE /api/users/u1", call.method+" "+call.path)
}

func TestUserNotFound(t *testing.T) {
	t.Parallel()
	c, _ := identityServer(t, 404, `{"error":"record not found"}`)
	_, err := c.GetUser(context.Background(), "nope")
	assert.True(t, IsNotFound(err))
	assert.True(t, IsNotFound(c.DeleteUser(context.Background(), "nope")))
}

func TestListUsersSearch(t *testing.T) {
	t.Parallel()
	var search string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/users", r.URL.Path)
		search = r.URL.Query().Get("search")
		_, _ = w.Write([]byte(`{"data":[{"id":"u1","username":"alice"}],"pagination":{"currentPage":1,"totalPages":1}}`))
	}))
	defer srv.Close()
	users, err := New(srv.URL, "key", nil).ListUsers(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, "alice", search)
	require.Len(t, users, 1)
	assert.Equal(t, "u1", users[0].ID)
}

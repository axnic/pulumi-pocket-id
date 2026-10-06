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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserGroupCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c, call := identityServer(t, 201, `{"id":"g1","name":"admins","friendlyName":"Admins"}`)
	g, err := c.CreateUserGroup(ctx, UserGroupInput{Name: "admins", FriendlyName: "Admins"})
	require.NoError(t, err)
	assert.Equal(t, "g1", g.ID)
	assert.Equal(t, "POST /api/user-groups key", call.method+" "+call.path+" "+call.apiKey)
	assert.Equal(t, "Admins", call.body["friendlyName"])

	c, call = identityServer(t, 200, `{"id":"g1","name":"ops","friendlyName":"Ops"}`)
	_, err = c.UpdateUserGroup(ctx, "g1", UserGroupInput{Name: "ops", FriendlyName: "Ops"})
	require.NoError(t, err)
	assert.Equal(t, "PUT /api/user-groups/g1", call.method+" "+call.path)

	c, call = identityServer(t, 200, `{"id":"g1","users":[{"id":"u1"},{"id":"u2"}]}`)
	g, err = c.GetUserGroup(ctx, "g1")
	require.NoError(t, err)
	assert.Equal(t, "GET /api/user-groups/g1", call.method+" "+call.path)
	assert.Len(t, g.Users, 2)

	c, call = identityServer(t, 204, ``)
	require.NoError(t, c.DeleteUserGroup(ctx, "g1"))
	assert.Equal(t, "DELETE /api/user-groups/g1", call.method+" "+call.path)

	c, _ = identityServer(t, 404, `{"error":"nope"}`)
	_, err = c.GetUserGroup(ctx, "x")
	assert.True(t, IsNotFound(err))
}

func TestSetUserGroupMembers(t *testing.T) {
	t.Parallel()
	c, call := identityServer(t, 200, `{"id":"g1","users":[{"id":"u1"}]}`)
	_, err := c.SetUserGroupMembers(context.Background(), "g1", []string{"u1"})
	require.NoError(t, err)
	assert.Equal(t, "PUT /api/user-groups/g1/users", call.method+" "+call.path)
	assert.Equal(t, []any{"u1"}, call.body["userIds"])

	c, call = identityServer(t, 200, `{"id":"g1"}`)
	_, err = c.SetUserGroupMembers(context.Background(), "g1", nil)
	require.NoError(t, err)
	assert.JSONEq(t, `{"userIds":[]}`, call.rawBody)
}

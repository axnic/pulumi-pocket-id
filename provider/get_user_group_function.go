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

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/axnic/pulumi-pocket-id/provider/internal/pocketidclient"
)

func init() {
	registerFunction(infer.Function[*GetUserGroup, GetUserGroupArgs, GetUserGroupResult](&GetUserGroup{}))
}

// GetUserGroup looks a user group up by ID or name.
type GetUserGroup struct{}

// GetUserGroupArgs are the arguments of getUserGroup.
type GetUserGroupArgs struct {
	ID   *string `pulumi:"userGroupId,optional"`
	Name *string `pulumi:"name,optional"`
}

// GetUserGroupResult is the result of getUserGroup.
type GetUserGroupResult struct {
	ID           string            `pulumi:"userGroupId"`
	Name         string            `pulumi:"name"`
	FriendlyName string            `pulumi:"friendlyName"`
	CustomClaims map[string]string `pulumi:"customClaims"`
	UserIDs      []string          `pulumi:"userIds"`
}

var _ infer.Annotated = (*GetUserGroupArgs)(nil)

// Annotate describes the arguments of getUserGroup.
func (a *GetUserGroupArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ID, "The ID of the group. Exactly one of userGroupId or name must be set.")
	an.Describe(&a.Name, "The exact name of the group. Exactly one of userGroupId or name must be set.")
}

var _ infer.Annotated = (*GetUserGroupResult)(nil)

// Annotate describes the result of getUserGroup.
func (r *GetUserGroupResult) Annotate(an infer.Annotator) {
	an.Describe(&r.ID, "The ID of the group.")
	an.Describe(&r.Name, "The name of the group.")
	an.Describe(&r.FriendlyName, "The human-readable name of the group.")
	an.Describe(&r.CustomClaims, "The custom claims of the group.")
	an.Describe(&r.UserIDs, "The IDs of the members of the group.")
}

var _ infer.Annotated = (*GetUserGroup)(nil)

// Annotate describes the getUserGroup function.
func (*GetUserGroup) Annotate(an infer.Annotator) {
	an.Describe(&GetUserGroup{}, "Looks up a Pocket-ID user group by ID or by exact name.")
}

// Invoke resolves the group.
func (*GetUserGroup) Invoke(
	ctx context.Context, req infer.FunctionRequest[GetUserGroupArgs],
) (infer.FunctionResponse[GetUserGroupResult], error) {
	in := req.Input
	hasID, hasName := in.ID != nil && *in.ID != "", in.Name != nil && *in.Name != ""
	if hasID == hasName {
		return infer.FunctionResponse[GetUserGroupResult]{}, errors.New("exactly one of userGroupId or name must be set")
	}
	client := clientFromContext(ctx)

	id := ""
	if hasID {
		id = *in.ID
	} else {
		groups, err := client.ListUserGroups(ctx, *in.Name)
		if err != nil {
			return infer.FunctionResponse[GetUserGroupResult]{}, err
		}
		var matches []pocketidclient.UserGroupMinimal
		for _, g := range groups {
			if g.Name == *in.Name {
				matches = append(matches, g)
			}
		}
		if len(matches) != 1 {
			return infer.FunctionResponse[GetUserGroupResult]{},
				fmt.Errorf("expected exactly one user group named %q, found %d", *in.Name, len(matches))
		}
		id = matches[0].ID
	}

	g, err := client.GetUserGroup(ctx, id)
	if err != nil {
		return infer.FunctionResponse[GetUserGroupResult]{}, fmt.Errorf("getting user group %q: %w", id, err)
	}
	users := make([]string, 0, len(g.Users))
	for _, u := range g.Users {
		users = append(users, u.ID)
	}
	return infer.FunctionResponse[GetUserGroupResult]{Output: GetUserGroupResult{
		ID: g.ID, Name: g.Name, FriendlyName: g.FriendlyName,
		CustomClaims: claimsToMap(g.CustomClaims), UserIDs: users,
	}}, nil
}

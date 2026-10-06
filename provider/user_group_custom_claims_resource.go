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

	"github.com/pulumi/pulumi-go-provider/infer"
)

func init() { registerResource(infer.Resource(&UserGroupCustomClaims{})) }

// UserGroupCustomClaims authoritatively manages the complete set of custom claims
// of one user group. Use at most one instance per user group.
type UserGroupCustomClaims struct{}

// UserGroupCustomClaimsArgs are the inputs of a UserGroupCustomClaims.
type UserGroupCustomClaimsArgs struct {
	UserGroupID string            `pulumi:"userGroupId" provider:"replaceOnChanges"`
	Claims      map[string]string `pulumi:"claims"`
}

// UserGroupCustomClaimsState is the persisted state of a UserGroupCustomClaims.
type UserGroupCustomClaimsState struct {
	UserGroupCustomClaimsArgs
}

var _ infer.Annotated = (*UserGroupCustomClaimsArgs)(nil)

// Annotate describes the inputs of a UserGroupCustomClaims.
func (a *UserGroupCustomClaimsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.UserGroupID, "The ID of the user group. Changing it replaces the resource.")
	an.Describe(&a.Claims, "The custom claims of the group, as key/value pairs. The set is authoritative: "+
		"claims not listed are removed from the group.")
}

var _ infer.Annotated = (*UserGroupCustomClaims)(nil)

// Annotate describes the UserGroupCustomClaims resource.
func (*UserGroupCustomClaims) Annotate(an infer.Annotator) {
	an.Describe(&UserGroupCustomClaims{}, "The authoritative set of custom claims of a user group. "+
		"Destroying the resource removes every custom claim from the group.")
}

// Create sets the claims of the group.
func (*UserGroupCustomClaims) Create(
	ctx context.Context, req infer.CreateRequest[UserGroupCustomClaimsArgs],
) (infer.CreateResponse[UserGroupCustomClaimsState], error) {
	state := UserGroupCustomClaimsState{UserGroupCustomClaimsArgs: req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[UserGroupCustomClaimsState]{ID: req.Inputs.UserGroupID, Output: state}, nil
	}
	_, err := clientFromContext(ctx).SetUserGroupCustomClaims(ctx, req.Inputs.UserGroupID, claimsToList(req.Inputs.Claims))
	if err != nil {
		return infer.CreateResponse[UserGroupCustomClaimsState]{}, err
	}
	return infer.CreateResponse[UserGroupCustomClaimsState]{ID: req.Inputs.UserGroupID, Output: state}, nil
}

// Read refreshes the claims from the group; a missing group is dropped from state.
func (*UserGroupCustomClaims) Read(
	ctx context.Context, req infer.ReadRequest[UserGroupCustomClaimsArgs, UserGroupCustomClaimsState],
) (infer.ReadResponse[UserGroupCustomClaimsArgs, UserGroupCustomClaimsState], error) {
	g, err := clientFromContext(ctx).GetUserGroup(ctx, req.ID)
	if isGone(err) {
		return infer.ReadResponse[UserGroupCustomClaimsArgs, UserGroupCustomClaimsState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[UserGroupCustomClaimsArgs, UserGroupCustomClaimsState]{}, err
	}
	args := UserGroupCustomClaimsArgs{UserGroupID: g.ID, Claims: claimsToMap(g.CustomClaims)}
	return infer.ReadResponse[UserGroupCustomClaimsArgs, UserGroupCustomClaimsState]{
		ID: g.ID, Inputs: args, State: UserGroupCustomClaimsState{UserGroupCustomClaimsArgs: args},
	}, nil
}

// Update sets the claims of the group.
func (*UserGroupCustomClaims) Update(
	ctx context.Context, req infer.UpdateRequest[UserGroupCustomClaimsArgs, UserGroupCustomClaimsState],
) (infer.UpdateResponse[UserGroupCustomClaimsState], error) {
	state := UserGroupCustomClaimsState{UserGroupCustomClaimsArgs: req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[UserGroupCustomClaimsState]{Output: state}, nil
	}
	if _, err := clientFromContext(ctx).SetUserGroupCustomClaims(ctx, req.ID,
		claimsToList(req.Inputs.Claims)); err != nil {
		return infer.UpdateResponse[UserGroupCustomClaimsState]{}, err
	}
	return infer.UpdateResponse[UserGroupCustomClaimsState]{Output: state}, nil
}

// Delete clears the claims of the group.
func (*UserGroupCustomClaims) Delete(
	ctx context.Context, req infer.DeleteRequest[UserGroupCustomClaimsState],
) (infer.DeleteResponse, error) {
	if _, err := clientFromContext(ctx).SetUserGroupCustomClaims(ctx, req.ID, nil); err != nil && !isGone(err) {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

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

	"github.com/axnic/pulumi-pocket-id/provider/internal/pocketidclient"
)

func init() { registerResource(infer.Resource(&UserGroup{})) }

// UserGroup manages a Pocket-ID user group. Members are managed with
// UserGroupMembers and custom claims with UserGroupCustomClaims.
type UserGroup struct{}

// UserGroupArgs are the inputs of a UserGroup.
type UserGroupArgs struct {
	Name         string `pulumi:"name"`
	FriendlyName string `pulumi:"friendlyName"`
}

// UserGroupState is the persisted state of a UserGroup.
type UserGroupState struct {
	UserGroupArgs
}

var _ infer.Annotated = (*UserGroupArgs)(nil)

// Annotate describes the inputs of a UserGroup.
func (a *UserGroupArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Name, "The unique technical name of the group.")
	an.Describe(&a.FriendlyName, "The human-readable name of the group.")
}

var _ infer.Annotated = (*UserGroup)(nil)

// Annotate describes the UserGroup resource.
func (*UserGroup) Annotate(an infer.Annotator) {
	an.Describe(&UserGroup{}, "A Pocket-ID user group. Members are managed with the UserGroupMembers resource "+
		"and custom claims with the UserGroupCustomClaims resource.")
}

func (a UserGroupArgs) input() pocketidclient.UserGroupInput {
	return pocketidclient.UserGroupInput{Name: a.Name, FriendlyName: a.FriendlyName}
}

// Create creates the group.
func (*UserGroup) Create(
	ctx context.Context, req infer.CreateRequest[UserGroupArgs],
) (infer.CreateResponse[UserGroupState], error) {
	state := UserGroupState{UserGroupArgs: req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[UserGroupState]{ID: req.Name, Output: state}, nil
	}
	g, err := clientFromContext(ctx).CreateUserGroup(ctx, req.Inputs.input())
	if err != nil {
		return infer.CreateResponse[UserGroupState]{}, err
	}
	return infer.CreateResponse[UserGroupState]{ID: g.ID, Output: state}, nil
}

// Read refreshes the group from the API; a missing group is dropped from state.
func (*UserGroup) Read(
	ctx context.Context, req infer.ReadRequest[UserGroupArgs, UserGroupState],
) (infer.ReadResponse[UserGroupArgs, UserGroupState], error) {
	g, err := clientFromContext(ctx).GetUserGroup(ctx, req.ID)
	if isGone(err) {
		return infer.ReadResponse[UserGroupArgs, UserGroupState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[UserGroupArgs, UserGroupState]{}, err
	}
	args := UserGroupArgs{Name: g.Name, FriendlyName: g.FriendlyName}
	return infer.ReadResponse[UserGroupArgs, UserGroupState]{
		ID: g.ID, Inputs: args, State: UserGroupState{UserGroupArgs: args},
	}, nil
}

// Update changes the group's names.
func (*UserGroup) Update(
	ctx context.Context, req infer.UpdateRequest[UserGroupArgs, UserGroupState],
) (infer.UpdateResponse[UserGroupState], error) {
	state := UserGroupState{UserGroupArgs: req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[UserGroupState]{Output: state}, nil
	}
	if _, err := clientFromContext(ctx).UpdateUserGroup(ctx, req.ID, req.Inputs.input()); err != nil {
		return infer.UpdateResponse[UserGroupState]{}, err
	}
	return infer.UpdateResponse[UserGroupState]{Output: state}, nil
}

// Delete removes the group.
func (*UserGroup) Delete(ctx context.Context, req infer.DeleteRequest[UserGroupState]) (infer.DeleteResponse, error) {
	if err := clientFromContext(ctx).DeleteUserGroup(ctx, req.ID); err != nil && !isGone(err) {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

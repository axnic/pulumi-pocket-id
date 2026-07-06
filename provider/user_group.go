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

// UserGroup manages a Pocket-ID user group.
type UserGroup struct{}

// UserGroupArgs are the inputs for a user group.
type UserGroupArgs struct {
	FriendlyName string `pulumi:"friendlyName" json:"friendlyName"`
	Name         string `pulumi:"name" json:"name"`
}

// UserGroupState is persisted in Pulumi state.
type UserGroupState struct {
	UserGroupArgs
}

// userGroupResponse mirrors the UserGroupDto response.
type userGroupResponse struct {
	Id           string `json:"id"`
	FriendlyName string `json:"friendlyName"`
	Name         string `json:"name"`
}

func (resp userGroupResponse) toState() UserGroupState {
	return UserGroupState{
		UserGroupArgs: UserGroupArgs{
			FriendlyName: resp.FriendlyName,
			Name:         resp.Name,
		},
	}
}

// Create creates a new user group.
func (*UserGroup) Create(
	ctx context.Context,
	req infer.CreateRequest[UserGroupArgs],
) (infer.CreateResponse[UserGroupState], error) {
	if req.DryRun {
		return infer.CreateResponse[UserGroupState]{
			ID:     req.Name,
			Output: UserGroupState{UserGroupArgs: req.Inputs},
		}, nil
	}

	client := clientFromContext(ctx)
	var resp userGroupResponse
	if err := client.do(ctx, "POST", "/api/user-groups", req.Inputs, &resp); err != nil {
		return infer.CreateResponse[UserGroupState]{}, err
	}
	return infer.CreateResponse[UserGroupState]{ID: resp.Id, Output: resp.toState()}, nil
}

// Read fetches the current state of a user group.
func (*UserGroup) Read(
	ctx context.Context,
	req infer.ReadRequest[UserGroupArgs, UserGroupState],
) (infer.ReadResponse[UserGroupArgs, UserGroupState], error) {
	client := clientFromContext(ctx)
	var resp userGroupResponse
	if err := client.do(ctx, "GET", "/api/user-groups/"+req.ID, nil, &resp); err != nil {
		return infer.ReadResponse[UserGroupArgs, UserGroupState]{}, err
	}
	state := resp.toState()
	return infer.ReadResponse[UserGroupArgs, UserGroupState]{ID: resp.Id, Inputs: state.UserGroupArgs, State: state}, nil
}

// Update modifies an existing user group.
func (*UserGroup) Update(
	ctx context.Context,
	req infer.UpdateRequest[UserGroupArgs, UserGroupState],
) (infer.UpdateResponse[UserGroupState], error) {
	client := clientFromContext(ctx)
	var resp userGroupResponse
	if err := client.do(ctx, "PUT", "/api/user-groups/"+req.ID, req.Inputs, &resp); err != nil {
		return infer.UpdateResponse[UserGroupState]{}, err
	}
	return infer.UpdateResponse[UserGroupState]{Output: resp.toState()}, nil
}

// Delete removes a user group.
func (*UserGroup) Delete(
	ctx context.Context,
	req infer.DeleteRequest[UserGroupState],
) (infer.DeleteResponse, error) {
	client := clientFromContext(ctx)
	if err := client.do(ctx, "DELETE", "/api/user-groups/"+req.ID, nil, nil); err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

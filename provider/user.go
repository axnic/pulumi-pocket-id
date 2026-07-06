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

// User manages a Pocket-ID user.
type User struct{}

// UserArgs are the inputs for a user.
type UserArgs struct {
	Username      string   `pulumi:"username" json:"username"`
	Email         *string  `pulumi:"email,optional" json:"email,omitempty"`
	EmailVerified bool     `pulumi:"emailVerified,optional" json:"emailVerified,omitempty"`
	FirstName     string   `pulumi:"firstName,optional" json:"firstName,omitempty"`
	LastName      *string  `pulumi:"lastName,optional" json:"lastName,omitempty"`
	DisplayName   string   `pulumi:"displayName,optional" json:"displayName,omitempty"`
	IsAdmin       bool     `pulumi:"isAdmin,optional" json:"isAdmin,omitempty"`
	Locale        *string  `pulumi:"locale,optional" json:"locale,omitempty"`
	Disabled      bool     `pulumi:"disabled,optional" json:"disabled,omitempty"`
	UserGroupIds  []string `pulumi:"userGroupIds,optional" json:"userGroupIds,omitempty"`
}

// UserState is persisted in Pulumi state.
type UserState struct {
	UserArgs
}

// userResponse mirrors the UserDto response.
type userResponse struct {
	Id            string `json:"id"`
	Username      string `json:"username"`
	Email         *string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
	FirstName     string `json:"firstName"`
	LastName      *string `json:"lastName"`
	DisplayName   string `json:"displayName"`
	IsAdmin       bool   `json:"isAdmin"`
	Locale        *string `json:"locale"`
	Disabled      bool   `json:"disabled"`
	UserGroups    []struct {
		Id string `json:"id"`
	} `json:"userGroups"`
}

func (resp userResponse) toState() UserState {
	groupIds := make([]string, 0, len(resp.UserGroups))
	for _, g := range resp.UserGroups {
		groupIds = append(groupIds, g.Id)
	}
	return UserState{
		UserArgs: UserArgs{
			Username:      resp.Username,
			Email:         resp.Email,
			EmailVerified: resp.EmailVerified,
			FirstName:     resp.FirstName,
			LastName:      resp.LastName,
			DisplayName:   resp.DisplayName,
			IsAdmin:       resp.IsAdmin,
			Locale:        resp.Locale,
			Disabled:      resp.Disabled,
			UserGroupIds:  groupIds,
		},
	}
}

// Create creates a new user.
func (*User) Create(
	ctx context.Context,
	req infer.CreateRequest[UserArgs],
) (infer.CreateResponse[UserState], error) {
	if req.DryRun {
		return infer.CreateResponse[UserState]{
			ID:     req.Name,
			Output: UserState{UserArgs: req.Inputs},
		}, nil
	}

	client := clientFromContext(ctx)
	var resp userResponse
	if err := client.do(ctx, "POST", "/api/users", req.Inputs, &resp); err != nil {
		return infer.CreateResponse[UserState]{}, err
	}
	state := resp.toState()
	return infer.CreateResponse[UserState]{ID: resp.Id, Output: state}, nil
}

// Read fetches the current state of a user.
func (*User) Read(
	ctx context.Context,
	req infer.ReadRequest[UserArgs, UserState],
) (infer.ReadResponse[UserArgs, UserState], error) {
	client := clientFromContext(ctx)
	var resp userResponse
	if err := client.do(ctx, "GET", "/api/users/"+req.ID, nil, &resp); err != nil {
		return infer.ReadResponse[UserArgs, UserState]{}, err
	}
	state := resp.toState()
	return infer.ReadResponse[UserArgs, UserState]{ID: resp.Id, Inputs: state.UserArgs, State: state}, nil
}

// Update modifies an existing user.
func (*User) Update(
	ctx context.Context,
	req infer.UpdateRequest[UserArgs, UserState],
) (infer.UpdateResponse[UserState], error) {
	client := clientFromContext(ctx)
	var resp userResponse
	if err := client.do(ctx, "PUT", "/api/users/"+req.ID, req.Inputs, &resp); err != nil {
		return infer.UpdateResponse[UserState]{}, err
	}
	return infer.UpdateResponse[UserState]{Output: resp.toState()}, nil
}

// Delete removes a user.
func (*User) Delete(
	ctx context.Context,
	req infer.DeleteRequest[UserState],
) (infer.DeleteResponse, error) {
	client := clientFromContext(ctx)
	if err := client.do(ctx, "DELETE", "/api/users/"+req.ID, nil, nil); err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

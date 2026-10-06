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

func init() { registerResource(infer.Resource(&UserCustomClaims{})) }

// UserCustomClaims authoritatively manages the complete set of custom claims
// of one user. Use at most one instance per user.
type UserCustomClaims struct{}

// UserCustomClaimsArgs are the inputs of a UserCustomClaims.
type UserCustomClaimsArgs struct {
	UserID string            `pulumi:"userId" provider:"replaceOnChanges"`
	Claims map[string]string `pulumi:"claims"`
}

// UserCustomClaimsState is the persisted state of a UserCustomClaims.
type UserCustomClaimsState struct {
	UserCustomClaimsArgs
}

var _ infer.Annotated = (*UserCustomClaimsArgs)(nil)

// Annotate describes the inputs of a UserCustomClaims.
func (a *UserCustomClaimsArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.UserID, "The ID of the user. Changing it replaces the resource.")
	an.Describe(&a.Claims, "The custom claims of the user, as key/value pairs. The set is authoritative: "+
		"claims not listed are removed from the user.")
}

var _ infer.Annotated = (*UserCustomClaims)(nil)

// Annotate describes the UserCustomClaims resource.
func (*UserCustomClaims) Annotate(an infer.Annotator) {
	an.Describe(&UserCustomClaims{}, "The authoritative set of custom claims of a user. "+
		"Destroying the resource removes every custom claim from the user.")
}

// Create sets the claims of the user.
func (*UserCustomClaims) Create(
	ctx context.Context, req infer.CreateRequest[UserCustomClaimsArgs],
) (infer.CreateResponse[UserCustomClaimsState], error) {
	state := UserCustomClaimsState{UserCustomClaimsArgs: req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[UserCustomClaimsState]{ID: req.Inputs.UserID, Output: state}, nil
	}
	_, err := clientFromContext(ctx).SetUserCustomClaims(ctx, req.Inputs.UserID, claimsToList(req.Inputs.Claims))
	if err != nil {
		return infer.CreateResponse[UserCustomClaimsState]{}, err
	}
	return infer.CreateResponse[UserCustomClaimsState]{ID: req.Inputs.UserID, Output: state}, nil
}

// Read refreshes the claims from the user; a missing user is dropped from state.
func (*UserCustomClaims) Read(
	ctx context.Context, req infer.ReadRequest[UserCustomClaimsArgs, UserCustomClaimsState],
) (infer.ReadResponse[UserCustomClaimsArgs, UserCustomClaimsState], error) {
	u, err := clientFromContext(ctx).GetUser(ctx, req.ID)
	if isGone(err) {
		return infer.ReadResponse[UserCustomClaimsArgs, UserCustomClaimsState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[UserCustomClaimsArgs, UserCustomClaimsState]{}, err
	}
	args := UserCustomClaimsArgs{UserID: u.ID, Claims: claimsToMap(u.CustomClaims)}
	return infer.ReadResponse[UserCustomClaimsArgs, UserCustomClaimsState]{
		ID: u.ID, Inputs: args, State: UserCustomClaimsState{UserCustomClaimsArgs: args},
	}, nil
}

// Update sets the claims of the user.
func (*UserCustomClaims) Update(
	ctx context.Context, req infer.UpdateRequest[UserCustomClaimsArgs, UserCustomClaimsState],
) (infer.UpdateResponse[UserCustomClaimsState], error) {
	state := UserCustomClaimsState{UserCustomClaimsArgs: req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[UserCustomClaimsState]{Output: state}, nil
	}
	if _, err := clientFromContext(ctx).SetUserCustomClaims(ctx, req.ID, claimsToList(req.Inputs.Claims)); err != nil {
		return infer.UpdateResponse[UserCustomClaimsState]{}, err
	}
	return infer.UpdateResponse[UserCustomClaimsState]{Output: state}, nil
}

// Delete clears the claims of the user.
func (*UserCustomClaims) Delete(
	ctx context.Context, req infer.DeleteRequest[UserCustomClaimsState],
) (infer.DeleteResponse, error) {
	if _, err := clientFromContext(ctx).SetUserCustomClaims(ctx, req.ID, nil); err != nil && !isGone(err) {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

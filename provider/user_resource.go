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

func init() { registerResource(infer.Resource(&User{})) }

// User manages a Pocket-ID user. Group membership is managed from the group
// side (see UserGroupMembers), never here.
type User struct{}

// UserArgs are the inputs of a User.
type UserArgs struct {
	Username      string  `pulumi:"username"`
	Email         *string `pulumi:"email,optional"`
	EmailVerified bool    `pulumi:"emailVerified,optional"`
	FirstName     *string `pulumi:"firstName,optional"`
	LastName      *string `pulumi:"lastName,optional"`
	DisplayName   *string `pulumi:"displayName,optional"`
	IsAdmin       bool    `pulumi:"isAdmin,optional"`
	Locale        *string `pulumi:"locale,optional"`
	Disabled      bool    `pulumi:"disabled,optional"`
}

// UserState is the persisted state of a User.
type UserState struct {
	UserArgs
}

var _ infer.Annotated = (*UserArgs)(nil)

// Annotate describes the inputs of a User.
func (a *UserArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Username, "The unique login name of the user.")
	an.Describe(&a.Email, "The email address of the user.")
	an.Describe(&a.EmailVerified, "Whether the email address is considered verified.")
	an.Describe(&a.FirstName, "The first name of the user.")
	an.Describe(&a.LastName, "The last name of the user.")
	an.Describe(&a.DisplayName, "The name displayed for the user. Pocket-ID may derive it from the first and last names "+
		"when unset.")
	an.Describe(&a.IsAdmin, "Whether the user is an administrator.")
	an.Describe(&a.Locale, "The locale of the user, for example en or fr.")
	an.Describe(&a.Disabled, "Whether the user is disabled and cannot sign in.")
}

var _ infer.Annotated = (*User)(nil)

// Annotate describes the User resource.
func (*User) Annotate(an infer.Annotator) {
	an.Describe(&User{}, "A Pocket-ID user. Group membership is managed with the UserGroupMembers resource.")
}

func (a UserArgs) input() pocketidclient.UserInput {
	return pocketidclient.UserInput{
		Username: a.Username, Email: a.Email, EmailVerified: a.EmailVerified,
		FirstName: a.FirstName, LastName: a.LastName, DisplayName: a.DisplayName,
		IsAdmin: a.IsAdmin, Locale: a.Locale, Disabled: a.Disabled,
	}
}

// Create creates the user.
func (*User) Create(ctx context.Context, req infer.CreateRequest[UserArgs]) (infer.CreateResponse[UserState], error) {
	state := UserState{UserArgs: req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[UserState]{ID: req.Name, Output: state}, nil
	}
	u, err := clientFromContext(ctx).CreateUser(ctx, req.Inputs.input())
	if err != nil {
		return infer.CreateResponse[UserState]{}, err
	}
	return infer.CreateResponse[UserState]{ID: u.ID, Output: state}, nil
}

// Read refreshes the user from the API; a missing user is dropped from state.
func (*User) Read(
	ctx context.Context, req infer.ReadRequest[UserArgs, UserState],
) (infer.ReadResponse[UserArgs, UserState], error) {
	u, err := clientFromContext(ctx).GetUser(ctx, req.ID)
	if isGone(err) {
		return infer.ReadResponse[UserArgs, UserState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[UserArgs, UserState]{}, err
	}
	// An empty prior username means an import: take everything from the API.
	// Otherwise optional strings the program left unset stay unset, so values
	// Pocket-ID derives itself (e.g. displayName) do not show up as drift.
	imported := req.State.Username == ""
	prior := req.State
	args := UserArgs{
		Username:      u.Username,
		Email:         optionalInput(u.Email, prior.Email, imported),
		EmailVerified: u.EmailVerified,
		FirstName:     optionalInput(&u.FirstName, prior.FirstName, imported),
		LastName:      optionalInput(u.LastName, prior.LastName, imported),
		DisplayName:   optionalInput(&u.DisplayName, prior.DisplayName, imported),
		IsAdmin:       u.IsAdmin,
		Locale:        optionalInput(u.Locale, prior.Locale, imported),
		Disabled:      u.Disabled,
	}
	return infer.ReadResponse[UserArgs, UserState]{ID: u.ID, Inputs: args, State: UserState{UserArgs: args}}, nil
}

// Update replaces the user's attributes.
func (*User) Update(
	ctx context.Context, req infer.UpdateRequest[UserArgs, UserState],
) (infer.UpdateResponse[UserState], error) {
	state := UserState{UserArgs: req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[UserState]{Output: state}, nil
	}
	if _, err := clientFromContext(ctx).UpdateUser(ctx, req.ID, req.Inputs.input()); err != nil {
		return infer.UpdateResponse[UserState]{}, err
	}
	return infer.UpdateResponse[UserState]{Output: state}, nil
}

// Delete removes the user.
func (*User) Delete(ctx context.Context, req infer.DeleteRequest[UserState]) (infer.DeleteResponse, error) {
	if err := clientFromContext(ctx).DeleteUser(ctx, req.ID); err != nil && !isGone(err) {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

// optionalInput maps an API string to an optional input: empty becomes nil, and a
// value the program left unset (prior nil) stays nil unless importing.
func optionalInput(api, prior *string, imported bool) *string {
	if api == nil || *api == "" {
		return nil
	}
	if prior == nil && !imported {
		return nil
	}
	return api
}

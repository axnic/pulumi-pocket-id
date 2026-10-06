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

func init() { registerFunction(infer.Function[*GetUser, GetUserArgs, GetUserResult](&GetUser{})) }

// GetUser looks a user up by ID or username.
type GetUser struct{}

// GetUserArgs are the arguments of getUser.
type GetUserArgs struct {
	ID       *string `pulumi:"userId,optional"`
	Username *string `pulumi:"username,optional"`
}

// GetUserResult is the result of getUser.
type GetUserResult struct {
	ID            string            `pulumi:"userId"`
	Username      string            `pulumi:"username"`
	Email         string            `pulumi:"email"`
	EmailVerified bool              `pulumi:"emailVerified"`
	FirstName     string            `pulumi:"firstName"`
	LastName      string            `pulumi:"lastName"`
	DisplayName   string            `pulumi:"displayName"`
	IsAdmin       bool              `pulumi:"isAdmin"`
	Locale        string            `pulumi:"locale"`
	Disabled      bool              `pulumi:"disabled"`
	CustomClaims  map[string]string `pulumi:"customClaims"`
	UserGroupIDs  []string          `pulumi:"userGroupIds"`
}

var _ infer.Annotated = (*GetUserArgs)(nil)

// Annotate describes the arguments of getUser.
func (a *GetUserArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ID, "The ID of the user. Exactly one of userId or username must be set.")
	an.Describe(&a.Username, "The exact username of the user. Exactly one of userId or username must be set.")
}

var _ infer.Annotated = (*GetUserResult)(nil)

// Annotate describes the result of getUser.
func (r *GetUserResult) Annotate(an infer.Annotator) {
	an.Describe(&r.ID, "The ID of the user.")
	an.Describe(&r.Username, "The username of the user.")
	an.Describe(&r.Email, "The email address of the user, empty if unset.")
	an.Describe(&r.EmailVerified, "Whether the email address is verified.")
	an.Describe(&r.FirstName, "The first name of the user.")
	an.Describe(&r.LastName, "The last name of the user, empty if unset.")
	an.Describe(&r.DisplayName, "The display name of the user.")
	an.Describe(&r.IsAdmin, "Whether the user is an administrator.")
	an.Describe(&r.Locale, "The locale of the user, empty if unset.")
	an.Describe(&r.Disabled, "Whether the user is disabled.")
	an.Describe(&r.CustomClaims, "The custom claims of the user.")
	an.Describe(&r.UserGroupIDs, "The IDs of the groups the user belongs to.")
}

var _ infer.Annotated = (*GetUser)(nil)

// Annotate describes the getUser function.
func (*GetUser) Annotate(an infer.Annotator) {
	an.Describe(&GetUser{}, "Looks up a Pocket-ID user by ID or by exact username.")
}

// Invoke resolves the user.
func (*GetUser) Invoke(
	ctx context.Context, req infer.FunctionRequest[GetUserArgs],
) (infer.FunctionResponse[GetUserResult], error) {
	in := req.Input
	hasID, hasName := in.ID != nil && *in.ID != "", in.Username != nil && *in.Username != ""
	if hasID == hasName {
		return infer.FunctionResponse[GetUserResult]{}, errors.New("exactly one of userId or username must be set")
	}
	client := clientFromContext(ctx)

	var u *pocketidclient.User
	if hasID {
		var err error
		if u, err = client.GetUser(ctx, *in.ID); err != nil {
			return infer.FunctionResponse[GetUserResult]{}, fmt.Errorf("getting user %q: %w", *in.ID, err)
		}
	} else {
		users, err := client.ListUsers(ctx, *in.Username)
		if err != nil {
			return infer.FunctionResponse[GetUserResult]{}, err
		}
		var matches []pocketidclient.User
		for _, c := range users {
			if c.Username == *in.Username {
				matches = append(matches, c)
			}
		}
		if len(matches) != 1 {
			return infer.FunctionResponse[GetUserResult]{},
				fmt.Errorf("expected exactly one user with username %q, found %d", *in.Username, len(matches))
		}
		u = &matches[0]
	}

	groups := make([]string, 0, len(u.UserGroups))
	for _, g := range u.UserGroups {
		groups = append(groups, g.ID)
	}
	return infer.FunctionResponse[GetUserResult]{Output: GetUserResult{
		ID: u.ID, Username: u.Username, Email: emptyIfNil(u.Email), EmailVerified: u.EmailVerified,
		FirstName: u.FirstName, LastName: emptyIfNil(u.LastName), DisplayName: u.DisplayName,
		IsAdmin: u.IsAdmin, Locale: emptyIfNil(u.Locale), Disabled: u.Disabled,
		CustomClaims: claimsToMap(u.CustomClaims), UserGroupIDs: groups,
	}}, nil
}

func emptyIfNil(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

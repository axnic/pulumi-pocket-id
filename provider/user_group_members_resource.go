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
	"slices"
	"sort"

	"github.com/pulumi/pulumi-go-provider/infer"
)

func init() { registerResource(infer.Resource(&UserGroupMembers{})) }

// UserGroupMembers authoritatively manages the complete member set of one
// group. Use at most one instance per group: it removes any user not listed.
type UserGroupMembers struct{}

// UserGroupMembersArgs are the inputs of a UserGroupMembers.
type UserGroupMembersArgs struct {
	GroupID string   `pulumi:"groupId" provider:"replaceOnChanges"`
	UserIDs []string `pulumi:"userIds"`
}

// UserGroupMembersState is the persisted state of a UserGroupMembers.
type UserGroupMembersState struct {
	UserGroupMembersArgs
}

var _ infer.Annotated = (*UserGroupMembersArgs)(nil)

// Annotate describes the inputs of a UserGroupMembers.
func (a *UserGroupMembersArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.GroupID, "The ID of the group. Changing it replaces the resource.")
	an.Describe(&a.UserIDs, "The IDs of all users that belong to the group. The set is authoritative: "+
		"users not listed are removed from the group.")
}

var _ infer.Annotated = (*UserGroupMembers)(nil)

// Annotate describes the UserGroupMembers resource.
func (*UserGroupMembers) Annotate(an infer.Annotator) {
	an.Describe(&UserGroupMembers{}, "The authoritative set of members of a user group. "+
		"Destroying the resource removes every member from the group.")
}

// Create sets the member set of the group.
func (*UserGroupMembers) Create(
	ctx context.Context, req infer.CreateRequest[UserGroupMembersArgs],
) (infer.CreateResponse[UserGroupMembersState], error) {
	state := UserGroupMembersState{UserGroupMembersArgs: req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[UserGroupMembersState]{ID: req.Inputs.GroupID, Output: state}, nil
	}
	if _, err := clientFromContext(ctx).SetUserGroupMembers(ctx, req.Inputs.GroupID, req.Inputs.UserIDs); err != nil {
		return infer.CreateResponse[UserGroupMembersState]{}, err
	}
	return infer.CreateResponse[UserGroupMembersState]{ID: req.Inputs.GroupID, Output: state}, nil
}

// Read refreshes the member set; a missing group is dropped from state.
func (*UserGroupMembers) Read(
	ctx context.Context, req infer.ReadRequest[UserGroupMembersArgs, UserGroupMembersState],
) (infer.ReadResponse[UserGroupMembersArgs, UserGroupMembersState], error) {
	g, err := clientFromContext(ctx).GetUserGroup(ctx, req.ID)
	if isGone(err) {
		return infer.ReadResponse[UserGroupMembersArgs, UserGroupMembersState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[UserGroupMembersArgs, UserGroupMembersState]{}, err
	}
	ids := make([]string, 0, len(g.Users))
	for _, u := range g.Users {
		ids = append(ids, u.ID)
	}
	args := UserGroupMembersArgs{GroupID: g.ID, UserIDs: orderLike(req.State.UserIDs, ids)}
	return infer.ReadResponse[UserGroupMembersArgs, UserGroupMembersState]{
		ID: g.ID, Inputs: args, State: UserGroupMembersState{UserGroupMembersArgs: args},
	}, nil
}

// Update sets the member set of the group.
func (*UserGroupMembers) Update(
	ctx context.Context, req infer.UpdateRequest[UserGroupMembersArgs, UserGroupMembersState],
) (infer.UpdateResponse[UserGroupMembersState], error) {
	state := UserGroupMembersState{UserGroupMembersArgs: req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[UserGroupMembersState]{Output: state}, nil
	}
	if _, err := clientFromContext(ctx).SetUserGroupMembers(ctx, req.ID, req.Inputs.UserIDs); err != nil {
		return infer.UpdateResponse[UserGroupMembersState]{}, err
	}
	return infer.UpdateResponse[UserGroupMembersState]{Output: state}, nil
}

// Delete empties the group.
func (*UserGroupMembers) Delete(
	ctx context.Context, req infer.DeleteRequest[UserGroupMembersState],
) (infer.DeleteResponse, error) {
	if _, err := clientFromContext(ctx).SetUserGroupMembers(ctx, req.ID, nil); err != nil && !isGone(err) {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

// orderLike returns actual as a set ordered like prior (members still present
// first, in prior order), then new ones sorted. The member set is unordered,
// so this keeps API ordering from showing up as a diff.
func orderLike(prior, actual []string) []string {
	out := make([]string, 0, len(actual))
	for _, id := range prior {
		if slices.Contains(actual, id) && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	var extra []string
	for _, id := range actual {
		if !slices.Contains(out, id) {
			extra = append(extra, id)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}

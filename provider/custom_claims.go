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
	"fmt"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

const (
	customClaimsOwnerUser      = "user"
	customClaimsOwnerUserGroup = "user-group"
)

// CustomClaim is a single key/value claim attached to a user or user group.
type CustomClaim struct {
	Key   string `pulumi:"key" json:"key"`
	Value string `pulumi:"value" json:"value"`
}

// CustomClaims manages the set of custom claims for a user or a user group.
//
// Pocket-ID stores custom claims as a full-replacement list on a parent resource
// (a user or a user group). This resource models that list as an authoritative set:
// every apply replaces the entire claims collection for the target owner.
type CustomClaims struct{}

// CustomClaimsArgs are the inputs for a custom claims resource.
type CustomClaimsArgs struct {
	// OwnerType is either "user" or "user-group".
	OwnerType string `pulumi:"ownerType" json:"ownerType"`
	// OwnerID is the ID of the user or user group that owns the claims.
	OwnerID string `pulumi:"ownerId" json:"ownerId"`
	// Claims is the full list of custom claims for the owner.
	Claims []CustomClaim `pulumi:"claims,optional" json:"claims,omitempty"`
}

// CustomClaimsState is persisted in Pulumi state.
type CustomClaimsState struct {
	CustomClaimsArgs
}

func validateOwnerType(t string) error {
	switch t {
	case customClaimsOwnerUser, customClaimsOwnerUserGroup:
		return nil
	default:
		return fmt.Errorf("ownerType must be %q or %q, got %q", customClaimsOwnerUser, customClaimsOwnerUserGroup, t)
	}
}

// Create (replaces) the full set of custom claims for the owner.
func (*CustomClaims) Create(
	ctx context.Context,
	req infer.CreateRequest[CustomClaimsArgs],
) (infer.CreateResponse[CustomClaimsState], error) {
	if err := validateOwnerType(req.Inputs.OwnerType); err != nil {
		return infer.CreateResponse[CustomClaimsState]{}, err
	}
	id := claimsID(req.Inputs.OwnerType, req.Inputs.OwnerID)
	if req.DryRun {
		return infer.CreateResponse[CustomClaimsState]{
			ID:     id,
			Output: CustomClaimsState{CustomClaimsArgs: req.Inputs},
		}, nil
	}
	client := clientFromContext(ctx)
	if err := putClaims(ctx, client, req.Inputs.OwnerType, req.Inputs.OwnerID, req.Inputs.Claims); err != nil {
		return infer.CreateResponse[CustomClaimsState]{}, err
	}
	return infer.CreateResponse[CustomClaimsState]{ID: id, Output: CustomClaimsState{CustomClaimsArgs: req.Inputs}}, nil
}

// Read fetches the current claims from the owner resource.
func (*CustomClaims) Read(
	ctx context.Context,
	req infer.ReadRequest[CustomClaimsArgs, CustomClaimsState],
) (infer.ReadResponse[CustomClaimsArgs, CustomClaimsState], error) {
	client := clientFromContext(ctx)
	claims, err := getClaims(ctx, client, req.State.OwnerType, req.State.OwnerID)
	if err != nil {
		return infer.ReadResponse[CustomClaimsArgs, CustomClaimsState]{}, err
	}
	inputs := req.State.CustomClaimsArgs
	inputs.Claims = claims
	return infer.ReadResponse[CustomClaimsArgs, CustomClaimsState]{
		ID:     claimsID(inputs.OwnerType, inputs.OwnerID),
		Inputs: inputs,
		State:  CustomClaimsState{CustomClaimsArgs: inputs},
	}, nil
}

// Update replaces the full set of claims with the new list.
func (*CustomClaims) Update(
	ctx context.Context,
	req infer.UpdateRequest[CustomClaimsArgs, CustomClaimsState],
) (infer.UpdateResponse[CustomClaimsState], error) {
	if err := validateOwnerType(req.Inputs.OwnerType); err != nil {
		return infer.UpdateResponse[CustomClaimsState]{}, err
	}
	client := clientFromContext(ctx)
	if err := putClaims(ctx, client, req.Inputs.OwnerType, req.Inputs.OwnerID, req.Inputs.Claims); err != nil {
		return infer.UpdateResponse[CustomClaimsState]{}, err
	}
	return infer.UpdateResponse[CustomClaimsState]{Output: CustomClaimsState{CustomClaimsArgs: req.Inputs}}, nil
}

// Delete clears all custom claims for the owner.
func (*CustomClaims) Delete(
	ctx context.Context,
	req infer.DeleteRequest[CustomClaimsState],
) (infer.DeleteResponse, error) {
	client := clientFromContext(ctx)
	if err := putClaims(ctx, client, req.State.OwnerType, req.State.OwnerID, nil); err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

// Diff forces a replacement when the owner identity changes, and an update otherwise.
func (*CustomClaims) Diff(
	_ context.Context,
	req infer.DiffRequest[CustomClaimsArgs, CustomClaimsState],
) (infer.DiffResponse, error) {
	if req.Inputs.OwnerType != req.State.OwnerType || req.Inputs.OwnerID != req.State.OwnerID {
		diff := map[string]p.PropertyDiff{}
		if req.Inputs.OwnerType != req.State.OwnerType {
			diff["ownerType"] = p.PropertyDiff{Kind: p.UpdateReplace, InputDiff: true}
		}
		if req.Inputs.OwnerID != req.State.OwnerID {
			diff["ownerId"] = p.PropertyDiff{Kind: p.UpdateReplace, InputDiff: true}
		}
		return p.DiffResponse{HasChanges: true, DeleteBeforeReplace: true, DetailedDiff: diff}, nil
	}
	changed := !equalClaims(req.Inputs.Claims, req.State.Claims)
	return p.DiffResponse{HasChanges: changed}, nil
}

func claimsID(ownerType, ownerID string) string {
	return ownerType + "/" + ownerID
}

func ownerPath(ownerType, ownerID string) string {
	return "/api/custom-claims/" + ownerType + "/" + ownerID
}

func putClaims(ctx context.Context, c *Client, ownerType, ownerID string, claims []CustomClaim) error {
	if claims == nil {
		claims = []CustomClaim{}
	}
	return c.do(ctx, "PUT", ownerPath(ownerType, ownerID), claims, nil)
}

func getClaims(ctx context.Context, c *Client, ownerType, ownerID string) ([]CustomClaim, error) {
	var resp struct {
		CustomClaims []CustomClaim `json:"customClaims"`
	}
	var path string
	switch ownerType {
	case customClaimsOwnerUser:
		path = "/api/users/" + ownerID
	case customClaimsOwnerUserGroup:
		path = "/api/user-groups/" + ownerID
	default:
		return nil, fmt.Errorf(
			"ownerType must be %q or %q, got %q", customClaimsOwnerUser, customClaimsOwnerUserGroup, ownerType,
		)
	}
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	if resp.CustomClaims == nil {
		resp.CustomClaims = []CustomClaim{}
	}
	return resp.CustomClaims, nil
}

func equalClaims(a, b []CustomClaim) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

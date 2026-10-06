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
	"time"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/axnic/pulumi-pocket-id/provider/internal/pocketidclient"
)

func init() { registerResource(infer.Resource(&SignupToken{})) }

// SignupToken manages a Pocket-ID signup token. Tokens are immutable: any
// input change replaces the resource.
type SignupToken struct{}

// SignupTokenArgs are the inputs of a SignupToken.
type SignupTokenArgs struct {
	TTL          int      `pulumi:"ttl" provider:"replaceOnChanges"`
	UsageLimit   int      `pulumi:"usageLimit" provider:"replaceOnChanges"`
	UserGroupIDs []string `pulumi:"userGroupIds,optional" provider:"replaceOnChanges"`
}

// SignupTokenState is the persisted state of a SignupToken.
type SignupTokenState struct {
	SignupTokenArgs
	Token     string `pulumi:"token" provider:"secret"`
	CreatedAt string `pulumi:"createdAt"`
	ExpiresAt string `pulumi:"expiresAt"`
}

var _ infer.Annotated = (*SignupTokenArgs)(nil)

// Annotate describes the inputs of a SignupToken.
func (a *SignupTokenArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.TTL, "The lifetime of the token in seconds. Changing it replaces the resource.")
	an.Describe(&a.UsageLimit, "How many times the token can be used to sign up. Changing it replaces the resource.")
	an.Describe(&a.UserGroupIDs, "The IDs of the groups users signing up with this token join. "+
		"Changing them replaces the resource.")
}

var _ infer.Annotated = (*SignupTokenState)(nil)

// Annotate describes the outputs of a SignupToken.
func (s *SignupTokenState) Annotate(an infer.Annotator) {
	an.Describe(&s.Token, "The secret signup token. It is captured at creation and carried in state; "+
		"it is not read back from the API, so it is empty after an import.")
	an.Describe(&s.CreatedAt, "The RFC 3339 timestamp at which the token was created.")
	an.Describe(&s.ExpiresAt, "The RFC 3339 timestamp at which the token expires.")
}

var _ infer.Annotated = (*SignupToken)(nil)

// Annotate describes the SignupToken resource.
func (*SignupToken) Annotate(an infer.Annotator) {
	an.Describe(&SignupToken{}, "A Pocket-ID signup token that lets new users create their own account.")
}

// Create creates the token and captures its secret.
func (*SignupToken) Create(
	ctx context.Context, req infer.CreateRequest[SignupTokenArgs],
) (infer.CreateResponse[SignupTokenState], error) {
	if req.DryRun {
		return infer.CreateResponse[SignupTokenState]{
			ID: req.Name, Output: SignupTokenState{SignupTokenArgs: req.Inputs},
		}, nil
	}
	t, err := clientFromContext(ctx).CreateSignupToken(ctx, pocketidclient.SignupTokenInput{
		TTLSeconds: int64(req.Inputs.TTL), UsageLimit: req.Inputs.UsageLimit, UserGroupIDs: req.Inputs.UserGroupIDs,
	})
	if err != nil {
		return infer.CreateResponse[SignupTokenState]{}, err
	}
	return infer.CreateResponse[SignupTokenState]{ID: t.ID, Output: SignupTokenState{
		SignupTokenArgs: req.Inputs,
		Token:           t.Token,
		CreatedAt:       t.CreatedAt.Format(time.RFC3339),
		ExpiresAt:       t.ExpiresAt.Format(time.RFC3339),
	}}, nil
}

// Read refreshes the token from the list endpoint; a missing token is dropped
// from state. The secret is kept from state.
func (*SignupToken) Read(
	ctx context.Context, req infer.ReadRequest[SignupTokenArgs, SignupTokenState],
) (infer.ReadResponse[SignupTokenArgs, SignupTokenState], error) {
	t, err := clientFromContext(ctx).GetSignupToken(ctx, req.ID)
	if isGone(err) {
		return infer.ReadResponse[SignupTokenArgs, SignupTokenState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[SignupTokenArgs, SignupTokenState]{}, err
	}
	ids := make([]string, 0, len(t.UserGroups))
	for _, g := range t.UserGroups {
		ids = append(ids, g.ID)
	}
	args := SignupTokenArgs{
		TTL:          req.State.TTL,
		UsageLimit:   t.UsageLimit,
		UserGroupIDs: orderLike(req.State.UserGroupIDs, ids),
	}
	if args.TTL == 0 { // import: derive the lifetime from the timestamps
		args.TTL = int(t.ExpiresAt.Sub(t.CreatedAt).Round(time.Second) / time.Second)
	}
	if len(args.UserGroupIDs) == 0 {
		args.UserGroupIDs = nil
	}
	return infer.ReadResponse[SignupTokenArgs, SignupTokenState]{
		ID: t.ID, Inputs: args, State: SignupTokenState{
			SignupTokenArgs: args,
			Token:           req.State.Token,
			CreatedAt:       t.CreatedAt.Format(time.RFC3339),
			ExpiresAt:       t.ExpiresAt.Format(time.RFC3339),
		},
	}, nil
}

// Delete removes the token.
func (*SignupToken) Delete(ctx context.Context, req infer.DeleteRequest[SignupTokenState]) (infer.DeleteResponse,
	error) {
	if err := clientFromContext(ctx).DeleteSignupToken(ctx, req.ID); err != nil && !isGone(err) {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

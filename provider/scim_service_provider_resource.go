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
	"strings"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/axnic/pulumi-pocket-id/provider/internal/pocketidclient"
)

func init() { registerResource(infer.Resource(&ScimServiceProvider{})) }

// ScimServiceProvider attaches a SCIM service provider to an OIDC client so
// that Pocket-ID provisions users and groups to it. The resource ID is the ID
// of the service provider.
type ScimServiceProvider struct{}

// ScimServiceProviderArgs are the inputs of the ScimServiceProvider resource.
type ScimServiceProviderArgs struct {
	OidcClientID string  `pulumi:"oidcClientId" provider:"replaceOnChanges"`
	Endpoint     string  `pulumi:"endpoint"`
	Token        *string `pulumi:"token,optional" provider:"secret"`
}

// ScimServiceProviderState is persisted in the Pulumi state.
type ScimServiceProviderState struct {
	ScimServiceProviderArgs
	CreatedAt    string `pulumi:"createdAt"`
	LastSyncedAt string `pulumi:"lastSyncedAt"`
}

var _ infer.Annotated = (*ScimServiceProvider)(nil)

// Annotate describes the resource.
func (*ScimServiceProvider) Annotate(a infer.Annotator) {
	a.Describe(&ScimServiceProvider{}, "A SCIM service provider attached to an OIDC client. "+
		"To import it, use the ID <oidcClientId>/<serviceProviderId>.")
}

var _ infer.Annotated = (*ScimServiceProviderArgs)(nil)

// Annotate describes the inputs.
func (a *ScimServiceProviderArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.OidcClientID, "The ID of the OIDC client the service provider belongs to. Changing it replaces the "+
		"resource.")
	an.Describe(&a.Endpoint, "The base URL of the SCIM endpoint.")
	an.Describe(&a.Token, "The bearer token used to authenticate against the SCIM endpoint. "+
		"Never read back from the API: it is kept from the state and unset after an import.")
}

var _ infer.Annotated = (*ScimServiceProviderState)(nil)

// Annotate describes the outputs.
func (s *ScimServiceProviderState) Annotate(a infer.Annotator) {
	a.Describe(&s.CreatedAt, "The creation date of the service provider.")
	a.Describe(&s.LastSyncedAt, "The date of the last synchronization, empty if it never ran.")
}

func scimRequest(a ScimServiceProviderArgs) pocketidclient.ScimServiceProviderRequest {
	return pocketidclient.ScimServiceProviderRequest{
		OidcClientID: a.OidcClientID, Endpoint: a.Endpoint, Token: oidcDeref(a.Token),
	}
}

// Create creates the service provider.
func (*ScimServiceProvider) Create(
	ctx context.Context, req infer.CreateRequest[ScimServiceProviderArgs],
) (infer.CreateResponse[ScimServiceProviderState], error) {
	if req.DryRun {
		return infer.CreateResponse[ScimServiceProviderState]{
			ID: req.Name, Output: ScimServiceProviderState{ScimServiceProviderArgs: req.Inputs},
		}, nil
	}
	got, err := clientFromContext(ctx).CreateScimServiceProvider(ctx, scimRequest(req.Inputs))
	if err != nil {
		return infer.CreateResponse[ScimServiceProviderState]{}, err
	}
	return infer.CreateResponse[ScimServiceProviderState]{ID: got.ID, Output: scimState(got, req.Inputs)}, nil
}

// Read fetches the service provider through its OIDC client. The token is carried over from the state.
func (*ScimServiceProvider) Read(
	ctx context.Context, req infer.ReadRequest[ScimServiceProviderArgs, ScimServiceProviderState],
) (infer.ReadResponse[ScimServiceProviderArgs, ScimServiceProviderState], error) {
	none := infer.ReadResponse[ScimServiceProviderArgs, ScimServiceProviderState]{}
	id, clientID := req.ID, req.State.OidcClientID
	if clientID == "" { // import: <oidcClientId>/<serviceProviderId>
		var ok bool
		if clientID, id, ok = strings.Cut(req.ID, "/"); !ok || clientID == "" || id == "" {
			return none, fmt.Errorf("invalid ID %q, expected <oidcClientId>/<serviceProviderId>", req.ID)
		}
	}
	got, err := clientFromContext(ctx).GetOidcClientScimServiceProvider(ctx, clientID)
	if isGone(err) || (err == nil && got.ID != id) {
		return none, nil
	}
	if err != nil {
		return none, err
	}
	args := ScimServiceProviderArgs{OidcClientID: clientID, Endpoint: got.Endpoint, Token: req.State.Token}
	return infer.ReadResponse[ScimServiceProviderArgs, ScimServiceProviderState]{
		ID: got.ID, Inputs: args, State: scimState(got, args),
	}, nil
}

// Update replaces the endpoint and token.
func (*ScimServiceProvider) Update(
	ctx context.Context, req infer.UpdateRequest[ScimServiceProviderArgs, ScimServiceProviderState],
) (infer.UpdateResponse[ScimServiceProviderState], error) {
	if req.DryRun {
		out := req.State
		out.ScimServiceProviderArgs = req.Inputs
		return infer.UpdateResponse[ScimServiceProviderState]{Output: out}, nil
	}
	got, err := clientFromContext(ctx).UpdateScimServiceProvider(ctx, req.ID, scimRequest(req.Inputs))
	if err != nil {
		return infer.UpdateResponse[ScimServiceProviderState]{}, err
	}
	return infer.UpdateResponse[ScimServiceProviderState]{Output: scimState(got, req.Inputs)}, nil
}

// Delete removes the service provider.
func (*ScimServiceProvider) Delete(
	ctx context.Context, req infer.DeleteRequest[ScimServiceProviderState],
) (infer.DeleteResponse, error) {
	if err := clientFromContext(ctx).DeleteScimServiceProvider(ctx, req.ID); err != nil && !isGone(err) {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

func scimState(sp *pocketidclient.ScimServiceProvider, args ScimServiceProviderArgs) ScimServiceProviderState {
	return ScimServiceProviderState{ScimServiceProviderArgs: args, CreatedAt: sp.CreatedAt, LastSyncedAt: sp.LastSyncedAt}
}

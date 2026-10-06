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
	"slices"
	"strings"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/axnic/pulumi-pocket-id/provider/internal/pocketidclient"
)

func init() { registerResource(infer.Resource(&APIClientGrant{})) }

// APIClientGrant manages the access of one OIDC client on one API. It is
// authoritative for that (API, client) pair: the permission sets are replaced
// as a whole and deleting the resource revokes all access. It is imported by
// the ID "<apiId>/<clientId>".
type APIClientGrant struct{}

// APIClientGrantArgs are the inputs of the APIClientGrant resource.
type APIClientGrantArgs struct {
	APIID                       string   `pulumi:"apiId" provider:"replaceOnChanges"`
	ClientID                    string   `pulumi:"clientId" provider:"replaceOnChanges"`
	ClientAccess                *bool    `pulumi:"clientAccess,optional"`
	ClientPermissionKeys        []string `pulumi:"clientPermissionKeys,optional"`
	UserDelegatedAccess         *bool    `pulumi:"userDelegatedAccess,optional"`
	UserDelegatedPermissionKeys []string `pulumi:"userDelegatedPermissionKeys,optional"`
}

// APIClientGrantState is the persisted state of the APIClientGrant resource.
type APIClientGrantState struct {
	APIClientGrantArgs
}

var _ infer.Annotated = (*APIClientGrantArgs)(nil)

// Annotate provides schema descriptions for APIClientGrantArgs.
func (a *APIClientGrantArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.APIID, "The ID of the API. Changing it replaces the grant.")
	an.Describe(&a.ClientID, "The ID of the OIDC client. Changing it replaces the grant.")
	an.Describe(&a.ClientAccess, "Whether the client may access the API on its own behalf (client credentials flow).")
	an.Describe(&a.ClientPermissionKeys, "The keys of the API permissions granted to the client on its own behalf.")
	an.Describe(&a.UserDelegatedAccess, "Whether the client may access the API on behalf of users.")
	an.Describe(&a.UserDelegatedPermissionKeys, "The keys of the API permissions the client may use on behalf of users.")
}

var _ infer.Annotated = (*APIClientGrant)(nil)

// Annotate describes the APIClientGrant resource.
func (APIClientGrant) Annotate(an infer.Annotator) {
	an.SetToken("index", "ApiClientGrant")
	an.Describe(&APIClientGrant{}, "The access of an OIDC client on an API. Authoritative for the (API, client) pair; "+
		"import with the ID \"<apiId>/<clientId>\".")
}

func (APIClientGrant) Create(
	ctx context.Context, req infer.CreateRequest[APIClientGrantArgs],
) (infer.CreateResponse[APIClientGrantState], error) {
	id := req.Inputs.APIID + "/" + req.Inputs.ClientID
	out := APIClientGrantState{req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[APIClientGrantState]{ID: id, Output: out}, nil
	}
	if err := putAPIClientGrant(ctx, req.Inputs); err != nil {
		return infer.CreateResponse[APIClientGrantState]{}, err
	}
	return infer.CreateResponse[APIClientGrantState]{ID: id, Output: out}, nil
}

func (APIClientGrant) Read(
	ctx context.Context, req infer.ReadRequest[APIClientGrantArgs, APIClientGrantState],
) (infer.ReadResponse[APIClientGrantArgs, APIClientGrantState], error) {
	type resp = infer.ReadResponse[APIClientGrantArgs, APIClientGrantState]
	apiID, clientID, ok := strings.Cut(req.ID, "/")
	if !ok || apiID == "" || clientID == "" {
		return resp{}, fmt.Errorf("invalid ID %q: expected \"<apiId>/<clientId>\"", req.ID)
	}
	c := clientFromContext(ctx)
	api, err := c.GetAPI(ctx, apiID)
	if isGone(err) {
		return resp{}, nil
	}
	if err != nil {
		return resp{}, err
	}
	clients, err := c.ListAPIClients(ctx, apiID)
	if isGone(err) {
		return resp{}, nil
	}
	if err != nil {
		return resp{}, err
	}
	i := slices.IndexFunc(clients, func(a pocketidclient.APIClientAccess) bool { return a.Client.ID == clientID })
	if i < 0 {
		return resp{}, nil
	}
	g := clients[i]
	prior := req.State.APIClientGrantArgs
	idToKey := map[string]string{}
	for _, pm := range api.Permissions {
		idToKey[pm.ID] = pm.Key
	}
	args := APIClientGrantArgs{
		APIID:                       apiID,
		ClientID:                    clientID,
		ClientAccess:                optBool(g.ClientAccess, prior.ClientAccess),
		ClientPermissionKeys:        keysFor(g.ClientPermissionIDs, idToKey, prior.ClientPermissionKeys),
		UserDelegatedAccess:         optBool(g.UserDelegatedAccess, prior.UserDelegatedAccess),
		UserDelegatedPermissionKeys: keysFor(g.UserDelegatedPermissionIDs, idToKey, prior.UserDelegatedPermissionKeys),
	}
	return resp{ID: req.ID, Inputs: args, State: APIClientGrantState{args}}, nil
}

func (APIClientGrant) Update(
	ctx context.Context, req infer.UpdateRequest[APIClientGrantArgs, APIClientGrantState],
) (infer.UpdateResponse[APIClientGrantState], error) {
	out := APIClientGrantState{req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[APIClientGrantState]{Output: out}, nil
	}
	if err := putAPIClientGrant(ctx, req.Inputs); err != nil {
		return infer.UpdateResponse[APIClientGrantState]{}, err
	}
	return infer.UpdateResponse[APIClientGrantState]{Output: out}, nil
}

func (APIClientGrant) Delete(
	ctx context.Context, req infer.DeleteRequest[APIClientGrantState],
) (infer.DeleteResponse, error) {
	err := clientFromContext(ctx).DeleteAPIClientGrant(ctx, req.State.APIID, req.State.ClientID)
	if err != nil && !isGone(err) {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

// putAPIClientGrant resolves permission keys to IDs and replaces the grant.
func putAPIClientGrant(ctx context.Context, a APIClientGrantArgs) error {
	c := clientFromContext(ctx)
	api, err := c.GetAPI(ctx, a.APIID)
	if err != nil {
		return err
	}
	keyToID := map[string]string{}
	for _, pm := range api.Permissions {
		keyToID[pm.Key] = pm.ID
	}
	toIDs := func(keys []string) ([]string, error) {
		ids := make([]string, 0, len(keys))
		for _, k := range keys {
			id, ok := keyToID[k]
			if !ok {
				return nil, fmt.Errorf("API %s has no permission with key %q", a.APIID, k)
			}
			ids = append(ids, id)
		}
		return ids, nil
	}
	clientIDs, err := toIDs(a.ClientPermissionKeys)
	if err != nil {
		return err
	}
	userIDs, err := toIDs(a.UserDelegatedPermissionKeys)
	if err != nil {
		return err
	}
	return c.SetAPIClientGrant(ctx, a.APIID, a.ClientID, pocketidclient.APIClientGrant{
		ClientAccess:               apiDeref2(a.ClientAccess),
		ClientPermissionIDs:        clientIDs,
		UserDelegatedAccess:        apiDeref2(a.UserDelegatedAccess),
		UserDelegatedPermissionIDs: userIDs,
	})
}

func apiDeref2(b *bool) bool { return b != nil && *b }

// optBool keeps an unset optional input unset when the remote value is false.
func optBool(remote bool, prior *bool) *bool {
	if !remote && prior == nil {
		return nil
	}
	return &remote
}

// keysFor maps permission IDs back to keys. The prior order is kept when the
// sets match, so that ordering never causes a spurious diff.
func keysFor(ids []string, idToKey map[string]string, prior []string) []string {
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		if k, ok := idToKey[id]; ok {
			keys = append(keys, k)
		} else {
			keys = append(keys, id)
		}
	}
	slices.Sort(keys)
	if len(keys) == 0 && prior == nil {
		return nil
	}
	if p := slices.Sorted(slices.Values(prior)); slices.Equal(p, keys) {
		return prior
	}
	return keys
}

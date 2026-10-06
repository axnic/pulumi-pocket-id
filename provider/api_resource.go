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

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/axnic/pulumi-pocket-id/provider/internal/pocketidclient"
)

func init() { registerResource(infer.Resource(&API{})) }

// API manages a Pocket-ID API (an OAuth resource server) together with its
// permissions. The permission list is authoritative: permissions that are not
// declared are removed.
type API struct{}

// APIPermissionArgs declares one permission of an API.
type APIPermissionArgs struct {
	Key         string  `pulumi:"key"`
	Name        string  `pulumi:"name"`
	Description *string `pulumi:"description,optional"`
}

// APIPermissionState is a permission as stored by Pocket-ID.
type APIPermissionState struct {
	APIPermissionArgs
	ID string `pulumi:"id"`
}

// APICimdAccessArgs declares what clients registered through a Client ID Metadata Document may use on an API.
type APICimdAccessArgs struct {
	Enabled        bool     `pulumi:"enabled"`
	PermissionKeys []string `pulumi:"permissionKeys,optional"`
}

var _ infer.Annotated = (*APICimdAccessArgs)(nil)

// Annotate provides schema descriptions for APICimdAccessArgs.
func (a *APICimdAccessArgs) Annotate(an infer.Annotator) {
	an.SetToken("index", "ApiCimdAccess")
	an.Describe(&a.Enabled, "Whether clients registered through a Client ID Metadata Document (CIMD) may use the API.")
	an.Describe(&a.PermissionKeys, "The keys of the API permissions that such clients may request. "+
		"Each key must be declared in `permissions`.")
}

// APIArgs are the inputs of the API resource.
type APIArgs struct {
	Name        string              `pulumi:"name"`
	Resource    string              `pulumi:"resource" provider:"replaceOnChanges"`
	Permissions []APIPermissionArgs `pulumi:"permissions,optional"`
	CimdAccess  *APICimdAccessArgs  `pulumi:"cimdAccess,optional"`
}

// APIState is the persisted state of the API resource.
type APIState struct {
	Name        string               `pulumi:"name"`
	Resource    string               `pulumi:"resource" provider:"replaceOnChanges"`
	Permissions []APIPermissionState `pulumi:"permissions"`
	CimdAccess  *APICimdAccessArgs   `pulumi:"cimdAccess,optional"`
	CreatedAt   string               `pulumi:"createdAt"`
}

var _ infer.Annotated = (*APIPermissionArgs)(nil)

// Annotate provides schema descriptions for APIPermissionArgs.
func (a *APIPermissionArgs) Annotate(an infer.Annotator) {
	an.SetToken("index", "ApiPermissionArgs")
	an.Describe(&a.Key, "The stable identifier of the permission, e.g. \"read:orders\". Unique within the API.")
	an.Describe(&a.Name, "The display name of the permission.")
	an.Describe(&a.Description, "A description of the permission.")
}

var _ infer.Annotated = (*APIPermissionState)(nil)

// Annotate provides schema descriptions for APIPermissionState.
func (a *APIPermissionState) Annotate(an infer.Annotator) {
	an.SetToken("index", "ApiPermissionState")
	an.Describe(&a.ID, "The Pocket-ID identifier of the permission.")
}

var _ infer.Annotated = (*APIArgs)(nil)

// Annotate provides schema descriptions for APIArgs.
func (a *APIArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Name, "The display name of the API.")
	an.Describe(&a.Resource, "The resource identifier (audience) of the API, usually a URL. Changing it replaces the API.")
	an.Describe(&a.Permissions, "The permissions of the API. The list is authoritative: permissions not listed here are "+
		"removed.")
	an.Describe(&a.CimdAccess, "The access granted to clients registered through a Client ID Metadata Document. "+
		"When set it is authoritative; when omitted the server value is left untouched.")
}

var _ infer.Annotated = (*APIState)(nil)

// Annotate provides schema descriptions for APIState.
func (a *APIState) Annotate(an infer.Annotator) {
	an.Describe(&a.Name, "The display name of the API.")
	an.Describe(&a.Resource, "The resource identifier (audience) of the API.")
	an.Describe(&a.Permissions, "The permissions of the API, including their Pocket-ID identifiers.")
	an.Describe(&a.CimdAccess, "The access granted to clients registered through a Client ID Metadata Document, "+
		"if managed.")
	an.Describe(&a.CreatedAt, "The RFC 3339 timestamp at which the API was created.")
}

var _ infer.Annotated = (*API)(nil)

// Annotate describes the API resource.
func (API) Annotate(an infer.Annotator) {
	an.SetToken("index", "Api")
	an.Describe(&API{}, "A Pocket-ID API (OAuth resource server) and its permissions.")
}

func (API) Create(ctx context.Context, req infer.CreateRequest[APIArgs]) (infer.CreateResponse[APIState], error) {
	if req.DryRun {
		if err := validateCimd(req.Inputs); err != nil {
			return infer.CreateResponse[APIState]{}, err
		}
		return infer.CreateResponse[APIState]{ID: "preview", Output: previewAPIState(req.Inputs)}, nil
	}
	if err := validateCimd(req.Inputs); err != nil {
		return infer.CreateResponse[APIState]{}, err
	}
	c := clientFromContext(ctx)
	api, err := c.CreateAPI(ctx, req.Inputs.Name, req.Inputs.Resource)
	if err != nil {
		return infer.CreateResponse[APIState]{}, err
	}
	if len(req.Inputs.Permissions) > 0 {
		api, err = c.SetAPIPermissions(ctx, api.ID, permissionInputs(req.Inputs.Permissions))
		if err != nil {
			_ = c.DeleteAPI(ctx, api.ID) // best effort: don't leave a half-configured API behind
			return infer.CreateResponse[APIState]{}, err
		}
	}
	out := apiStateFrom(api)
	if req.Inputs.CimdAccess != nil {
		if err = applyCimd(ctx, c, api.ID, *req.Inputs.CimdAccess); err != nil {
			_ = c.DeleteAPI(ctx, api.ID)
			return infer.CreateResponse[APIState]{}, err
		}
		out.CimdAccess = req.Inputs.CimdAccess
	}
	return infer.CreateResponse[APIState]{ID: api.ID, Output: out}, nil
}

func (API) Read(
	ctx context.Context, req infer.ReadRequest[APIArgs, APIState],
) (infer.ReadResponse[APIArgs, APIState], error) {
	api, err := clientFromContext(ctx).GetAPI(ctx, req.ID)
	if isGone(err) {
		return infer.ReadResponse[APIArgs, APIState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[APIArgs, APIState]{}, err
	}
	state := apiStateFrom(api)
	// Only read the CIMD access when it is managed, or on import (no prior inputs) when it differs from the default.
	if req.State.CimdAccess != nil || (req.Inputs.Name == "" && (api.AllowCimdClients || cimdKeys(api) != nil)) {
		state.CimdAccess = &APICimdAccessArgs{Enabled: api.AllowCimdClients, PermissionKeys: cimdKeys(api)}
	}
	return infer.ReadResponse[APIArgs, APIState]{ID: api.ID, Inputs: apiArgsFrom(state), State: state}, nil
}

func (API) Diff(_ context.Context, req infer.DiffRequest[APIArgs, APIState]) (p.DiffResponse, error) {
	diff := map[string]p.PropertyDiff{}
	if req.Inputs.Name != req.State.Name {
		diff["name"] = p.PropertyDiff{Kind: p.Update}
	}
	if req.Inputs.Resource != req.State.Resource {
		diff["resource"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	if !samePermissions(req.Inputs.Permissions, req.State.Permissions) {
		diff["permissions"] = p.PropertyDiff{Kind: p.Update}
	}
	if !sameCimd(req.Inputs.CimdAccess, req.State.CimdAccess) {
		diff["cimdAccess"] = p.PropertyDiff{Kind: p.Update}
	}
	return p.DiffResponse{HasChanges: len(diff) > 0, DetailedDiff: diff}, nil
}

func (API) Update(
	ctx context.Context, req infer.UpdateRequest[APIArgs, APIState],
) (infer.UpdateResponse[APIState], error) {
	if err := validateCimd(req.Inputs); err != nil {
		return infer.UpdateResponse[APIState]{}, err
	}
	if req.DryRun {
		out := previewAPIState(req.Inputs)
		out.CreatedAt = req.State.CreatedAt
		return infer.UpdateResponse[APIState]{Output: out}, nil
	}
	c := clientFromContext(ctx)
	out := req.State
	if req.Inputs.Name != req.State.Name {
		api, err := c.UpdateAPI(ctx, req.ID, req.Inputs.Name)
		if err != nil {
			return infer.UpdateResponse[APIState]{}, err
		}
		out.Name = api.Name
	}
	permsChanged := !samePermissions(req.Inputs.Permissions, req.State.Permissions)
	if permsChanged {
		api, err := c.SetAPIPermissions(ctx, req.ID, permissionInputs(req.Inputs.Permissions))
		if err != nil {
			return infer.UpdateResponse[APIState]{}, err
		}
		out.Permissions = apiStateFrom(api).Permissions
	}
	// Re-apply after a permission change: the server may have dropped removed permissions from the CIMD set.
	if req.Inputs.CimdAccess != nil && (!sameCimd(req.Inputs.CimdAccess, req.State.CimdAccess) || permsChanged) {
		if err := applyCimd(ctx, c, req.ID, *req.Inputs.CimdAccess); err != nil {
			return infer.UpdateResponse[APIState]{}, err
		}
	}
	out.CimdAccess = req.Inputs.CimdAccess
	return infer.UpdateResponse[APIState]{Output: out}, nil
}

func (API) Delete(ctx context.Context, req infer.DeleteRequest[APIState]) (infer.DeleteResponse, error) {
	if err := clientFromContext(ctx).DeleteAPI(ctx, req.ID); err != nil && !isGone(err) {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

func permissionInputs(in []APIPermissionArgs) []pocketidclient.APIPermissionInput {
	out := make([]pocketidclient.APIPermissionInput, 0, len(in))
	for _, a := range in {
		out = append(out, pocketidclient.APIPermissionInput{Key: a.Key, Name: a.Name, Description: apiDeref(a.Description)})
	}
	return out
}

func apiStateFrom(api *pocketidclient.API) APIState {
	perms := make([]APIPermissionState, 0, len(api.Permissions))
	for _, pm := range api.Permissions {
		perms = append(perms, APIPermissionState{
			APIPermissionArgs: APIPermissionArgs{Key: pm.Key, Name: pm.Name, Description: apiNilIfEmpty(pm.Description)},
			ID:                pm.ID,
		})
	}
	return APIState{Name: api.Name, Resource: api.Resource, Permissions: perms, CreatedAt: api.CreatedAt}
}

func apiArgsFrom(s APIState) APIArgs {
	args := APIArgs{Name: s.Name, Resource: s.Resource, CimdAccess: s.CimdAccess}
	for _, pm := range s.Permissions {
		args.Permissions = append(args.Permissions, pm.APIPermissionArgs)
	}
	return args
}

// previewAPIState is the best-known state during preview: permission IDs are unknown.
func previewAPIState(in APIArgs) APIState {
	out := APIState{Name: in.Name, Resource: in.Resource, Permissions: []APIPermissionState{}, CimdAccess: in.CimdAccess}
	for _, pm := range in.Permissions {
		out.Permissions = append(out.Permissions, APIPermissionState{APIPermissionArgs: pm})
	}
	return out
}

// samePermissions compares permission sets by key, ignoring order and IDs.
func samePermissions(in []APIPermissionArgs, state []APIPermissionState) bool {
	if len(in) != len(state) {
		return false
	}
	norm := func(a APIPermissionArgs) APIPermissionArgs {
		return APIPermissionArgs{a.Key, a.Name, apiNilIfEmpty(apiDeref(a.Description))}
	}
	want := map[string]APIPermissionArgs{}
	for _, a := range in {
		want[a.Key] = norm(a)
	}
	for _, s := range state {
		w, ok := want[s.Key]
		g := norm(s.APIPermissionArgs)
		if !ok || w.Name != g.Name || apiDeref(w.Description) != apiDeref(g.Description) {
			return false
		}
	}
	return len(want) == len(in)
}

// validateCimd checks that every CIMD permission key is declared in the permissions of the API.
func validateCimd(in APIArgs) error {
	if in.CimdAccess == nil {
		return nil
	}
	declared := map[string]bool{}
	for _, pm := range in.Permissions {
		declared[pm.Key] = true
	}
	for _, k := range in.CimdAccess.PermissionKeys {
		if !declared[k] {
			return fmt.Errorf("cimdAccess.permissionKeys: permission %q is not declared in permissions", k)
		}
	}
	return nil
}

// applyCimd resolves permission keys to IDs (from the current server state) and replaces the CIMD access.
func applyCimd(ctx context.Context, c *pocketidclient.Client, id string, in APICimdAccessArgs) error {
	api, err := c.GetAPI(ctx, id)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(in.PermissionKeys))
	for _, k := range in.PermissionKeys {
		i := slices.IndexFunc(api.Permissions, func(pm pocketidclient.APIPermission) bool { return pm.Key == k })
		if i < 0 {
			return fmt.Errorf("cimdAccess.permissionKeys: permission %q does not exist on the API", k)
		}
		ids = append(ids, api.Permissions[i].ID)
	}
	return c.SetAPICimdAccess(ctx, id, in.Enabled, ids)
}

// cimdKeys returns the keys of the permissions open to CIMD clients (nil if none).
func cimdKeys(api *pocketidclient.API) []string {
	var keys []string
	for _, pm := range api.Permissions {
		if pm.AllowedForCimdClients {
			keys = append(keys, pm.Key)
		}
	}
	return keys
}

// sameCimd compares CIMD access by value, ignoring key order and duplicates.
func sameCimd(a, b *APICimdAccessArgs) bool {
	if a == nil || b == nil {
		return a == b
	}
	norm := func(k []string) []string {
		k = slices.Clone(k)
		slices.Sort(k)
		return slices.Compact(k)
	}
	return a.Enabled == b.Enabled && slices.Equal(norm(a.PermissionKeys), norm(b.PermissionKeys))
}

func apiDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func apiNilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

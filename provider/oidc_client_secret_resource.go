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
)

func init() { registerResource(infer.Resource(&OidcClientSecret{})) }

// OidcClientSecret generates a secret for an OIDC client. Several secrets can
// coexist on one client, which allows rotation without downtime. The resource
// ID is "<clientId>/<secretId>".
type OidcClientSecret struct{}

// OidcClientSecretArgs are the inputs of the OidcClientSecret resource.
type OidcClientSecretArgs struct {
	ClientID  string  `pulumi:"clientId" provider:"replaceOnChanges"`
	ExpiresAt *string `pulumi:"expiresAt,optional" provider:"replaceOnChanges"`
}

// OidcClientSecretState is persisted in the Pulumi state.
type OidcClientSecretState struct {
	OidcClientSecretArgs
	SecretID  string `pulumi:"secretId"`
	Secret    string `pulumi:"secret" provider:"secret"`
	Prefix    string `pulumi:"prefix"`
	CreatedAt string `pulumi:"createdAt"`
}

var _ infer.Annotated = (*OidcClientSecret)(nil)

// Annotate describes the resource.
func (*OidcClientSecret) Annotate(a infer.Annotator) {
	a.Describe(&OidcClientSecret{}, "A client secret of an OIDC client. Several secrets can coexist, "+
		"which allows rotating a secret without downtime: create the new one, then remove the old one. "+
		"Every property is immutable: changing one replaces the secret.")
}

var _ infer.Annotated = (*OidcClientSecretArgs)(nil)

// Annotate describes the inputs.
func (a *OidcClientSecretArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ClientID, "The ID of the OIDC client the secret belongs to. Changing it replaces the secret.")
	an.Describe(&a.ExpiresAt, "The expiration date of the secret (RFC 3339). Never expires when unset. "+
		"Changing it replaces the secret.")
}

var _ infer.Annotated = (*OidcClientSecretState)(nil)

// Annotate describes the outputs.
func (s *OidcClientSecretState) Annotate(a infer.Annotator) {
	a.Describe(&s.SecretID, "The ID of the secret.")
	a.Describe(&s.Secret, "The plain-text secret. Only returned by Pocket-ID at creation, so it is kept "+
		"from the state and is empty after an import.")
	a.Describe(&s.Prefix, "The first characters of the secret, as reported by Pocket-ID.")
	a.Describe(&s.CreatedAt, "The creation date of the secret.")
}

// Create generates the secret.
func (*OidcClientSecret) Create(
	ctx context.Context, req infer.CreateRequest[OidcClientSecretArgs],
) (infer.CreateResponse[OidcClientSecretState], error) {
	if req.DryRun {
		return infer.CreateResponse[OidcClientSecretState]{
			ID: req.Inputs.ClientID + "/" + req.Name, Output: OidcClientSecretState{OidcClientSecretArgs: req.Inputs},
		}, nil
	}
	got, err := clientFromContext(ctx).CreateOidcClientSecret(ctx, req.Inputs.ClientID, oidcDeref(req.Inputs.ExpiresAt))
	if err != nil {
		return infer.CreateResponse[OidcClientSecretState]{}, err
	}
	state := OidcClientSecretState{
		OidcClientSecretArgs: req.Inputs, SecretID: got.ID, Secret: got.Secret, Prefix: got.Prefix, CreatedAt: got.CreatedAt,
	}
	return infer.CreateResponse[OidcClientSecretState]{ID: req.Inputs.ClientID + "/" + got.ID, Output: state}, nil
}

// Read looks the secret up in the client's secret list. The plain-text value is carried over from the state.
func (*OidcClientSecret) Read(
	ctx context.Context, req infer.ReadRequest[OidcClientSecretArgs, OidcClientSecretState],
) (infer.ReadResponse[OidcClientSecretArgs, OidcClientSecretState], error) {
	none := infer.ReadResponse[OidcClientSecretArgs, OidcClientSecretState]{}
	clientID, secretID, ok := strings.Cut(req.ID, "/")
	if !ok || clientID == "" || secretID == "" {
		return none, fmt.Errorf("invalid ID %q, expected <clientId>/<secretId>", req.ID)
	}
	secrets, err := clientFromContext(ctx).ListOidcClientSecrets(ctx, clientID)
	if isGone(err) {
		return none, nil
	}
	if err != nil {
		return none, err
	}
	for _, s := range secrets {
		if s.ID != secretID {
			continue
		}
		args := OidcClientSecretArgs{ClientID: clientID, ExpiresAt: req.State.ExpiresAt}
		if req.State.ClientID == "" {
			args.ExpiresAt = oidcNonEmpty(s.ExpiresAt)
		}
		state := OidcClientSecretState{
			OidcClientSecretArgs: args, SecretID: s.ID, Secret: req.State.Secret, Prefix: s.Prefix, CreatedAt: s.CreatedAt,
		}
		return infer.ReadResponse[OidcClientSecretArgs, OidcClientSecretState]{ID: req.ID, Inputs: args, State: state}, nil
	}
	return none, nil
}

// Delete revokes the secret.
func (*OidcClientSecret) Delete(
	ctx context.Context, req infer.DeleteRequest[OidcClientSecretState],
) (infer.DeleteResponse, error) {
	err := clientFromContext(ctx).DeleteOidcClientSecret(ctx, req.State.ClientID, req.State.SecretID)
	if err != nil && !isGone(err) {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}

// Copyright 2025, axnic.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package pocketidclient

import (
	"context"
	"net/http"
)

// OpenIDConfiguration is the OpenID Connect discovery document served by Pocket-ID.
type OpenIDConfiguration struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	DeviceAuthorizationEndpoint       string   `json:"device_authorization_endpoint"`
	IntrospectionEndpoint             string   `json:"introspection_endpoint"`
	EndSessionEndpoint                string   `json:"end_session_endpoint"`
	JwksURI                           string   `json:"jwks_uri"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// GetOpenIDConfiguration fetches /.well-known/openid-configuration. The
// endpoint is public; the client still sends its API key, which is ignored.
func (c *Client) GetOpenIDConfiguration(ctx context.Context) (*OpenIDConfiguration, error) {
	var out OpenIDConfiguration
	if err := c.Do(ctx, http.MethodGet, "/.well-known/openid-configuration", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

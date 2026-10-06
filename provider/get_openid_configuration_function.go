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
package provider

import (
	"context"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// GetOpenIDConfiguration reads the OpenID Connect discovery document of the
// instance (/.well-known/openid-configuration).
type GetOpenIDConfiguration struct{}

// GetOpenIDConfigurationArgs has no input: the document describes the
// configured instance.
type GetOpenIDConfigurationArgs struct{}

// GetOpenIDConfigurationResult is the discovery document.
type GetOpenIDConfigurationResult struct {
	Issuer                            string   `pulumi:"issuer"`
	AuthorizationEndpoint             string   `pulumi:"authorizationEndpoint"`
	TokenEndpoint                     string   `pulumi:"tokenEndpoint"`
	UserinfoEndpoint                  string   `pulumi:"userinfoEndpoint"`
	DeviceAuthorizationEndpoint       string   `pulumi:"deviceAuthorizationEndpoint"`
	IntrospectionEndpoint             string   `pulumi:"introspectionEndpoint"`
	EndSessionEndpoint                string   `pulumi:"endSessionEndpoint"`
	JwksURI                           string   `pulumi:"jwksUri"`
	GrantTypesSupported               []string `pulumi:"grantTypesSupported"`
	ScopesSupported                   []string `pulumi:"scopesSupported"`
	ClaimsSupported                   []string `pulumi:"claimsSupported"`
	ResponseTypesSupported            []string `pulumi:"responseTypesSupported"`
	SubjectTypesSupported             []string `pulumi:"subjectTypesSupported"`
	IDTokenSigningAlgValuesSupported  []string `pulumi:"idTokenSigningAlgValuesSupported"`
	CodeChallengeMethodsSupported     []string `pulumi:"codeChallengeMethodsSupported"`
	TokenEndpointAuthMethodsSupported []string `pulumi:"tokenEndpointAuthMethodsSupported"`
}

var _ infer.Annotated = (*GetOpenIDConfiguration)(nil)

// Annotate describes the function.
func (f *GetOpenIDConfiguration) Annotate(a infer.Annotator) {
	a.SetToken("index", "getOpenIdConfiguration")
	a.Describe(f, "Reads the OpenID Connect discovery document of the Pocket-ID instance: "+
		"issuer, endpoints, JWKS URI and supported capabilities. Use it to configure applications that consume the "+
		"instance as an OIDC provider.")
}

var _ infer.Annotated = (*GetOpenIDConfigurationResult)(nil)

// Annotate provides schema descriptions for the result fields.
func (r *GetOpenIDConfigurationResult) Annotate(a infer.Annotator) {
	a.Describe(&r.Issuer, "The issuer URL, i.e. the base URL of the instance.")
	a.Describe(&r.AuthorizationEndpoint, "The OAuth 2.0 authorization endpoint.")
	a.Describe(&r.TokenEndpoint, "The OAuth 2.0 token endpoint.")
	a.Describe(&r.UserinfoEndpoint, "The OIDC userinfo endpoint.")
	a.Describe(&r.DeviceAuthorizationEndpoint, "The device authorization endpoint, empty if the instance does not "+
		"advertise one.")
	a.Describe(&r.IntrospectionEndpoint, "The token introspection endpoint, empty if the instance does not advertise one.")
	a.Describe(&r.EndSessionEndpoint, "The RP-initiated logout endpoint, empty if the instance does not advertise one.")
	a.Describe(&r.JwksURI, "The URL of the JSON Web Key Set used to verify issued tokens.")
	a.Describe(&r.GrantTypesSupported, "The supported OAuth 2.0 grant types.")
	a.Describe(&r.ScopesSupported, "The supported scopes.")
	a.Describe(&r.ClaimsSupported, "The supported claims.")
	a.Describe(&r.ResponseTypesSupported, "The supported response types.")
	a.Describe(&r.SubjectTypesSupported, "The supported subject identifier types.")
	a.Describe(&r.IDTokenSigningAlgValuesSupported, "The supported ID token signing algorithms.")
	a.Describe(&r.CodeChallengeMethodsSupported, "The supported PKCE code challenge methods.")
	a.Describe(&r.TokenEndpointAuthMethodsSupported, "The supported client authentication methods at the token endpoint.")
}

// Invoke fetches the discovery document.
func (GetOpenIDConfiguration) Invoke(
	ctx context.Context, _ infer.FunctionRequest[GetOpenIDConfigurationArgs],
) (infer.FunctionResponse[GetOpenIDConfigurationResult], error) {
	d, err := clientFromContext(ctx).GetOpenIDConfiguration(ctx)
	if err != nil {
		return infer.FunctionResponse[GetOpenIDConfigurationResult]{}, err
	}
	return infer.FunctionResponse[GetOpenIDConfigurationResult]{Output: GetOpenIDConfigurationResult{
		Issuer: d.Issuer, AuthorizationEndpoint: d.AuthorizationEndpoint, TokenEndpoint: d.TokenEndpoint,
		UserinfoEndpoint: d.UserinfoEndpoint, DeviceAuthorizationEndpoint: d.DeviceAuthorizationEndpoint,
		IntrospectionEndpoint: d.IntrospectionEndpoint, EndSessionEndpoint: d.EndSessionEndpoint, JwksURI: d.JwksURI,
		GrantTypesSupported: nonNil(d.GrantTypesSupported), ScopesSupported: nonNil(d.ScopesSupported),
		ClaimsSupported:        nonNil(d.ClaimsSupported),
		ResponseTypesSupported: nonNil(d.ResponseTypesSupported), SubjectTypesSupported: nonNil(d.SubjectTypesSupported),
		IDTokenSigningAlgValuesSupported:  nonNil(d.IDTokenSigningAlgValuesSupported),
		CodeChallengeMethodsSupported:     nonNil(d.CodeChallengeMethodsSupported),
		TokenEndpointAuthMethodsSupported: nonNil(d.TokenEndpointAuthMethodsSupported),
	}}, nil
}

func init() { registerFunction(infer.Function(&GetOpenIDConfiguration{})) }

// nonNil turns a missing list into an empty one so outputs are never null.
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

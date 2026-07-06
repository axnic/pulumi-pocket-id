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

// Package provider implements the Pulumi provider for Pocket-ID.
package provider

import (
	"fmt"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

// Version is initialized by the Go linker to contain the semver of this build.
var Version string

// Name controls how this provider is referenced in package names and elsewhere.
const Name string = "pocket-id"

// Provider creates a new instance of the Pocket-ID provider.
func Provider() p.Provider {
	prov, err := infer.NewProviderBuilder().
		WithDisplayName("pocket-id").
		WithDescription("A Pulumi provider for Pocket-ID, a passkey-only OIDC provider.").
		WithHomepage("https://github.com/pocket-id/pocket-id").
		WithNamespace("pocketid").
		WithResources(
			infer.Resource(&OidcClient{}),
			infer.Resource(&User{}),
			infer.Resource(&UserGroup{}),
			infer.Resource(&CustomClaims{}),
		).
		WithConfig(infer.Config(&Config{})).
		WithModuleMap(map[tokens.ModuleName]tokens.ModuleName{
			"provider": "index",
		}).Build()
	if err != nil {
		panic(fmt.Errorf("unable to build provider: %w", err))
	}
	return prov
}

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
	"os"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/axnic/pulumi-pocket-id/provider/internal/pocketidclient"
)

// Environment variables used as a fallback when the matching config field is unset.
const (
	envBaseURL = "POCKET_ID_BASE_URL"
	envAPIKey  = "POCKET_ID_API_KEY" //nolint:gosec // env var name, not a credential
)

// Config holds the provider-level configuration for the Pocket-ID provider.
type Config struct {
	// BaseURL is the URL of the Pocket-ID server, e.g. "https://pocket-id.example.com".
	BaseURL string `pulumi:"baseUrl,optional"`
	// APIKey is a Pocket-ID API key (or the server's STATIC_API_KEY), sent in the `X-API-Key` header.
	APIKey string `pulumi:"apiKey,optional" provider:"secret"`

	client *pocketidclient.Client
}

var _ infer.CustomConfigure = (*Config)(nil)
var _ infer.Annotated = (*Config)(nil)

// Annotate provides schema descriptions for the provider configuration.
func (c *Config) Annotate(a infer.Annotator) {
	a.Describe(&c.BaseURL, "The base URL of the Pocket-ID server, e.g. \"https://pocket-id.example.com\". "+
		"Falls back to the "+envBaseURL+" environment variable if not set.")
	a.Describe(&c.APIKey, "A Pocket-ID API key sent with every request in the X-API-Key header. "+
		"Falls back to the "+envAPIKey+" environment variable if not set.")
}

// Configure applies the environment fallbacks and builds the shared API client.
func (c *Config) Configure(_ context.Context) error {
	if c.BaseURL == "" {
		c.BaseURL = os.Getenv(envBaseURL)
	}
	if c.APIKey == "" {
		c.APIKey = os.Getenv(envAPIKey)
	}
	if c.BaseURL == "" {
		return fmt.Errorf("pocket-id provider: 'baseUrl' must be set, or the %s environment variable", envBaseURL)
	}
	if c.APIKey == "" {
		return fmt.Errorf("pocket-id provider: 'apiKey' must be set, or the %s environment variable", envAPIKey)
	}
	c.client = pocketidclient.New(c.BaseURL, c.APIKey, nil)
	return nil
}

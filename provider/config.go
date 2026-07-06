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

// Config holds the provider-level configuration for the Pocket-ID provider.
type Config struct {
	// BaseUrl is the URL of the Pocket-ID server, e.g. "https://pocket-id.example.com".
	BaseUrl string `pulumi:"baseUrl" json:"baseUrl"`

	// ApiKey is a Pocket-ID API key used to authenticate against the REST API.
	// It is sent with every request via the `X-API-Key` header.
	ApiKey string `pulumi:"apiKey" provider:"secret" json:"apiKey"`
}

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

package tests

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

func TestConfigureRequiresBaseUrl(t *testing.T) {
	t.Parallel()
	prov := testServer(t)
	err := prov.Configure(p.ConfigureRequest{
		Args: property.NewMap(map[string]property.Value{
			"apiKey": property.New("k"),
		}),
	})
	require.Error(t, err)
}

func TestConfigureRequiresApiKey(t *testing.T) {
	t.Parallel()
	prov := testServer(t)
	err := prov.Configure(p.ConfigureRequest{
		Args: property.NewMap(map[string]property.Value{
			"baseUrl": property.New("http://localhost:1411"),
		}),
	})
	require.Error(t, err)
}

func TestWrongApiKeyIsRejected(t *testing.T) {
	t.Parallel()
	_, srv := newFakeServer(t, "correct-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, "wrong-key")

	_, err := prov.Create(p.CreateRequest{
		Urn: urn("UserGroup"),
		Properties: property.NewMap(map[string]property.Value{
			"friendlyName": property.New("Engineers"),
			"name":         property.New("engineers"),
		}),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestReadMissingReturnsError(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)

	_, err := prov.Read(p.ReadRequest{
		ID:  "grp-does-not-exist",
		Urn: urn("UserGroup"),
		Properties: property.NewMap(map[string]property.Value{
			"friendlyName": property.New("x"),
			"name":         property.New("x"),
		}),
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found"),
		"expected 404/not-found error, got: %v", err)
}

func TestBaseURLTrailingSlashTrimmed(t *testing.T) {
	t.Parallel()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	// A trailing slash must not break request paths.
	configure(t, prov, srv.URL+"/", fake.apiKey)

	_, err := prov.Create(p.CreateRequest{
		Urn: urn("UserGroup"),
		Properties: property.NewMap(map[string]property.Value{
			"friendlyName": property.New("Engineers"),
			"name":         property.New("engineers"),
		}),
	})
	require.NoError(t, err)
}

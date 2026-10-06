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
	"testing"

	"github.com/stretchr/testify/require"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// identityEnv is an integration provider wired to a fresh fake Pocket-ID.
func identityEnv(t *testing.T) integration.Server {
	t.Helper()
	fake, srv := newFakeServer(t, "test-key")
	prov := testServer(t)
	configure(t, prov, srv.URL, fake.apiKey)
	return prov
}

// identityOffline is a provider whose API is unreachable: any API call fails.
func identityOffline(t *testing.T) integration.Server {
	t.Helper()
	prov := testServer(t)
	configure(t, prov, "http://127.0.0.1:1", "test-key")
	return prov
}

func idStrings(ss ...string) property.Value {
	vals := make([]property.Value, len(ss))
	for i, s := range ss {
		vals[i] = property.New(s)
	}
	return property.New(property.NewArray(vals))
}

func idStringMap(m map[string]string) property.Value {
	vals := make(map[string]property.Value, len(m))
	for k, v := range m {
		vals[k] = property.New(v)
	}
	return property.New(property.NewMap(vals))
}

func idStringSlice(v property.Value) []string {
	out := []string{}
	for _, e := range v.AsArray().AsSlice() {
		out = append(out, e.AsString())
	}
	return out
}

// idCreate creates a resource and returns its ID and outputs.
func idCreate(t *testing.T, prov integration.Server, typ string, props map[string]property.Value) (string,
	property.Map) {
	t.Helper()
	resp, err := prov.Create(p.CreateRequest{Urn: urn(typ), Properties: property.NewMap(props)})
	require.NoError(t, err)
	return resp.ID, resp.Properties
}

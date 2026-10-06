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
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

func TestConfigureRequiresBaseUrl(t *testing.T) {
	t.Parallel()
	prov := testServer(t)
	err := prov.Configure(p.ConfigureRequest{
		Args: property.NewMap(map[string]property.Value{
			keyAPIKey: property.New("k"),
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

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

// appConfigVariable is one entry of the application configuration. Pocket-ID
// stores every value as a string, whatever its logical type.
type appConfigVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// GetAppConfig returns every application configuration variable, including
// the private ones, as a key -> raw string value map.
func (c *Client) GetAppConfig(ctx context.Context) (map[string]string, error) {
	var vars []appConfigVariable
	if err := c.Do(ctx, http.MethodGet, "/api/application-configuration/all", nil, nil, &vars); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(vars))
	for _, v := range vars {
		out[v.Key] = v.Value
	}
	return out, nil
}

// UpdateAppConfig replaces the application configuration with values. The API
// requires the complete set of mandatory keys, so callers merge their changes
// over the result of GetAppConfig first.
func (c *Client) UpdateAppConfig(ctx context.Context, values map[string]string) error {
	return c.Do(ctx, http.MethodPut, "/api/application-configuration", nil, values, nil)
}

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
	"sort"

	"github.com/axnic/pulumi-pocket-id/provider/internal/pocketidclient"
)

// claimsToList turns a claims map into the API list, sorted by key.
func claimsToList(m map[string]string) []pocketidclient.CustomClaim {
	list := make([]pocketidclient.CustomClaim, 0, len(m))
	for k, v := range m {
		list = append(list, pocketidclient.CustomClaim{Key: k, Value: v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Key < list[j].Key })
	return list
}

// claimsToMap turns the API claim list into a map.
func claimsToMap(list []pocketidclient.CustomClaim) map[string]string {
	m := make(map[string]string, len(list))
	for _, c := range list {
		m[c.Key] = c.Value
	}
	return m
}
